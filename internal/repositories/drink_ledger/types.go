package drink_ledger

import "github.com/KirkDiggler/ronnied/internal/models"

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
