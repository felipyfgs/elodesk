package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var errInvalidAttachmentToken = errors.New("invalid attachment token")

func SignAttachmentToken(secret []byte, accountID, attachmentID int64) string {
	payload := fmt.Sprintf("%d:%d", accountID, attachmentID)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func SignAttachmentTokenWithTTL(secret []byte, accountID, attachmentID int64, ttl time.Duration) string {
	exp := time.Now().Add(ttl).Unix()
	payload := fmt.Sprintf("%d:%d:%d", accountID, attachmentID, exp)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func VerifyAttachmentToken(secret []byte, token string) (int64, int64, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return 0, 0, errInvalidAttachmentToken
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, 0, errInvalidAttachmentToken
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, 0, errInvalidAttachmentToken
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(payloadBytes)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return 0, 0, errInvalidAttachmentToken
	}

	fields := strings.Split(string(payloadBytes), ":")
	if len(fields) != 2 && len(fields) != 3 {
		return 0, 0, errInvalidAttachmentToken
	}
	accountID, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, 0, errInvalidAttachmentToken
	}
	attachmentID, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0, 0, errInvalidAttachmentToken
	}
	if len(fields) == 3 {
		exp, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			return 0, 0, errInvalidAttachmentToken
		}
		if time.Now().Unix() > exp {
			return 0, 0, errInvalidAttachmentToken
		}
	}
	return accountID, attachmentID, nil
}
