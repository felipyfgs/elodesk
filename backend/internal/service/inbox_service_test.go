package service

import (
	"errors"
	"testing"
)

func TestValidateAgentReplyTimeWindow(t *testing.T) {
	t.Helper()

	cases := []struct {
		name    string
		attrs   map[string]any
		wantErr bool
	}{
		{name: "nil attrs", attrs: nil, wantErr: false},
		{name: "empty attrs", attrs: map[string]any{}, wantErr: false},
		{name: "other keys ok", attrs: map[string]any{"x": "y"}, wantErr: false},
		{name: "positive float64", attrs: map[string]any{"agent_reply_time_window": float64(30)}, wantErr: false},
		{name: "positive int", attrs: map[string]any{"agent_reply_time_window": 15}, wantErr: false},
		{name: "positive int64", attrs: map[string]any{"agent_reply_time_window": int64(5)}, wantErr: false},
		{name: "zero rejected", attrs: map[string]any{"agent_reply_time_window": float64(0)}, wantErr: true},
		{name: "negative rejected", attrs: map[string]any{"agent_reply_time_window": -1}, wantErr: true},
		{name: "string rejected", attrs: map[string]any{"agent_reply_time_window": "30"}, wantErr: true},
		{name: "nil value rejected", attrs: map[string]any{"agent_reply_time_window": nil}, wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateAgentReplyTimeWindow(tc.attrs)
			if tc.wantErr && !errors.Is(err, ErrInvalidAgentReplyTimeWindow) {
				t.Fatalf("err = %v, want ErrInvalidAgentReplyTimeWindow", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
		})
	}
}
