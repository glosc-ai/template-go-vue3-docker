package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Manager struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

func NewManager(secret, issuer string, ttl time.Duration) (*Manager, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT secret must contain at least 32 characters")
	}
	if issuer == "" {
		return nil, fmt.Errorf("JWT issuer is required")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("JWT TTL must be positive")
	}
	return &Manager{secret: []byte(secret), issuer: issuer, ttl: ttl}, nil
}

func (m *Manager) Issue(subject string) (string, error) {
	if subject == "" {
		return "", fmt.Errorf("JWT subject is required")
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    m.issuer,
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("signing JWT: %w", err)
	}
	return token, nil
}

func (m *Manager) Parse(tokenValue string) (string, error) {
	claims := new(jwt.RegisteredClaims)
	token, err := jwt.ParseWithClaims(tokenValue, claims, func(token *jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return "", fmt.Errorf("validating JWT: %w", err)
	}
	if !token.Valid || claims.Subject == "" {
		return "", fmt.Errorf("JWT is invalid")
	}
	return claims.Subject, nil
}
