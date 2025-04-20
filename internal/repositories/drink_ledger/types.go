package drink_ledger

import (
	"time"
	"github.com/KirkDiggler/ronnied/internal/models"
)

// AddDrinkRecordInput contains parameters for adding a drink record
type AddDrinkRecordInput struct {
	Record *models.DrinkLedger
}

// GetDrinkRecordsForGameInput contains parameters for retrieving drink records for a game
type GetDrinkRecordsForGameInput struct {
	GameID string
}

// GetDrinkRecordsForGameOutput contains the result of retrieving drink records for a game
type GetDrinkRecordsForGameOutput struct {
	Records []*models.DrinkLedger
}

// GetDrinkRecordsForPlayerInput contains parameters for retrieving drink records for a player
type GetDrinkRecordsForPlayerInput struct {
	PlayerID string
}

// GetDrinkRecordsForPlayerOutput contains the result of retrieving drink records for a player
type GetDrinkRecordsForPlayerOutput struct {
	Records []*models.DrinkLedger
}

// MarkDrinkPaidInput contains parameters for marking a drink as paid
type MarkDrinkPaidInput struct {
	DrinkID string
}

// CreateDrinkRecordInput contains parameters for creating a new drink record
type CreateDrinkRecordInput struct {
	GameID       string
	FromPlayerID string // Empty for system-assigned drinks
	ToPlayerID   string
	Reason       models.DrinkReason
	Timestamp    time.Time
	SessionID    string // ID of the session this drink belongs to
}

// CreateDrinkRecordOutput contains the result of creating a new drink record
type CreateDrinkRecordOutput struct {
	Record *models.DrinkLedger
}

// ArchiveDrinkRecordsInput contains parameters for archiving drink records
type ArchiveDrinkRecordsInput struct {
	// GameID is the ID of the game to archive drink records for
	GameID string
}

// DeleteDrinkRecordsInput contains parameters for deleting drink records
type DeleteDrinkRecordsInput struct {
	// GameID is the ID of the game to delete drink records for
	GameID string
}
