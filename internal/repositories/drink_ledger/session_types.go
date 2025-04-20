package drink_ledger

import (
	"github.com/KirkDiggler/ronnied/internal/models"
)

// CreateSessionInput contains parameters for creating a new session
type CreateSessionInput struct {
	// ChannelID is the Discord channel this session belongs to
	ChannelID string
	
	// CreatedBy is the user ID who created the session
	CreatedBy string
}

// CreateSessionOutput contains the result of creating a new session
type CreateSessionOutput struct {
	// Session is the newly created session
	Session *models.Session
}

// GetCurrentSessionInput contains parameters for retrieving the current session
type GetCurrentSessionInput struct {
	// ChannelID is the Discord channel to get the session for
	ChannelID string
}

// GetCurrentSessionOutput contains the result of retrieving the current session
type GetCurrentSessionOutput struct {
	// Session is the current active session, or nil if none exists
	Session *models.Session
}

// GetDrinkRecordsForSessionInput contains parameters for retrieving drink records for a session
type GetDrinkRecordsForSessionInput struct {
	// SessionID is the ID of the session to get drink records for
	SessionID string
}

// GetDrinkRecordsForSessionOutput contains the result of retrieving drink records for a session
type GetDrinkRecordsForSessionOutput struct {
	// Records is the list of drink records for the session
	Records []*models.DrinkLedger
}
