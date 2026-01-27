package portal

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// PortalClaims represents the JWT claims for portal authentication
type PortalClaims struct {
	jwt.RegisteredClaims
	PublicKey string `json:"public_key"`
	Address   string `json:"address"`
	TokenID   string `json:"token_id"`
}

const (
	portalTokenExpireDuration = 7 * 24 * time.Hour
	portalTokenIDLength       = 32
)

// PortalAuthService handles JWT authentication for the portal
type PortalAuthService struct {
	jwtSecret []byte
	logger    *logrus.Logger
}

// NewPortalAuthService creates a new portal authentication service
func NewPortalAuthService(secret string, logger *logrus.Logger) *PortalAuthService {
	return &PortalAuthService{
		jwtSecret: []byte(secret),
		logger:    logger.WithField("service", "portal-auth").Logger,
	}
}

// generateTokenID generates a random token ID
func generatePortalTokenID() (string, error) {
	b := make([]byte, portalTokenIDLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GenerateToken creates a new JWT token for the portal
func (a *PortalAuthService) GenerateToken(publicKey, address string) (string, error) {
	tokenID, err := generatePortalTokenID()
	if err != nil {
		return "", err
	}

	expirationTime := time.Now().Add(portalTokenExpireDuration)
	claims := &PortalClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		PublicKey: publicKey,
		Address:   address,
		TokenID:   tokenID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (a *PortalAuthService) ValidateToken(tokenStr string) (*PortalClaims, error) {
	claims := &PortalClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return a.jwtSecret, nil
	})
	if err != nil {
		return nil, errors.New("invalid token: " + err.Error())
	}
	if !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	if claims.PublicKey == "" {
		return nil, errors.New("token missing public key")
	}
	if claims.Address == "" {
		return nil, errors.New("token missing address")
	}
	if claims.TokenID == "" {
		return nil, errors.New("token missing token ID")
	}

	return claims, nil
}
