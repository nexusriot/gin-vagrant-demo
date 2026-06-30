package domain

import "time"

// Item is the core domain entity.
type Item struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateItemRequest is the validated payload for POST /v1/items.
type CreateItemRequest struct {
	Title string `json:"title" binding:"required,min=1,max=255"`
	Body  string `json:"body"  binding:"max=65535"`
}

// UpdateItemRequest is the validated payload for PUT /v1/items/:id.
type UpdateItemRequest struct {
	Title string `json:"title" binding:"required,min=1,max=255"`
	Body  string `json:"body"  binding:"max=65535"`
}
