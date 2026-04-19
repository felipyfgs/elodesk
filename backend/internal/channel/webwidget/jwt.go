package webwidget

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type VisitorClaims struct {
	Sub            string `json:"sub"`
	WebsiteToken   string `json:"website_token"`
	ConversationID int64  `json:"conversation_id,omitempty"`
	ContactID      int64  `json:"contact_id"`
	jwt.RegisteredClaims
}

type VisitorJWTService struct {
	secret []byte
	ttl    time.Duration
}

func NewVisitorJWTService(secret string, ttlDays int) *VisitorJWTService {
	return &VisitorJWTService{
		secret: []byte(secret),
		ttl:    time.Duration(ttlDays) * 24 * time.Hour,
	}
}

func (s *VisitorJWTService) Issue(contactID int64, contactIdentifier, websiteToken string, conversationID int64) (string, error) {
	now := time.Now().UTC()
	claims := VisitorClaims{
		Sub:            contactIdentifier,
		WebsiteToken:   websiteToken,
		ConversationID: conversationID,
		ContactID:      contactID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign visitor jwt: %w", err)
	}
	return signed, nil
}

func (s *VisitorJWTService) Parse(tokenStr string) (*VisitorClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &VisitorClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse visitor jwt: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid visitor jwt")
	}
	claims, ok := token.Claims.(*VisitorClaims)
	if !ok {
		return nil, fmt.Errorf("invalid visitor jwt claims type")
	}
	return claims, nil
}
