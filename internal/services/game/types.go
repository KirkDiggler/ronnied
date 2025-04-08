package game

import (
	"time"

	"github.com/KirkDiggler/ronnied/internal/common/clock"
	"github.com/KirkDiggler/ronnied/internal/common/uuid"
	"github.com/KirkDiggler/ronnied/internal/dice"
	"github.com/KirkDiggler/ronnied/internal/models"
	drinkLedgerRepo "github.com/KirkDiggler/ronnied/internal/repositories/drink_ledger"
	gameRepo "github.com/KirkDiggler/ronnied/internal/repositories/game"
	playerRepo "github.com/KirkDiggler/ronnied/internal/repositories/player"
)

// GameStatus represents the current state of a game
type GameStatus string

const (
	// GameStatusWaiting indicates a game is waiting for players to join
	GameStatusWaiting GameStatus = "waiting"

	// GameStatusActive indicates a game is in progress
	GameStatusActive GameStatus = "active"

	// GameStatusRollOff indicates a game is in a roll-off state
	GameStatusRollOff GameStatus = "roll_off"

	// GameStatusCompleted indicates a game has been completed
	GameStatusCompleted GameStatus = "completed"
)

// DrinkReason represents why a drink was assigned
type DrinkReason string

const (
	// DrinkReasonCriticalHit indicates a drink assigned due to rolling a critical hit (6)
	DrinkReasonCriticalHit DrinkReason = "critical_hit"

	// DrinkReasonCriticalFail indicates a drink assigned due to rolling a critical fail (1)
	DrinkReasonCriticalFail DrinkReason = "critical_fail"

	// DrinkReasonLowestRoll indicates a drink assigned due to having the lowest roll
	DrinkReasonLowestRoll DrinkReason = "lowest_roll"
)

// RollOffType represents the type of roll-off
type RollOffType string

const (
	// RollOffTypeHighest indicates a roll-off for players with the highest roll
	RollOffTypeHighest RollOffType = "highest"

	// RollOffTypeLowest indicates a roll-off for players with the lowest roll
	RollOffTypeLowest RollOffType = "lowest"
)

// Config holds configuration for the game service
type Config struct {
	// Maximum number of players per game
	MaxPlayers int

	// Number of sides on the dice
	DiceSides int

	// Value that counts as a critical hit
	CriticalHitValue int

	// Value that counts as a critical fail
	CriticalFailValue int

	// Maximum number of concurrent games
	MaxConcurrentGames int

	// Repository dependencies
	GameRepo        gameRepo.Repository
	PlayerRepo      playerRepo.Repository
	DrinkLedgerRepo drinkLedgerRepo.Repository

	// Service dependencies
	DiceRoller    dice.Roller
	Clock         clock.Clock
	UUIDGenerator uuid.UUID
}

// CreateGameInput contains parameters for creating a new game
type CreateGameInput struct {
	// ChannelID is the Discord channel ID where the game is being played
	ChannelID string

	// CreatorID is the Discord user ID of the player creating the game
	CreatorID string

	// CreatorName is the display name of the player creating the game
	CreatorName string
}

// CreateGameOutput contains the result of creating a new game
type CreateGameOutput struct {
	// GameID is the unique identifier for the created game
	GameID string
}

// JoinGameInput contains parameters for joining a game
type JoinGameInput struct {
	// GameID is the unique identifier for the game to join
	GameID string

	// PlayerID is the Discord user ID of the player joining the game
	PlayerID string

	// PlayerName is the display name of the player joining the game
	PlayerName string
}

// JoinGameOutput contains the result of joining a game
type JoinGameOutput struct {
	// Success indicates if the player successfully joined the game
	Success       bool
	AlreadyJoined bool // Indicates if the player was already in the game
}

// LeaveGameInput contains parameters for leaving a game
type LeaveGameInput struct {
	// GameID is the unique identifier for the game
	GameID string

	// PlayerID is the Discord user ID of the player
	PlayerID string
}

