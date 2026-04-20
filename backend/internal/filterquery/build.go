package filterquery

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var identRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,63}$`)

type Condition struct {
	AttributeKey   string          `json:"attribute_key"`
	FilterOperator string          `json:"filter_operator"`
	Value          json.RawMessage `json:"value"`
}

type FilterQuery struct {
	Operator   string      `json:"operator"`
	Conditions []Condition `json:"conditions"`
}

var standardConversationKeys = map[string]bool{
	"status": true, "priority": true, "assignee_id": true, "team_id": true,
	"contact_id": true, "inbox_id": true, "labels": true,
	"created_at": true, "updated_at": true, "last_activity_at": true,
}

var standardContactKeys = map[string]bool{
	"name": true, "email": true, "phone_number": true, "identifier": true,
	"blocked": true, "last_activity_at": true, "created_at": true, "updated_at": true,
}

var validOperators = map[string]bool{
	"equal_to": true, "not_equal_to": true, "contains": true, "starts_with": true,
	"greater_than": true, "less_than": true, "in": true, "between": true,
	"is_null": true, "is_not_null": true,
}

// BuildSQL compiles the user-supplied filter JSON into a parameterized SQL
// fragment. startArgN lets callers reserve leading placeholders (e.g. $1 for
// account_id) so the generated clause can be appended to an existing query
// without colliding with caller-owned arguments.
func BuildSQL(raw json.RawMessage, filterType string, customAttrs []string, startArgN int) (string, []any, error) {
	if startArgN < 1 {
		startArgN = 1
	}
	var q FilterQuery
	if err := json.Unmarshal(raw, &q); err != nil {
		return "", nil, fmt.Errorf("invalid query JSON: %w", err)
	}

	if q.Operator != "AND" && q.Operator != "OR" {
		return "", nil, fmt.Errorf("operator must be AND or OR")
	}
	if len(q.Conditions) == 0 {
		return "", nil, nil
	}
	if len(q.Conditions) > 20 {
		return "", nil, fmt.Errorf("max 20 conditions allowed")
	}

	tableName := "contacts"
	base := standardContactKeys
	if filterType == "conversation" {
		tableName = "conversations"
		base = standardConversationKeys
	}
	whitelist := make(map[string]bool, len(base)+len(customAttrs))
	for k := range base {
		whitelist[k] = true
	}
	for _, k := range customAttrs {
		if !identRegex.MatchString(k) {
			return "", nil, fmt.Errorf("invalid custom attribute key: %s", k)
		}
		whitelist[k] = true
	}

	var clauses []string
	var args []any
	argN := startArgN

	for _, cond := range q.Conditions {
		if !whitelist[cond.AttributeKey] {
			return "", nil, fmt.Errorf("invalid_attribute_key: %s", cond.AttributeKey)
		}
		if !validOperators[cond.FilterOperator] {
			return "", nil, fmt.Errorf("invalid_filter_operator: %s", cond.FilterOperator)
		}

		clause, newArgs, nextN, err := buildCondition(cond, tableName, argN)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, clause)
		args = append(args, newArgs...)
		argN = nextN
	}

	if len(clauses) == 0 {
		return "", nil, nil
	}

	where := strings.Join(clauses, " "+q.Operator+" ")
	return where, args, nil
}

func buildCondition(cond Condition, tableName string, argN int) (string, []any, int, error) {
	col := cond.AttributeKey

	isCustom := !isStandardKey(col, tableName)

	if isCustom {
		if !identRegex.MatchString(col) {
			return "", nil, argN, fmt.Errorf("invalid custom attribute key")
		}
		col = fmt.Sprintf("additional_attributes->>'%s'", col)
	}

	switch cond.FilterOperator {
	case "equal_to":
		if cond.Value == nil || string(cond.Value) == "null" {
			return fmt.Sprintf("%s IS NULL", col), nil, argN, nil
		}
		v, err := unmarshalValue(cond.Value)
		if err != nil {
			return "", nil, argN, fmt.Errorf("invalid value for 'equal_to' operator")
		}
		return fmt.Sprintf("%s = $%d", col, argN), []any{v}, argN + 1, nil
	case "not_equal_to":
		if cond.Value == nil || string(cond.Value) == "null" {
			return fmt.Sprintf("%s IS NOT NULL", col), nil, argN, nil
		}
		v, err := unmarshalValue(cond.Value)
		if err != nil {
			return "", nil, argN, fmt.Errorf("invalid value for 'not_equal_to' operator")
		}
		return fmt.Sprintf("%s != $%d", col, argN), []any{v}, argN + 1, nil
	case "contains":
		v, err := unmarshalValue(cond.Value)
		if err != nil {
			return "", nil, argN, fmt.Errorf("invalid value for 'contains' operator")
		}
		return fmt.Sprintf("%s ILIKE $%d", col, argN), []any{"%" + v + "%"}, argN + 1, nil
	case "starts_with":
		v, err := unmarshalValue(cond.Value)
		if err != nil {
			return "", nil, argN, fmt.Errorf("invalid value for 'starts_with' operator")
		}
		return fmt.Sprintf("%s ILIKE $%d", col, argN), []any{v + "%"}, argN + 1, nil
	case "greater_than":
		v, err := unmarshalValue(cond.Value)
		if err != nil {
			return "", nil, argN, fmt.Errorf("invalid value for 'greater_than' operator")
		}
		if isCustom {
			return fmt.Sprintf("(%s)::numeric > $%d", col, argN), []any{v}, argN + 1, nil
		}
		return fmt.Sprintf("%s > $%d", col, argN), []any{v}, argN + 1, nil
	case "less_than":
		v, err := unmarshalValue(cond.Value)
		if err != nil {
			return "", nil, argN, fmt.Errorf("invalid value for 'less_than' operator")
		}
		if isCustom {
			return fmt.Sprintf("(%s)::numeric < $%d", col, argN), []any{v}, argN + 1, nil
		}
		return fmt.Sprintf("%s < $%d", col, argN), []any{v}, argN + 1, nil
	case "in":
		var values []string
		if err := json.Unmarshal(cond.Value, &values); err != nil {
			return "", nil, argN, fmt.Errorf("invalid value for 'in' operator")
		}
		placeholders := make([]string, len(values))
		args := make([]any, len(values))
		for i, v := range values {
			placeholders[i] = fmt.Sprintf("$%d", argN)
			args[i] = v
			argN++
		}
		return fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ",")), args, argN, nil
	case "between":
		var values []string
		if err := json.Unmarshal(cond.Value, &values); err != nil || len(values) != 2 {
			return "", nil, argN, fmt.Errorf("'between' requires array of 2 values")
		}
		if isCustom {
			return fmt.Sprintf("(%s)::numeric BETWEEN $%d AND $%d", col, argN, argN+1), []any{values[0], values[1]}, argN + 2, nil
		}
		return fmt.Sprintf("%s BETWEEN $%d AND $%d", col, argN, argN+1), []any{values[0], values[1]}, argN + 2, nil
	case "is_null":
		return fmt.Sprintf("%s IS NULL", col), nil, argN, nil
	case "is_not_null":
		return fmt.Sprintf("%s IS NOT NULL", col), nil, argN, nil
	default:
		return "", nil, argN, fmt.Errorf("unsupported operator: %s", cond.FilterOperator)
	}
}

func unmarshalValue(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return fmt.Sprintf("%g", f), nil
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return fmt.Sprintf("%t", b), nil
	}
	return "", fmt.Errorf("unsupported value type")
}

func isStandardKey(key, tableName string) bool {
	if tableName == "conversations" {
		return standardConversationKeys[key]
	}
	return standardContactKeys[key]
}
