package types

// CreateTagDto represents the request payload for creating a new tag
type CreateTagDto struct {
    Name  string `json:"name" validate:"required,min=2,max=50"`
    Color string `json:"color" validate:"required,hexcolor"`
}

// Tag represents a plugin tag with unique identifier, name and color
type Tag struct {
    ID    string `json:"id" validate:"required"`
    Name  string `json:"name" validate:"required,min=2,max=50"`
    Color string `json:"color" validate:"required,hexcolor"`
}
