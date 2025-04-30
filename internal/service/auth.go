package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt"
)

type Claims struct {
	jwt.StandardClaims
}

const (
	expireDuration = 7 * 24 * time.Hour
)

type AuthService struct {
	JWTSecret []byte
}

// NewAuthService creates a new authentication service
func NewAuthService(secret string) *AuthService {
	return &AuthService{
		JWTSecret: []byte(secret),
	}
}

// GenerateToken creates a new JWT token
func (a *AuthService) GenerateToken() (string, error) {
	expirationTime := time.Now().Add(expireDuration).Unix()
	claims := &Claims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.JWTSecret)
}

// ValidateToken validates a JWT token
func (a *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return a.JWTSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}
	return claims, nil
}

// RefreshToken refreshes a JWT token
func (a *AuthService) RefreshToken(oldToken string) (string, error) {
	_, err := a.ValidateToken(oldToken)
	if err != nil {
		return "", err
	}
	return a.GenerateToken()
}
