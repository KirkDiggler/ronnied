package models

import (
	"time"
)

// Session represents a drinking session across multiple games
type Session struct {
	// ID is the unique identifier for this session
	ID string
	
	// ChannelID is the Discord channel this session belongs to
	ChannelID string
	
	// CreatedAt is when the session was created
	CreatedAt time.Time
	
	// CreatedBy is the user ID who created the session
	CreatedBy string
	
	// Active indicates if this is the current active session
	Active bool
}
