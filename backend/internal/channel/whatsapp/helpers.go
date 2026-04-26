package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// waSourceIDPattern collapses inbound WhatsApp sender IDs to their canonical
// digits-only form (drops the device suffix `:N` and the host segment
// `@s.whatsapp.net` / `@g.us`). Mirrors Chatwoot's
// PhoneNumberNormalizationService — without it we end up with multiple
// contact_inboxes per contact (one per device session).
var waSourceIDPattern = regexp.MustCompile(`^(\d+)(?::\d+)?(?:@s\.whatsapp\.net|@g\.us)?$`)

// normalizeWaSourceID promotes raw WhatsApp JIDs to their phone-number-only
// canonical form. Returns the input unchanged when it doesn't match — keeps
// us safe against unexpected provider formats.
func normalizeWaSourceID(raw string) string {
	raw = strings.TrimSpace(raw)
	if m := waSourceIDPattern.FindStringSubmatch(raw); m != nil {
		return m[1]
	}
	return raw
}

func buildSendBody(to, content, mediaURL, mediaType, templateName, templateLang, templateComponents string) string {
	if templateName != "" {
		tmpl := map[string]interface{}{
			"messaging_product": "whatsapp",
			"to":                to,
			"type":              "template",
			"template": map[string]interface{}{
				"name": templateName,
				"language": map[string]string{
					"code": templateLang,
				},
			},
		}
		if templateComponents != "" {
			var comps []interface{}
			if err := json.Unmarshal([]byte(templateComponents), &comps); err == nil {
				tmpl["template"].(map[string]interface{})["components"] = comps
			}
		}
		b, _ := json.Marshal(tmpl)
		return string(b)
	}

	if mediaURL != "" && mediaType != "" {
		media := map[string]interface{}{
			"link": mediaURL,
		}
		if mediaType == "document" {
			media["filename"] = "file"
		}
		msg := map[string]interface{}{
			"messaging_product": "whatsapp",
			"to":                to,
			"type":              mediaType,
			mediaType:           media,
		}
		if content != "" {
			msg[mediaType].(map[string]interface{})["caption"] = content
		}
		b, _ := json.Marshal(msg)
		return string(b)
	}

	msg := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text": map[string]string{
			"body": content,
		},
	}
	b, _ := json.Marshal(msg)
	return string(b)
}

func ProviderForType(providerType string, httpClient *http.Client) (Provider, error) {
	switch providerType {
	case "whatsapp_cloud":
		return NewCloudProvider(httpClient), nil
	case "default_360dialog":
		return NewDialog360Provider(httpClient), nil
	default:
		return nil, fmt.Errorf("unknown whatsapp provider: %s", providerType)
	}
}

func dedupKey(wamid string) string {
	return "elodesk:wa:dedup:" + wamid
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if !strings.HasPrefix(phone, "+") {
		phone = "+" + phone
	}
	return phone
}

func WithPhoneNumberID(ctx context.Context, phoneNumberID string) context.Context {
	return context.WithValue(ctx, ctxKeyPhoneNumberID{}, phoneNumberID)
}
