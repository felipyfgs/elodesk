package meta

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func AppSecretProof(accessToken, appSecret string) string {
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write([]byte(accessToken))
	return hex.EncodeToString(mac.Sum(nil))
}
