package twitter

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type OAuthClient struct {
	httpClient     *http.Client
	consumerKey    string
	consumerSecret string
	callbackURL    string
}

func NewOAuthClient(consumerKey, consumerSecret, callbackURL string) *OAuthClient {
	return &OAuthClient{
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		consumerKey:    consumerKey,
		consumerSecret: consumerSecret,
		callbackURL:    callbackURL,
	}
}

func (c *OAuthClient) ConsumerKey() string { return c.consumerKey }

type RequestTokenResponse struct {
	OAuthToken             string
	OAuthTokenSecret       string
	OAuthCallbackConfirmed bool
}

type AccessTokenResponse struct {
	OAuthToken       string
	OAuthTokenSecret string
	UserID           string
	ScreenName       string
}

func (c *OAuthClient) RequestToken(ctx context.Context) (*RequestTokenResponse, error) {
	endpoint := apiBase + requestTokenPath
	extra := map[string]string{"oauth_callback": c.callbackURL}
	body, err := c.signedPost(ctx, endpoint, "", "", extra, nil)
	if err != nil {
		return nil, err
	}
	values, err := url.ParseQuery(body)
	if err != nil {
		return nil, fmt.Errorf("twitter request_token: parse: %w", err)
	}
	confirmed, _ := strconv.ParseBool(values.Get("oauth_callback_confirmed"))
	return &RequestTokenResponse{
		OAuthToken:             values.Get("oauth_token"),
		OAuthTokenSecret:       values.Get("oauth_token_secret"),
		OAuthCallbackConfirmed: confirmed,
	}, nil
}

func (c *OAuthClient) AuthorizeURL(requestToken string) string {
	return apiBase + authenticatePath + "?oauth_token=" + url.QueryEscape(requestToken)
}

func (c *OAuthClient) AccessToken(ctx context.Context, requestToken, requestTokenSecret, verifier string) (*AccessTokenResponse, error) {
	endpoint := apiBase + accessTokenPath
	extra := map[string]string{"oauth_verifier": verifier}
	body, err := c.signedPost(ctx, endpoint, requestToken, requestTokenSecret, extra, nil)
	if err != nil {
		return nil, err
	}
	values, err := url.ParseQuery(body)
	if err != nil {
		return nil, fmt.Errorf("twitter access_token: parse: %w", err)
	}
	return &AccessTokenResponse{
		OAuthToken:       values.Get("oauth_token"),
		OAuthTokenSecret: values.Get("oauth_token_secret"),
		UserID:           values.Get("user_id"),
		ScreenName:       values.Get("screen_name"),
	}, nil
}

func (c *OAuthClient) signedPost(
	ctx context.Context, endpoint, token, tokenSecret string,
	extraOauth map[string]string, body url.Values,
) (string, error) {
	header, err := c.buildAuthHeader(http.MethodPost, endpoint, token, tokenSecret, extraOauth, body)
	if err != nil {
		return "", err
	}

	var reader io.Reader
	if body != nil {
		reader = strings.NewReader(body.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, reader)
	if err != nil {
		return "", fmt.Errorf("twitter oauth: new request: %w", err)
	}
	req.Header.Set("Authorization", header)
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("twitter oauth: do: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("twitter oauth: read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("twitter oauth: status %d: %s", resp.StatusCode, string(raw))
	}
	return string(raw), nil
}

func (c *OAuthClient) buildAuthHeader(
	method, endpoint, token, tokenSecret string,
	extraOauth map[string]string, body url.Values,
) (string, error) {
	nonce, err := newNonce()
	if err != nil {
		return "", err
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	oauthParams := map[string]string{
		"oauth_consumer_key":     c.consumerKey,
		"oauth_nonce":            nonce,
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        timestamp,
		"oauth_version":          "1.0",
	}
	if token != "" {
		oauthParams["oauth_token"] = token
	}
	for k, v := range extraOauth {
		oauthParams[k] = v
	}

	signature := signRequest(method, endpoint, oauthParams, body, c.consumerSecret, tokenSecret)
	oauthParams["oauth_signature"] = signature

	keys := make([]string, 0, len(oauthParams))
	for k := range oauthParams {
		if !strings.HasPrefix(k, "oauth_") {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, percentEncode(k)+`="`+percentEncode(oauthParams[k])+`"`)
	}
	return "OAuth " + strings.Join(parts, ", "), nil
}

func signRequest(method, endpoint string, oauthParams map[string]string, body url.Values, consumerSecret, tokenSecret string) string {
	all := make(map[string]string, len(oauthParams)+len(body))
	for k, v := range oauthParams {
		if k == "oauth_signature" {
			continue
		}
		all[k] = v
	}
	for k, vs := range body {
		if len(vs) > 0 {
			all[k] = vs[0]
		}
	}

	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, percentEncode(k)+"="+percentEncode(all[k]))
	}
	paramString := strings.Join(pairs, "&")

	baseURL := endpoint
	if i := strings.IndexByte(endpoint, '?'); i >= 0 {
		baseURL = endpoint[:i]
	}
	base := strings.ToUpper(method) + "&" + percentEncode(baseURL) + "&" + percentEncode(paramString)
	signingKey := percentEncode(consumerSecret) + "&" + percentEncode(tokenSecret)

	mac := hmac.New(sha1.New, []byte(signingKey))
	mac.Write([]byte(base))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func percentEncode(in string) string {
	var b strings.Builder
	b.Grow(len(in))
	for _, r := range in {
		switch {
		case (r >= 'A' && r <= 'Z'), (r >= 'a' && r <= 'z'), (r >= '0' && r <= '9'),
			r == '-', r == '.', r == '_', r == '~':
			b.WriteRune(r)
		default:
			for _, b2 := range []byte(string(r)) {
				b.WriteString(fmt.Sprintf("%%%02X", b2))
			}
		}
	}
	return b.String()
}

func newNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("twitter oauth: nonce: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
