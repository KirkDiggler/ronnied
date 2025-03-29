package drink_ledger

import (
	"context"
)

// Repository defines the interface for drink ledger data persistence
type Repository interface {
	// AddDrinkRecord adds a drink record to the ledger
	AddDrinkRecord(ctx context.Context, input *AddDrinkRecordInput) error
	
	// GetDrinkRecordsForGame retrieves all drink records for a game
	GetDrinkRecordsForGame(ctx context.Context, input *GetDrinkRecordsForGameInput) (*GetDrinkRecordsForGameOutput, error)
	
	// GetDrinkRecordsForPlayer retrieves all drink records for a player
	GetDrinkRecordsForPlayer(ctx context.Context, input *GetDrinkRecordsForPlayerInput) (*GetDrinkRecordsForPlayerOutput, error)
	
	// MarkDrinkPaid marks a drink as paid
	MarkDrinkPaid(ctx context.Context, input *MarkDrinkPaidInput) error
}
