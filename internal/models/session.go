package models

import (
	"time"
)

// Session represents a drinking session
type Session struct {
	// ID is the unique identifier for this session
	ID string `json:"id"`

	// GuildID is the Discord server/guild this session belongs to
	GuildID string `json:"guild_id"`

	// CreatedAt is when the session was created
	CreatedAt time.Time `json:"created_at"`

	// CreatedBy is the user ID who created the session
	CreatedBy string `json:"created_by"`

	// Active indicates if this is the current active session
	Active bool `json:"active"`
}
