package meta

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// AppSecretProof derives the appsecret_proof parameter required by the Meta
// Graph API for enhanced security: HMAC-SHA256(appSecret, accessToken).
func AppSecretProof(accessToken, appSecret string) string {
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write([]byte(accessToken))
	return hex.EncodeToString(mac.Sum(nil))
}
