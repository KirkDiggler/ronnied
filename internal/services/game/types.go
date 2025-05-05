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

// RollDiceOutput contains the result of rolling dice
type RollDiceOutput struct {
	// Basic roll information
	PlayerID   string
	PlayerName string
	RollValue  int
	
	// Game state information
	Game       *models.Game
	
	// Roll outcome flags
	IsCriticalHit  bool
	IsCriticalFail bool
	
	// Roll-off related information
	IsRollOffRoll bool // Was this roll in a roll-off game?
	ParentGameID  string // If this is a roll-off, what's the parent game ID?
	
	// Game state indicators
	AllPlayersRolled bool // Have all players rolled in this game?
	
	// Roll-off indicators
	NeedsRollOff  bool // Does this game need a roll-off now?
	RollOffType   RollOffType // Type of roll-off needed (if any)
	RollOffGameID string // ID of the roll-off game (if created)
	
	// Redirect indicators
	NeedsToRollInRollOff bool // Should player be rolling in a roll-off instead?
	
	// Game IDs that need UI updates
	GameIDsToUpdate []string
	
	// Player options for critical hit (who can receive a drink)
	EligiblePlayers []PlayerOption
	
	// User-friendly messages (for handler convenience)
	Result  string // Primary outcome message
	Details string // Additional context about the result
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
	// Game is the game to end
	Game *models.Game
}

// EndGameOutput contains the result of ending a game
type EndGameOutput struct {
	// Success indicates if the game was successfully ended
	Success bool

	// FinalLeaderboard contains the final standings for the game
	FinalLeaderboard []*PlayerStats

	// NeedsHighestRollOff indicates if a highest roll-off is needed
	NeedsHighestRollOff bool

	// HighestRollOffGameID is the ID of the highest roll-off game
	HighestRollOffGameID string

	// HighestRollOffPlayerIDs contains the IDs of players in the highest roll-off
	HighestRollOffPlayerIDs []string

	// NeedsLowestRollOff indicates if a lowest roll-off is needed
	NeedsLowestRollOff bool

	// LowestRollOffGameID is the ID of the lowest roll-off game
	LowestRollOffGameID string

	// LowestRollOffPlayerIDs contains the IDs of players in the lowest roll-off
	LowestRollOffPlayerIDs []string

	// Backward compatibility fields
	// NeedsRollOff indicates if a roll-off is needed (either highest or lowest)
	NeedsRollOff bool

	// RollOffType indicates the type of roll-off needed (highest or lowest)
	RollOffType RollOffType

	// RollOffGameID is the ID of the roll-off game
	RollOffGameID string

	// RollOffPlayerIDs contains the IDs of players in the roll-off
	RollOffPlayerIDs []string

	// SessionID is the ID of the session this game belongs to
	SessionID string

	// SessionLeaderboard contains the current session leaderboard
	SessionLeaderboard []LeaderboardEntry
}

// StartGameInput defines the input for starting a game
type StartGameInput struct {
	GameID     string
	PlayerID   string
	ForceStart bool // Set to true when a non-creator tries to start the game after timeout
}