// LeaveGameOutput contains the result of leaving a game
type LeaveGameOutput struct {
	// Success indicates if the player successfully left the game
	Success bool
}

// RollDiceInput contains parameters for rolling dice
type RollDiceInput struct {
	// GameID is the unique identifier for the game
	GameID string

	// PlayerID is the Discord user ID of the player
	PlayerID string
}

// PlayerOption represents a player who can be selected for a drink assignment
type PlayerOption struct {
	// PlayerID is the unique identifier for the player
	PlayerID string

	// PlayerName is the display name of the player
	PlayerName string

	// IsCurrentPlayer indicates if this is the player who rolled
	IsCurrentPlayer bool
}

// RollDiceOutput contains the result of a dice roll
type RollDiceOutput struct {
	// Value is the result of the dice roll
	Value int
	
	// RollValue is an alias for Value to maintain compatibility
	RollValue int
	
	// PlayerID is the ID of the player who rolled
	PlayerID string
	
	// PlayerName is the name of the player who rolled
	PlayerName string

	// IsCriticalHit indicates if the roll was a critical hit
	IsCriticalHit bool

	// IsCriticalFail indicates if the roll was a critical fail
	IsCriticalFail bool

	// IsLowestRoll indicates if the roll was the lowest in the game
	// This will be false initially and may be updated after all players roll
	IsLowestRoll bool

	// NeedsRollOff indicates if a roll-off is needed
	NeedsRollOff bool

	// RollOffType indicates the type of roll-off needed (if any)
	RollOffType RollOffType

	// RollOffGameID is the ID of the roll-off game (if created)
	RollOffGameID string

	// AllPlayersRolled indicates if all players in the game have rolled
	AllPlayersRolled bool

	// --- Domain Result Information ---

	// Result is the primary outcome message of the roll
	Result string

	// Details provides additional context about the roll result
	Details string

	// ActiveRollOffGameID is the ID of an active roll-off game the player should participate in
	// If empty, the player should roll in the current game
	ActiveRollOffGameID string

	// EligiblePlayers is a list of players who can be assigned a drink (for critical hits)
	EligiblePlayers []PlayerOption

	// Game is the current game state
	Game *models.Game
}

// AssignDrinkInput contains parameters for assigning a drink
type AssignDrinkInput struct {
	// GameID is the unique identifier for the game
	GameID string

	// FromPlayerID is the Discord user ID of the player assigning the drink
	FromPlayerID string

	// ToPlayerID is the Discord user ID of the player receiving the drink
	ToPlayerID string

	// Reason is why the drink is being assigned
	Reason DrinkReason
}

// AssignDrinkOutput contains the result of assigning a drink
type AssignDrinkOutput struct {
	// Success indicates if the drink was successfully assigned
	Success bool

	// GameEnded indicates if the game ended as a result of this drink assignment
	GameEnded bool

	// EndGameOutput contains the result of ending the game (if applicable)
	EndGameOutput *EndGameOutput
}

// PlayerStats represents a player's statistics in a game
type PlayerStats struct {
	// PlayerID is the Discord user ID of the player
	PlayerID string

	// PlayerName is the display name of the player
	PlayerName string

	// DrinksAssigned is the number of drinks assigned to others
	DrinksAssigned int

	// DrinksReceived is the number of drinks received from others
	DrinksReceived int

	// LastRoll is the value of the player's last roll
	LastRoll int

	// LastRollTime is when the player last rolled
	LastRollTime time.Time
}

// EndGameInput contains parameters for ending a game
type EndGameInput struct {
	// GameID is the unique identifier for the game
	GameID string
}

// EndGameOutput contains the result of ending a game
type EndGameOutput struct {
	// Success indicates if the game was successfully ended
	Success bool

	// FinalLeaderboard contains the final standings for the game
	FinalLeaderboard []*PlayerStats

	// NeedsRollOff indicates if a roll-off is needed
	NeedsRollOff bool

	// RollOffGameID is the ID of the roll-off game
	RollOffGameID string

	// RollOffType indicates the type of roll-off (highest or lowest)
	RollOffType RollOffType

	// RollOffPlayerIDs contains the IDs of players in the roll-off
	RollOffPlayerIDs []string
}

