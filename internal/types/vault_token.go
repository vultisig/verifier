package types

import "time"

// VaultToken represents a token stored in the database
type VaultToken struct {
	ID         string     `json:"id"`
	TokenID    string     `json:"token_id"`
	PublicKey  string     `json:"public_key"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastUsedAt time.Time  `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
}

func (t *VaultToken) IsRevoked() bool {
	return t.RevokedAt != nil &&
		!t.RevokedAt.IsZero() &&
		t.RevokedAt.Before(time.Now())
}

// VaultTokenCreate represents the data needed to create a new vault token
type VaultTokenCreate struct {
	PublicKey string    `json:"public_key"`
	TokenID   string    `json:"token_id"`
	ExpiresAt time.Time `json:"expires_at"`
}
