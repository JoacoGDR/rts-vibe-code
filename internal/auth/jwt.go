package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID      string `json:"sub"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	jwt.RegisteredClaims
}

type Issuer struct {
	secret    []byte
	accessTTL time.Duration
}

func NewIssuer(secret string, accessTTL time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), accessTTL: accessTTL}
}

type Identity struct {
	UserID      uuid.UUID
	Email       string
	DisplayName string
}

func (i *Issuer) IssueAccess(id Identity) (string, time.Time, error) {
	expiry := time.Now().Add(i.accessTTL)
	claims := Claims{
		UserID:      id.UserID.String(),
		Email:       id.Email,
		DisplayName: id.DisplayName,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiry),
			Issuer:    "supremacy",
			Subject:   id.UserID.String(),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiry, nil
}

func (i *Issuer) Verify(token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return i.secret, nil
	})
	if err != nil {
		return nil, ErrTokenInvalid
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}
