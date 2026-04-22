package twitter

import (
	"context"
	"fmt"
)

// SendOptions controls the outbound DM payload. Only text is supported on
// the v2 DM endpoint at the moment — attachments will be added later.
type SendOptions struct {
	ParticipantID string
	Content       string
}

// Send dispatches an outbound DM and returns the v2 dm_event id of the
// created message.
func Send(ctx context.Context, api *APIClient, accessToken, accessTokenSecret string, opts SendOptions) (string, error) {
	if accessToken == "" || accessTokenSecret == "" {
		return "", fmt.Errorf("twitter send: missing access token pair")
	}
	if opts.ParticipantID == "" {
		return "", fmt.Errorf("twitter send: missing participant id")
	}
	if opts.Content == "" {
		return "", fmt.Errorf("twitter send: empty content")
	}
	return api.SendDM(ctx, accessToken, accessTokenSecret, opts.ParticipantID, opts.Content)
}
