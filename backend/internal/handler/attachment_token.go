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
// Formatos suportados:
//   - permanente: base64url("{accountID}:{attachmentID}").base64url(sig)
//   - time-based: base64url("{accountID}:{attachmentID}:{expUnix}").base64url(sig)
//
// O caminho preferencial agora é PERMANENTE — espelha exatamente o
// ActiveStorage signed_id do Chatwoot, em que a URL do blob nunca muda. Isso
// permite que o cache HTTP do navegador (Cache-Control: max-age=1y, immutable)
// satisfaça GETs subsequentes sem refazer o download a cada navegação.
//
// O formato time-based continua sendo aceito por compatibilidade com tokens
// já distribuídos (webhooks, integradores externos), mas novos tokens são
// gerados sempre na forma permanente.
//
// Assinatura: HMAC-SHA256(payload) com a KEK do backend (já tem 32+ bytes
// validados na config). KEK aqui é só pra derivar uma chave HMAC — não há
// criptografia, só integridade.

var errInvalidAttachmentToken = errors.New("invalid attachment token")

// SignAttachmentToken gera o token PERMANENTE (sem expiração). Espelha o
// ActiveStorage signed_id do Chatwoot — a URL final é determinística pra um
// dado (accountID, attachmentID), o que faz o navegador acertar o cache HTTP
// em todas as navegações subsequentes.
func SignAttachmentToken(secret []byte, accountID, attachmentID int64) string {
	payload := fmt.Sprintf("%d:%d", accountID, attachmentID)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// SignAttachmentTokenWithTTL é a variante time-based, usada apenas onde
// necessário (não há call-sites internos hoje — mantida exportada porque
// integrações externas podem querer URLs com janela de validade).
func SignAttachmentTokenWithTTL(secret []byte, accountID, attachmentID int64, ttl time.Duration) string {
	exp := time.Now().Add(ttl).Unix()
	payload := fmt.Sprintf("%d:%d:%d", accountID, attachmentID, exp)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// VerifyAttachmentToken valida assinatura (e expiração quando presente) e
// devolve (accountID, attachmentID). Aceita os dois formatos pra permitir
// rotação gradual. Erro genérico de propósito — não diferenciar mau-formado
// vs expirado evita oracles.
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
