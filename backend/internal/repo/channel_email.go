package repo

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrChannelEmailNotFound = errors.New("channel email not found")

const channelEmailSelectColumns = `id, account_id, email, name, provider,
	imap_address, imap_port, imap_login, imap_password_ciphertext, imap_enable_ssl, imap_enabled, last_uid_seen,
	smtp_address, smtp_port, smtp_login, smtp_password_ciphertext, smtp_enable_ssl,
	provider_config, verified_for_sending, requires_reauth, created_at, updated_at`

type channelEmailScanner interface {
	Scan(dest ...any) error
}

func scanChannelEmail(s channelEmailScanner, m *model.ChannelEmail) error {
	return s.Scan(
		&m.ID, &m.AccountID, &m.Email, &m.Name, &m.Provider,
		&m.ImapAddress, &m.ImapPort, &m.ImapLogin, &m.ImapPasswordCiphertext, &m.ImapEnableSSL, &m.ImapEnabled, &m.LastUIDSeen,
		&m.SmtpAddress, &m.SmtpPort, &m.SmtpLogin, &m.SmtpPasswordCiphertext, &m.SmtpEnableSSL,
		&m.ProviderConfig, &m.VerifiedForSending, &m.RequiresReauth, &m.CreatedAt, &m.UpdatedAt,
	)
}

type ChannelEmailRepo struct {
	pool *pgxpool.Pool
}

func NewChannelEmailRepo(pool *pgxpool.Pool) *ChannelEmailRepo {
	return &ChannelEmailRepo{pool: pool}
}

func (r *ChannelEmailRepo) Create(ctx context.Context, m *model.ChannelEmail) error {
	query := `INSERT INTO channels_email
		(account_id, email, name, provider,
		 imap_address, imap_port, imap_login, imap_password_ciphertext, imap_enable_ssl, imap_enabled,
		 smtp_address, smtp_port, smtp_login, smtp_password_ciphertext, smtp_enable_ssl,
		 provider_config)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		RETURNING id, last_uid_seen, verified_for_sending, requires_reauth, created_at, updated_at`
	return r.pool.QueryRow(ctx, query,
		m.AccountID, m.Email, m.Name, m.Provider,
		m.ImapAddress, m.ImapPort, m.ImapLogin, m.ImapPasswordCiphertext, m.ImapEnableSSL, m.ImapEnabled,
		m.SmtpAddress, m.SmtpPort, m.SmtpLogin, m.SmtpPasswordCiphertext, m.SmtpEnableSSL,
		m.ProviderConfig,
	).Scan(&m.ID, &m.LastUIDSeen, &m.VerifiedForSending, &m.RequiresReauth, &m.CreatedAt, &m.UpdatedAt)
}

func (r *ChannelEmailRepo) FindByID(ctx context.Context, id, accountID int64) (*model.ChannelEmail, error) {
	query := `SELECT ` + channelEmailSelectColumns + ` FROM channels_email WHERE id = $1 AND account_id = $2`
	row := r.pool.QueryRow(ctx, query, id, accountID)
	var m model.ChannelEmail
	if err := scanChannelEmail(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelEmailNotFound, err)
		}
		return nil, fmt.Errorf("find channel email by id: %w", err)
	}
	return &m, nil
}

func (r *ChannelEmailRepo) FindByInboxID(ctx context.Context, inboxID int64) (*model.ChannelEmail, error) {
	query := `SELECT ce.` + channelEmailSelectColumns +
		` FROM channels_email ce JOIN inboxes i ON i.channel_id = ce.id
		  WHERE i.id = $1 AND i.channel_type = 'Channel::Email'`
	row := r.pool.QueryRow(ctx, query, inboxID)
	var m model.ChannelEmail
	if err := scanChannelEmail(row, &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: %w", ErrChannelEmailNotFound, err)
		}
		return nil, fmt.Errorf("find channel email by inbox id: %w", err)
	}
	return &m, nil
}

// ListImapEnabled returns all channels_email rows with imap_enabled=true for the poller scheduler.
func (r *ChannelEmailRepo) ListImapEnabled(ctx context.Context) ([]model.ChannelEmail, error) {
	query := `SELECT ` + channelEmailSelectColumns + ` FROM channels_email WHERE imap_enabled = true ORDER BY id`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list imap-enabled email channels: %w", err)
	}
	defer rows.Close()
	var out []model.ChannelEmail
	for rows.Next() {
		var m model.ChannelEmail
		if err := scanChannelEmail(rows, &m); err != nil {
			return nil, fmt.Errorf("scan channel email: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *ChannelEmailRepo) UpdateLastUIDSeen(ctx context.Context, id, uid int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_email SET last_uid_seen = $1, updated_at = NOW() WHERE id = $2`, uid, id)
	return err
}

func (r *ChannelEmailRepo) SetRequiresReauth(ctx context.Context, id int64, v bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_email SET requires_reauth = $1, updated_at = NOW() WHERE id = $2`, v, id)
	return err
}

func (r *ChannelEmailRepo) UpdateProviderConfig(ctx context.Context, id int64, configCiphertext string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels_email SET provider_config = $1, verified_for_sending = TRUE, updated_at = NOW() WHERE id = $2`,
		configCiphertext, id)
	return err
}
