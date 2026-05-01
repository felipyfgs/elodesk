package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend/internal/model"
	"backend/internal/repo"
)

var (
	ErrMacroInvalidAction = errors.New("invalid macro action")
	ErrMacroNotFound      = errors.New("macro not found")
	ErrMacroActionFailed  = errors.New("macro action failed")
)

var allowedMacroActions = map[string]bool{
	"assign_agent":  true,
	"assign_team":   true,
	"add_label":     true,
	"remove_label":  true,
	"change_status": true,
	"snooze_until":  true,
	"send_message":  true,
	"add_note":      true,
}

type MacroAction struct {
	Name   string          `json:"name"`
	Params json.RawMessage `json:"params,omitempty"`
}

type MacroService struct {
	macroRepo *repo.MacroRepo
	pool      *pgxpool.Pool
}

func NewMacroService(macroRepo *repo.MacroRepo, pool *pgxpool.Pool) *MacroService {
	return &MacroService{macroRepo: macroRepo, pool: pool}
}

func validateMacroActions(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var actions []MacroAction
	if err := json.Unmarshal(raw, &actions); err != nil {
		return fmt.Errorf("%w: actions must be a JSON array", ErrMacroInvalidAction)
	}
	for i, a := range actions {
		if !allowedMacroActions[a.Name] {
			return fmt.Errorf("%w: action[%d] '%s' not allowed", ErrMacroInvalidAction, i, a.Name)
		}
	}
	return nil
}

func validateMacroConditions(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("conditions must be valid JSON: %w", err)
	}
	return nil
}

func (s *MacroService) Create(ctx context.Context, accountID, userID int64, name, visibility string, conditions, actions json.RawMessage) (*model.Macro, error) {
	if err := validateMacroActions(actions); err != nil {
		return nil, err
	}
	if err := validateMacroConditions(conditions); err != nil {
		return nil, err
	}

	condStr := "{}"
	if len(conditions) > 0 {
		condStr = string(conditions)
	}
	actStr := "[]"
	if len(actions) > 0 {
		actStr = string(actions)
	}

	m := &model.Macro{
		AccountID:  accountID,
		Name:       name,
		Visibility: visibility,
		Conditions: condStr,
		Actions:    actStr,
		CreatedBy:  userID,
	}
	if err := s.macroRepo.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("macro.create: %w", err)
	}
	return m, nil
}

func (s *MacroService) List(ctx context.Context, accountID int64) ([]model.Macro, error) {
	return s.macroRepo.ListByAccount(ctx, accountID)
}

func (s *MacroService) Get(ctx context.Context, accountID, id int64) (*model.Macro, error) {
	return s.macroRepo.FindByID(ctx, id, accountID)
}

type UpdateMacroInput struct {
	Name       *string
	Visibility *string
	Conditions json.RawMessage
	Actions    json.RawMessage
}

func (s *MacroService) Update(ctx context.Context, accountID, id int64, in UpdateMacroInput) (*model.Macro, error) {
	current, err := s.macroRepo.FindByID(ctx, id, accountID)
	if err != nil {
		return nil, fmt.Errorf("macro.update: %w", err)
	}

	if in.Actions != nil {
		if err := validateMacroActions(in.Actions); err != nil {
			return nil, err
		}
		current.Actions = string(in.Actions)
	}
	if in.Conditions != nil {
		if err := validateMacroConditions(in.Conditions); err != nil {
			return nil, err
		}
		current.Conditions = string(in.Conditions)
	}
	if in.Name != nil {
		current.Name = *in.Name
	}
	if in.Visibility != nil {
		current.Visibility = *in.Visibility
	}

	if err := s.macroRepo.Update(ctx, current); err != nil {
		return nil, fmt.Errorf("macro.update: %w", err)
	}
	return current, nil
}

func (s *MacroService) Delete(ctx context.Context, accountID, id int64) error {
	return s.macroRepo.Delete(ctx, id, accountID)
}

type ApplyMacroResult struct {
	ExecutedActions int
	FailedIndex     int
}

