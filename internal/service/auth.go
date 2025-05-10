package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/types"
)

type Claims struct {
	jwt.RegisteredClaims
	PublicKey string `json:"public_key"`
	TokenID   string `json:"token_id"`
}

const (
	expireDuration = 7 * 24 * time.Hour
	tokenIDLength  = 32
)

type AuthService struct {
	JWTSecret []byte
	db        storage.DatabaseStorage
}

// NewAuthService creates a new authentication service
func NewAuthService(secret string, db storage.DatabaseStorage) *AuthService {
	return &AuthService{
		JWTSecret: []byte(secret),
		db:        db,
	}
}

// generateTokenID generates a random token ID
func generateTokenID() (string, error) {
	b := make([]byte, tokenIDLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GenerateToken creates a new JWT token and stores it in the database
func (a *AuthService) GenerateToken(publicKey string) (string, error) {
	// Generate a unique token ID
	tokenID, err := generateTokenID()
	if err != nil {
		return "", err
	}

	expirationTime := time.Now().Add(expireDuration)
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		PublicKey: publicKey,
		TokenID:   tokenID,
	}

	// Create JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.JWTSecret)
	if err != nil {
		return "", err
	}

	// Store token in database
	_, err = a.db.CreateVaultToken(context.Background(), types.VaultTokenCreate{
		PublicKey: publicKey,
		TokenID:   tokenID,
		ExpiresAt: expirationTime,
	})
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and checks its revocation status
func (a *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
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
	if claims.TokenID == "" {
		return nil, errors.New("token missing token ID")
	}

	// Check if token is revoked
	dbToken, err := a.db.GetVaultToken(context.Background(), claims.TokenID)
	if err != nil {
		return nil, errors.New("token not found in database")
	}

	if dbToken == nil {
		return nil, errors.New("token not found in database")
	}

	if dbToken.IsRevoked {
		return nil, errors.New("token has been revoked")
	}

	// Update last used timestamp
	err = a.db.UpdateVaultTokenLastUsed(context.Background(), claims.TokenID)
	if err != nil {
		// Log error but don't fail the request
		// TODO: Add proper logging
	}

	return claims, nil
}

// RefreshToken refreshes a JWT token while preserving the public key
func (a *AuthService) RefreshToken(oldToken string) (string, error) {
	claims, err := a.ValidateToken(oldToken)
	if err != nil {
		return "", err
	}

	// Revoke old token
	err = a.db.RevokeVaultToken(context.Background(), claims.TokenID)
	if err != nil {
		return "", err
	}

	// Generate new token
	return a.GenerateToken(claims.PublicKey)
}

// RevokeToken revokes a specific token
func (a *AuthService) RevokeToken(ctx context.Context, vaultKey, tokenID string) error {
	tok, err := a.db.GetVaultToken(ctx, tokenID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("token not found: %w", err)
		}
		return fmt.Errorf("failed to get token: %w", err)
	}

	if tok == nil {
		return fmt.Errorf("token not found")
	}

	if tok.PublicKey != vaultKey {
		return fmt.Errorf("unauthorized token revocation: token belongs to different vault")
	}

	return a.db.RevokeVaultToken(ctx, tokenID)
}

// RevokeAllTokens revokes all tokens for a specific public key
func (a *AuthService) RevokeAllTokens(publicKey string) error {
	return a.db.RevokeAllVaultTokens(context.Background(), publicKey)
}

// GetActiveTokens returns all active tokens for a public key
func (a *AuthService) GetActiveTokens(publicKey string) ([]types.VaultToken, error) {
	return a.db.GetActiveVaultTokens(context.Background(), publicKey)
}
