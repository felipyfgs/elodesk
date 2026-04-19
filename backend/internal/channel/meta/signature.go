package meta

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
)

// VerifySignature validates the X-Hub-Signature-256 header against the raw
// request body. Returns true when the signature matches. When
// META_ALLOW_UNSIGNED=true the check is bypassed (dev only).
func VerifySignature(body []byte, header, appSecret string) bool {
	if os.Getenv("META_ALLOW_UNSIGNED") == "true" {
		return true
	}
	if header == "" || appSecret == "" {
		return false
	}
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	expected := mac.Sum(nil)
	return hmac.Equal(expected, provided)
}
