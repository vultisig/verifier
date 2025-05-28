package types

// Tag represents a plugin tag with unique identifier, name and color
type Tag struct {
	ID    string `json:"id" validate:"required"`
	Name  string `json:"name" validate:"required,min=2,max=50"`
	Color string `json:"color" validate:"required,hexcolor"`
}