// StartGameInput contains parameters for starting a game
type StartGameInput struct {
	// GameID is the unique identifier for the game
	GameID string

	// PlayerID is the Discord user ID of the player starting the game
	// This should match the game's creator ID
	PlayerID string
}

// StartGameOutput contains the result of starting a game
type StartGameOutput struct {
	// Success indicates if the game was successfully started
	Success bool
}

// HandleRollOffInput contains parameters for handling a roll-off
type HandleRollOffInput struct {
	// ParentGameID is the ID of the original game
	ParentGameID string

	// RollOffGameID is the ID of the roll-off game
	RollOffGameID string

	// PlayerIDs are the IDs of players participating in the roll-off
	PlayerIDs []string

	// Type is the type of roll-off (highest or lowest)
	Type RollOffType
}

// HandleRollOffOutput contains the result of handling a roll-off
type HandleRollOffOutput struct {
	// Success indicates if the roll-off was successfully handled
	Success bool

	// WinnerPlayerIDs contains the IDs of players who won the roll-off
	WinnerPlayerIDs []string

	// NeedsAnotherRollOff indicates if another roll-off is needed
	NeedsAnotherRollOff bool

	// NextRollOffGameID is the ID of the next roll-off game (if needed)
	NextRollOffGameID string
}

// GetGameByChannelInput defines the input for retrieving a game by channel ID
type GetGameByChannelInput struct {
	ChannelID string
}

// GetGameByChannelOutput defines the output for retrieving a game by channel ID
type GetGameByChannelOutput struct {
	Game *models.Game
}

// GetLeaderboardInput defines the input for retrieving a game's leaderboard
type GetLeaderboardInput struct {
	GameID string
}

// LeaderboardEntry represents a single entry in the leaderboard
type LeaderboardEntry struct {
	PlayerID   string
	PlayerName string
	DrinkCount int
}

// GetLeaderboardOutput defines the output for retrieving a game's leaderboard
type GetLeaderboardOutput struct {
	GameID  string
	Entries []LeaderboardEntry
}

// AbandonGameInput contains parameters for forcefully abandoning a game
type AbandonGameInput struct {
	// GameID is the unique identifier for the game
	GameID string
}

// AbandonGameOutput contains the result of abandoning a game
type AbandonGameOutput struct {
	// Success indicates if the game was successfully abandoned
	Success bool
}

// UpdateGameMessageInput contains parameters for updating a game's message ID
type UpdateGameMessageInput struct {
	// GameID is the unique identifier for the game
	GameID string
	
	// MessageID is the Discord message ID to associate with the game
	MessageID string
}

// UpdateGameMessageOutput contains the result of updating a game's message ID
type UpdateGameMessageOutput struct {
	// Success indicates if the message ID was successfully updated
	Success bool
}

// GetGameInput defines the input for retrieving a game by ID
type GetGameInput struct {
	// GameID is the unique identifier for the game
	GameID string
}

// GetGameOutput contains the result of retrieving a game by ID
type GetGameOutput struct {
	// Game is the retrieved game
	Game *models.Game
}

// GetDrinkRecordsInput contains parameters for retrieving drink records for a game
type GetDrinkRecordsInput struct {
	GameID string
}

// GetDrinkRecordsOutput contains the result of retrieving drink records for a game
type GetDrinkRecordsOutput struct {
	Records []*models.DrinkLedger
}

// GetGameLeaderboardInput defines the input for retrieving the leaderboard for a game
type GetGameLeaderboardInput struct {
	GameID string
}

// GetGameLeaderboardOutput defines the output for retrieving the leaderboard for a game
type GetGameLeaderboardOutput struct {
	GameID  string
	Entries []LeaderboardEntry
}
