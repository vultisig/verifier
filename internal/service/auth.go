package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	jwt.RegisteredClaims
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
	expirationTime := time.Now().Add(expireDuration)
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.JWTSecret)
}

// ValidateToken validates a JWT token and returns the claims
func (a *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return a.JWTSecret, nil
	})
	if err != nil {
		return nil, errors.New("invalid token: " + err.Error())
	}
	if !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	// Validate key fields
	if claims.PublicKey == "" {
		return nil, errors.New("token missing public key")
	}

	return claims, nil
}

// RefreshToken refreshes a JWT token while preserving the public key
func (a *AuthService) RefreshToken(oldToken string) (string, error) {
	claims, err := a.ValidateToken(oldToken)
	if err != nil {
		return "", err
	}
	return a.GenerateToken(claims.PublicKey)
}

// GetTokenPublicKey extracts the public key from a token without validating
// This is useful for debugging or when validation isn't needed
func (a *AuthService) GetTokenPublicKey(tokenStr string) (string, error) {
	token, _ := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return nil, nil // We don't validate the token here
	})

	if token == nil {
		return "", errors.New("cannot parse token")
	}

	if claims, ok := token.Claims.(*Claims); ok {
		return claims.PublicKey, nil
	}

	return "", errors.New("cannot extract public key from token")
}
