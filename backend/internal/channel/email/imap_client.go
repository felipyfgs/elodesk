package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-sasl"

	"backend/internal/model"
)

// ImapClient wraps go-imap/v2 and handles PLAIN and XOAUTH2 auth.
type ImapClient struct {
	client *imapclient.Client
}

// Connect dials the IMAP server and authenticates.
// For provider=generic it uses PLAIN login; for google/microsoft it uses OAuthBearer.
func Connect(ctx context.Context, ch *model.ChannelEmail, decryptFn func(string) (string, error)) (*ImapClient, error) {
	addr := fmt.Sprintf("%s:%d", *ch.ImapAddress, *ch.ImapPort)

	var c *imapclient.Client
	var err error

	opts := &imapclient.Options{
		TLSConfig: &tls.Config{ServerName: *ch.ImapAddress},
	}

	if ch.ImapEnableSSL {
		c, err = imapclient.DialTLS(addr, opts)
	} else {
		c, err = imapclient.DialInsecure(addr, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("imap dial %s: %w", addr, err)
	}

	switch ch.Provider {
	case "generic":
		if ch.ImapLogin == nil || ch.ImapPasswordCiphertext == nil {
			_ = c.Close()
			return nil, fmt.Errorf("imap legacy: missing login or password")
		}
		password, err := decryptFn(*ch.ImapPasswordCiphertext)
		if err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("imap decrypt password: %w", err)
		}
		if err := c.Login(*ch.ImapLogin, password).Wait(); err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("imap login: %w", err)
		}

	case "google", "microsoft":
		if ch.ImapLogin == nil {
			_ = c.Close()
			return nil, fmt.Errorf("imap oauth: missing login")
		}
		if ch.ProviderConfig == nil {
			_ = c.Close()
			return nil, fmt.Errorf("imap oauth: provider_config is nil")
		}
		accessToken, err := decryptFn(*ch.ProviderConfig)
		if err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("imap oauth: decrypt provider_config: %w", err)
		}
		saslClient := sasl.NewOAuthBearerClient(&sasl.OAuthBearerOptions{
			Username: *ch.ImapLogin,
			Token:    accessToken,
		})
		if err := c.Authenticate(saslClient); err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("imap oauthbearer: %w", err)
		}

	default:
		_ = c.Close()
		return nil, fmt.Errorf("imap: unknown provider %q", ch.Provider)
	}

	_ = ctx
	return &ImapClient{client: c}, nil
}

// FetchSince fetches messages with UIDs > sinceUID from INBOX.
// It returns a list of raw RFC 2822 messages paired with their UIDs.
func (ic *ImapClient) FetchSince(sinceUID uint32) ([]FetchedMessage, error) {
	if _, err := ic.client.Select("INBOX", nil).Wait(); err != nil {
		return nil, fmt.Errorf("imap select INBOX: %w", err)
	}

	from := imap.UID(sinceUID + 1)
	uidRange := imap.UIDSet{imap.UIDRange{Start: from, Stop: 0}}
	criteria := &imap.SearchCriteria{
		UID: []imap.UIDSet{uidRange},
	}

	searchData, err := ic.client.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("imap uid search: %w", err)
	}
	allUIDs := searchData.AllUIDs()
	if len(allUIDs) == 0 {
		return nil, nil
	}

	fetchSet := imap.UIDSetNum(allUIDs...)
	fetchCmd := ic.client.Fetch(fetchSet, &imap.FetchOptions{
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{{}},
	})
	defer func() { _ = fetchCmd.Close() }()

	var msgs []FetchedMessage
	for {
		msgData := fetchCmd.Next()
		if msgData == nil {
			break
		}
		var uid imap.UID
		var raw []byte
		for {
			item := msgData.Next()
			if item == nil {
				break
			}
			switch v := item.(type) {
			case imapclient.FetchItemDataUID:
				uid = v.UID
			case imapclient.FetchItemDataBodySection:
				raw, err = io.ReadAll(v.Literal)
				if err != nil {
					return nil, fmt.Errorf("imap read body: %w", err)
				}
			}
		}
		if raw != nil {
			msgs = append(msgs, FetchedMessage{UID: uint32(uid), Raw: raw})
		}
	}
	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("imap fetch close: %w", err)
	}
	return msgs, nil
}

// Close terminates the IMAP connection.
func (ic *ImapClient) Close() error {
	return ic.client.Logout().Wait()
}

// FetchedMessage pairs a UID with a raw RFC 2822 message body.
type FetchedMessage struct {
	UID uint32
	Raw []byte
}
