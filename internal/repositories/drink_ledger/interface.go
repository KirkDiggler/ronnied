package drink_ledger

//go:generate mockgen -package=mocks -destination=mocks/mock_repository.go github.com/KirkDiggler/ronnied/internal/repositories/drink_ledger Repository

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
	
	// CreateDrinkRecord creates a new drink record with a generated UUID
	CreateDrinkRecord(ctx context.Context, input *CreateDrinkRecordInput) (*CreateDrinkRecordOutput, error)
	
	// ArchiveDrinkRecords marks all drink records for a game as archived
	ArchiveDrinkRecords(ctx context.Context, input *ArchiveDrinkRecordsInput) error
	
	// DeleteDrinkRecords deletes all drink records for a game
	DeleteDrinkRecords(ctx context.Context, input *DeleteDrinkRecordsInput) error
	
	// CreateSession creates a new drinking session
	CreateSession(ctx context.Context, input *CreateSessionInput) (*CreateSessionOutput, error)
	
	// GetCurrentSession retrieves the current active session for a channel
	GetCurrentSession(ctx context.Context, input *GetCurrentSessionInput) (*GetCurrentSessionOutput, error)
	
	// GetDrinkRecordsForSession retrieves all drink records for a session
	GetDrinkRecordsForSession(ctx context.Context, input *GetDrinkRecordsForSessionInput) (*GetDrinkRecordsForSessionOutput, error)
}
