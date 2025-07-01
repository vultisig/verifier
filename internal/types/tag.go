package types

import "time"

// Tag represents a plugin tag with unique identifier, name
type Tag struct {
	ID        string    `json:"id" validate:"required"`
	Name      string    `json:"name" validate:"required,min=2,max=50"`
	CreatedAt time.Time `json:"created_at" validate:"required"`
}