// Apply executes macro actions on a conversation in a single database
// transaction. Each action is validated first, then applied against tx. If any
// fails the transaction rolls back and no state change is persisted.
func (s *MacroService) Apply(ctx context.Context, accountID, conversationID, macroID, userID int64) (*ApplyMacroResult, error) {
	macro, err := s.macroRepo.FindByID(ctx, macroID, accountID)
	if err != nil {
		return nil, fmt.Errorf("macro.apply: %w", err)
	}

	var actions []MacroAction
	if err := json.Unmarshal([]byte(macro.Actions), &actions); err != nil {
		return nil, fmt.Errorf("macro.apply: %w", err)
	}

	for i, a := range actions {
		if !allowedMacroActions[a.Name] {
			return &ApplyMacroResult{ExecutedActions: i, FailedIndex: i}, fmt.Errorf("%w: %s", ErrMacroInvalidAction, a.Name)
		}
	}

	var conv macroConversation
	row := s.pool.QueryRow(ctx,
		`SELECT id, account_id, inbox_id, contact_id FROM conversations WHERE id = $1 AND account_id = $2`,
		conversationID, accountID,
	)
	if err := row.Scan(&conv.ID, &conv.AccountID, &conv.InboxID, &conv.ContactID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: conversation not found", repo.ErrConversationNotFound)
		}
		return nil, fmt.Errorf("macro.apply load conv: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("macro.apply begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for i, a := range actions {
		if err := s.executeAction(ctx, tx, &conv, userID, a); err != nil {
			return &ApplyMacroResult{ExecutedActions: i, FailedIndex: i}, fmt.Errorf("%w at index %d: %w", ErrMacroActionFailed, i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("macro.apply commit: %w", err)
	}

	return &ApplyMacroResult{ExecutedActions: len(actions)}, nil
}

type macroConversation struct {
	ID        int64
	AccountID int64
	InboxID   int64
	ContactID int64
}

func (s *MacroService) executeAction(ctx context.Context, tx pgx.Tx, conv *macroConversation, userID int64, a MacroAction) error {
	switch a.Name {
	case "assign_agent":
		var p struct {
			AgentID *int64 `json:"agent_id"`
		}
		if err := unmarshalParams(a.Params, &p); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`UPDATE conversations SET assignee_id = $1, updated_at = NOW() WHERE id = $2 AND account_id = $3`,
			p.AgentID, conv.ID, conv.AccountID)
		return err
	case "assign_team":
		var p struct {
			TeamID *int64 `json:"team_id"`
		}
		if err := unmarshalParams(a.Params, &p); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`UPDATE conversations SET team_id = $1, updated_at = NOW() WHERE id = $2 AND account_id = $3`,
			p.TeamID, conv.ID, conv.AccountID)
		return err
	case "add_label":
		var p struct {
			LabelID int64 `json:"label_id"`
		}
		if err := unmarshalParams(a.Params, &p); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO label_taggings (account_id, label_id, taggable_type, taggable_id)
				VALUES ($1, $2, 'conversation', $3)
				ON CONFLICT (label_id, taggable_type, taggable_id) DO NOTHING`,
			conv.AccountID, p.LabelID, conv.ID)
		return err
	case "remove_label":
		var p struct {
			LabelID int64 `json:"label_id"`
		}
		if err := unmarshalParams(a.Params, &p); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`DELETE FROM label_taggings WHERE account_id = $1 AND label_id = $2 AND taggable_type = 'conversation' AND taggable_id = $3`,
			conv.AccountID, p.LabelID, conv.ID)
		return err
	case "change_status":
		var p struct {
			Status string `json:"status"`
		}
		if err := unmarshalParams(a.Params, &p); err != nil {
			return err
		}
		status, ok := parseMacroStatus(p.Status)
		if !ok {
			return fmt.Errorf("unknown status %q", p.Status)
		}
		_, err := tx.Exec(ctx,
			`UPDATE conversations SET status = $1, updated_at = NOW() WHERE id = $2 AND account_id = $3`,
			status, conv.ID, conv.AccountID)
		return err
	case "snooze_until":
		var p struct {
			Until string `json:"until"`
		}
		if err := unmarshalParams(a.Params, &p); err != nil {
			return err
		}
		until, err := time.Parse(time.RFC3339, p.Until)
		if err != nil {
			return fmt.Errorf("invalid snooze_until timestamp: %w", err)
		}
		attrs := fmt.Sprintf(`{"snoozed_until":%q}`, until.UTC().Format(time.RFC3339))
		_, err = tx.Exec(ctx,
			`UPDATE conversations SET status = $1, additional_attributes = COALESCE(additional_attributes::jsonb, '{}'::jsonb) || $2::jsonb, updated_at = NOW()
				WHERE id = $3 AND account_id = $4`,
			model.ConversationSnoozed, attrs, conv.ID, conv.AccountID)
		return err
	case "send_message":
		var p struct {
			Content string `json:"content"`
		}
		if err := unmarshalParams(a.Params, &p); err != nil {
			return err
		}
		if p.Content == "" {
			return fmt.Errorf("send_message requires content")
		}
		senderType := "User"
		_, err := tx.Exec(ctx,
			`INSERT INTO messages (account_id, inbox_id, conversation_id, message_type, content_type, content, private, status, sender_type, sender_id)
				VALUES ($1, $2, $3, $4, $5, $6, FALSE, $7, $8, $9)`,
			conv.AccountID, conv.InboxID, conv.ID, model.MessageOutgoing, model.ContentTypeText, p.Content, model.MessageSent, senderType, userID)
		return err
	case "add_note":
		var p struct {
			Content string `json:"content"`
		}
		if err := unmarshalParams(a.Params, &p); err != nil {
			return err
		}
		if p.Content == "" {
			return fmt.Errorf("add_note requires content")
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO notes (account_id, contact_id, user_id, content) VALUES ($1, $2, $3, $4)`,
			conv.AccountID, conv.ContactID, userID, p.Content)
		return err
	}
	return fmt.Errorf("%w: %s", ErrMacroInvalidAction, a.Name)
}

func unmarshalParams(raw json.RawMessage, dst any) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	return nil
}

func parseMacroStatus(s string) (model.ConversationStatus, bool) {
	switch s {
	case "open":
		return model.ConversationOpen, true
	case "resolved":
		return model.ConversationResolved, true
	case "pending":
		return model.ConversationPending, true
	case "snoozed":
		return model.ConversationSnoozed, true
	}
	return 0, false
}