// StartGameOutput contains the result of starting a game
type StartGameOutput struct {
	// Success indicates if the game was successfully started
	Success      bool
	ForceStarted bool   // Whether the game was force-started by a non-creator
	CreatorID    string // The ID of the original creator who delayed starting
	CreatorName  string // The name of the original creator
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
	DrinkCount int // Total drinks this player owes
	PaidCount  int // Number of drinks this player has paid
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

// GetPlayerTabInput contains parameters for retrieving a player's current tab
type GetPlayerTabInput struct {
	// GameID is the ID of the game to get the tab for
	GameID string

	// PlayerID is the ID of the player to get the tab for
	PlayerID string
}

// PlayerTabEntry represents a single drink in a player's tab
type PlayerTabEntry struct {
	// FromPlayerID is the ID of the player who assigned the drink
	FromPlayerID string

	// FromPlayerName is the name of the player who assigned the drink
	FromPlayerName string

	// ToPlayerID is the ID of the player who received the drink
	ToPlayerID string

	// ToPlayerName is the name of the player who received the drink
	ToPlayerName string

	// Reason is why the drink was assigned
	Reason models.DrinkReason

	// Timestamp is when the drink was assigned
	Timestamp time.Time

	// Paid indicates whether the drink has been paid (taken)
	Paid bool
}

// PlayerTab contains information about a player's drinks
type PlayerTab struct {
	// PlayerID is the ID of the player
	PlayerID string

	// PlayerName is the name of the player
	PlayerName string

	// DrinksOwed are drinks the player needs to take
	DrinksOwed []*PlayerTabEntry

	// DrinksAssigned are drinks the player has assigned to others
	DrinksAssigned []*PlayerTabEntry

	// TotalOwed is the total number of drinks the player needs to take
	TotalOwed int

	// TotalAssigned is the total number of drinks the player has assigned
	TotalAssigned int

	// NetDrinks is the net number of drinks (owed - assigned)
	NetDrinks int
}

// GetPlayerTabOutput contains the result of retrieving a player's tab
type GetPlayerTabOutput struct {
	// Tab is the player's tab information
	Tab *PlayerTab

	// Game is the game the tab is for
	Game *models.Game
}

// ResetGameTabInput contains parameters for resetting a game's drink ledger
type ResetGameTabInput struct {
	// GameID is the ID of the game to reset the tab for
	GameID string

	// ResetterID is the ID of the player who is resetting the tab
	ResetterID string

	// ArchiveRecords determines whether to archive the drink records or delete them
	// If true, records will be marked as archived but kept in the database
	// If false, records will be deleted
	ArchiveRecords bool
}

// GameTabSummary contains a summary of a game's drink ledger before reset
type GameTabSummary struct {
	// GameID is the ID of the game
	GameID string

	// ResetTime is when the tab was reset
	ResetTime time.Time

	// ResetterID is the ID of the player who reset the tab
	ResetterID string

	// ResetterName is the name of the player who reset the tab
	ResetterName string

	// Leaderboard is the leaderboard at the time of reset
	Leaderboard []LeaderboardEntry

	// TotalDrinks is the total number of drinks assigned in the game
	TotalDrinks int
}

// ResetGameTabOutput contains the result of resetting a game's drink ledger
type ResetGameTabOutput struct {
	// Success indicates whether the reset was successful
	Success bool

	// PreviousTab is a summary of the game's drink ledger before reset
	PreviousTab *GameTabSummary

	// Game is the game with the reset tab
	Game *models.Game
}

// PayDrinkInput contains parameters for paying a drink
type PayDrinkInput struct {
	// GameID is the ID of the game
	GameID string

	// PlayerID is the ID of the player paying the drink
	PlayerID string
}

// PayDrinkOutput represents the output of the PayDrink method
type PayDrinkOutput struct {
	// Success indicates whether the drink was successfully paid
	Success bool

	// Game is the game the drink was paid in
	Game *models.Game

	// DrinkRecord is the drink record that was marked as paid
	DrinkRecord *models.DrinkLedger
}

// CreateSessionInput represents the input for the CreateSession method
type CreateSessionInput struct {
	// ChannelID is the Discord channel ID for this session
	ChannelID string

	// CreatedBy is the user ID who created the session
	CreatedBy string
}

// CreateSessionOutput represents the output of the CreateSession method
type CreateSessionOutput struct {
	// Success indicates whether the session was successfully created
	Success bool

	// Session is the newly created session
	Session *models.Session
}

// GetSessionLeaderboardInput represents the input for the GetSessionLeaderboard method
type GetSessionLeaderboardInput struct {
	// ChannelID is the Discord channel ID to get the leaderboard for
	// If specified, will use the current session for this channel
	ChannelID string

	// SessionID is the specific session ID to get the leaderboard for
	// If specified, will override ChannelID
	SessionID string
}

// GetSessionLeaderboardOutput represents the output of the GetSessionLeaderboard method
type GetSessionLeaderboardOutput struct {
	// Success indicates whether the leaderboard was successfully retrieved
	Success bool

	// Session is the session this leaderboard is for
	Session *models.Session

	// Entries is the list of leaderboard entries
	Entries []LeaderboardEntry
}

// StartNewSessionInput is the input for StartNewSession
type StartNewSessionInput struct {
	ChannelID string
	CreatorID string
}

// StartNewSessionOutput is the output for StartNewSession
type StartNewSessionOutput struct {
	Success   bool
	Session   *models.Session
	SessionID string
}
