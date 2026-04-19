package webwidget

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

func ComputeIdentifierHash(hmacToken, identifier string) string {
	mac := hmac.New(sha256.New, []byte(hmacToken))
	mac.Write([]byte(identifier))
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyIdentifierHash(hmacToken, identifier, providedHash string) bool {
	expected := ComputeIdentifierHash(hmacToken, identifier)
	if len(expected) != len(providedHash) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(providedHash)) == 1
}

func MustMatchHMAC(hmacToken, identifier, providedHash string) error {
	if providedHash == "" {
		return fmt.Errorf("identifier_hash is required")
	}
	if !VerifyIdentifierHash(hmacToken, identifier, providedHash) {
		return fmt.Errorf("invalid identifier hash")
	}
	return nil
}
