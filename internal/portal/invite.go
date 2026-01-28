package portal

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	InviteLinkExpiry = 8 * time.Hour
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrInviteExpired    = errors.New("invite has expired")
	ErrInvalidPayload   = errors.New("invalid invite payload")
)

// InvitePayload contains the data encoded in a magic link
type InvitePayload struct {
	PluginID  string `json:"plugin_id"`
	LinkID    string `json:"link_id"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"expires_at"`
	InvitedBy string `json:"invited_by"`
}

// InviteService handles magic link generation and validation
type InviteService struct {
	hmacSecret []byte
	baseURL    string
}

// NewInviteService creates a new invite service
func NewInviteService(hmacSecret, baseURL string) *InviteService {
	return &InviteService{
		hmacSecret: []byte(hmacSecret),
		baseURL:    baseURL,
	}
}

// GenerateInviteLink creates a new magic link for inviting a team member
func (s *InviteService) GenerateInviteLink(pluginID, role, invitedBy string) (string, string, error) {
	linkID := uuid.New().String()

	payload := InvitePayload{
		PluginID:  pluginID,
		LinkID:    linkID,
		Role:      role,
		ExpiresAt: time.Now().Add(InviteLinkExpiry).Unix(),
		InvitedBy: invitedBy,
	}

	// Encode payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Base64 encode the payload
	encodedPayload := base64.URLEncoding.EncodeToString(payloadBytes)

	// Generate HMAC signature
	signature := s.sign(payloadBytes)
	encodedSignature := base64.URLEncoding.EncodeToString(signature)

	// Build the full URL
	link := fmt.Sprintf("%s/invite/accept?data=%s&sig=%s", s.baseURL, encodedPayload, encodedSignature)

	return link, linkID, nil
}

// ValidateInviteLink validates a magic link and returns the payload
func (s *InviteService) ValidateInviteLink(encodedPayload, encodedSignature string) (*InvitePayload, error) {
	// Decode the payload
	payloadBytes, err := base64.URLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode payload", ErrInvalidPayload)
	}

	// Decode the signature
	signature, err := base64.URLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode signature", ErrInvalidPayload)
	}

	// Verify HMAC signature
	if !s.verify(payloadBytes, signature) {
		return nil, ErrInvalidSignature
	}

	// Parse the payload
	var payload InvitePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal payload", ErrInvalidPayload)
	}

	// Check expiration
	if time.Now().Unix() > payload.ExpiresAt {
		return nil, ErrInviteExpired
	}

	// Validate role
	if payload.Role != "editor" && payload.Role != "viewer" {
		return nil, fmt.Errorf("%w: invalid role", ErrInvalidPayload)
	}

	return &payload, nil
}

// sign creates an HMAC-SHA256 signature for the given data
func (s *InviteService) sign(data []byte) []byte {
	h := hmac.New(sha256.New, s.hmacSecret)
	h.Write(data)
	return h.Sum(nil)
}

// verify checks if the signature is valid for the given data
func (s *InviteService) verify(data, signature []byte) bool {
	expected := s.sign(data)
	return hmac.Equal(expected, signature)
}
