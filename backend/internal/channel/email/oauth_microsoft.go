package email

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

var microsoftScopes = []string{
	"Mail.Read",
	"Mail.Send",
	"offline_access",
}

func microsoftOAuthConfig() *oauth2.Config {
	tenantID := os.Getenv("MICROSOFT_OAUTH_TENANT_ID")
	if tenantID == "" {
		tenantID = "common"
	}
	return &oauth2.Config{
		ClientID:     os.Getenv("MICROSOFT_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("MICROSOFT_OAUTH_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("MICROSOFT_OAUTH_REDIRECT_URI"),
		Scopes:       microsoftScopes,
		Endpoint:     microsoft.AzureADEndpoint(tenantID),
	}
}

// MicrosoftAuthURL returns the consent-screen URL for a given state token.
func MicrosoftAuthURL(state string) string {
	return microsoftOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// MicrosoftExchangeCode exchanges an authorization code for tokens.
func MicrosoftExchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	cfg := microsoftOAuthConfig()
	tok, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("microsoft exchange: %w", err)
	}
	return &OAuthTokens{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresOn:    tok.Expiry,
	}, nil
}

// MicrosoftRefreshToken exchanges a refresh_token for a new access_token.
func MicrosoftRefreshToken(ctx context.Context, refreshToken string) (*OAuthTokens, error) {
	cfg := microsoftOAuthConfig()
	src := cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	tok, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("microsoft refresh: %w", err)
	}
	return &OAuthTokens{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresOn:    tok.Expiry,
	}, nil
}
