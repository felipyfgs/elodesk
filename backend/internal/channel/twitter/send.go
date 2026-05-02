package twitter

import (
	"context"
	"fmt"
)

type SendOptions struct {
	ParticipantID string
	Content       string
}

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
