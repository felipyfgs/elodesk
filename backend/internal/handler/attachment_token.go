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

// Token URL-safe pra GET /attachments/:id/file. Padrão espelha o ActiveStorage
// signed_id do Chatwoot: payload base64url + assinatura HMAC.
//
// Formato:  base64url(payload).base64url(sig)
// Payload:  "{accountID}:{attachmentID}:{expUnix}"
//
// Assinatura: HMAC-SHA256(payload) com a KEK do backend (já tem 32+ bytes
// validados na config). KEK aqui é só pra derivar uma chave HMAC — não há
// criptografia, só integridade/expiração.

var errInvalidAttachmentToken = errors.New("invalid attachment token")

// SignAttachmentToken gera o token. ttl define janela de validade — tipicamente
// 15min, igual ao MinIO presignedTTL atual.
func SignAttachmentToken(secret []byte, accountID, attachmentID int64, ttl time.Duration) string {
	exp := time.Now().Add(ttl).Unix()
	payload := fmt.Sprintf("%d:%d:%d", accountID, attachmentID, exp)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// VerifyAttachmentToken valida assinatura e expiração e devolve (accountID,
// attachmentID). Erro genérico de propósito — não diferenciar mau-formado vs
// expirado evita oracles.
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
	if len(fields) != 3 {
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
	exp, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return 0, 0, errInvalidAttachmentToken
	}
	if time.Now().Unix() > exp {
		return 0, 0, errInvalidAttachmentToken
	}
	return accountID, attachmentID, nil
}
