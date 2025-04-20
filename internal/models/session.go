package models

import (
	"time"
)

// Session represents a drinking session
type Session struct {
	// ID is the unique identifier for this session
	ID string

	// GuildID is the Discord server/guild this session belongs to
	GuildID string

	// CreatedAt is when the session was created
	CreatedAt time.Time

	// CreatedBy is the user ID who created the session
	CreatedBy string

	// Active indicates if this is the current active session
	Active bool
}
