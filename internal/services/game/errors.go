package game

// GameError is a custom error type for game-related errors
type GameError string

// Error implements the error interface
func (e GameError) Error() string {
	return string(e)
}

// Define errors
const (
	ErrGameNotFound        GameError = "game not found"
	ErrPlayerNotFound      GameError = "player not found"
	ErrPlayerAlreadyInGame GameError = "player already in game"
	ErrGameAlreadyExists   GameError = "game already exists for this channel"
	ErrInvalidGameState    GameError = "invalid game state"
	ErrPlayerNotInGame     GameError = "player not in game"
	ErrGameFull            GameError = "game is at maximum capacity"
	ErrRollOffGameNotFound GameError = "no active roll-off game found"
	ErrNilConfig           GameError = "config cannot be nil"
	ErrNilGameRepo         GameError = "game repository cannot be nil"
	ErrNilPlayerRepo       GameError = "player repository cannot be nil"
	ErrNilDrinkLedgerRepo  GameError = "drink ledger repository cannot be nil"
	ErrNilDiceRoller       GameError = "dice roller cannot be nil"
	ErrNilClock            GameError = "clock cannot be nil"
	ErrNilUUIDGenerator    GameError = "UUID generator cannot be nil"
)
