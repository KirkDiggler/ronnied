# Game Service

The Game Service manages the core game logic for the Ronnied dice drinking game.

## File Structure

- `types.go` - Contains all data structures and request/response types
- `interface.go` - Defines the service interface
- `service.go` - Implements the service functionality

## Responsibilities

1. **Game Session Management**
   - Create and track game sessions by Discord channel
   - Handle player joining and leaving
   - Maintain game state (waiting, active, completed)

2. **Game Mechanics**
   - Process player dice rolls
   - Apply game rules (critical hits/fails)
   - Track turn order if applicable
   - Handle roll-offs for ties (both highest and lowest rolls)

3. **Drink Assignment**
   - Maintain a ledger of drink assignments between players
   - Process drink assignments from critical hits (6)
   - Process drink assignments from critical fails (1)
   - Process drink assignments for lowest rolls
   - Track drink history with timestamps and reasons

4. **Leaderboard**
   - Generate current standings
   - Track statistics (most drinks assigned/received)
   - Provide game summary information

## Configuration

```go
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
}
```

## Dependencies

The Game Service depends on:
- Player Repository - For storing player information and drink tallies
- Game Repository - For persisting game state
- Dice functionality - For generating random dice rolls

## Interface

```go
// Service defines the interface for game operations
type Service interface {
    // CreateGame creates a new game session in a Discord channel
    CreateGame(ctx context.Context, input *CreateGameInput) (*CreateGameOutput, error)
    
    // JoinGame adds a player to an existing game
    JoinGame(ctx context.Context, input *JoinGameInput) (*JoinGameOutput, error)
    
    // LeaveGame removes a player from a game
    LeaveGame(ctx context.Context, input *LeaveGameInput) (*LeaveGameOutput, error)
    
    // RollDice performs a dice roll for a player
    RollDice(ctx context.Context, input *RollDiceInput) (*RollDiceOutput, error)
    
    // AssignDrink records that one player has assigned a drink to another
    AssignDrink(ctx context.Context, input *AssignDrinkInput) (*AssignDrinkOutput, error)
    
    // GetLeaderboard returns the current standings for a game
    GetLeaderboard(ctx context.Context, input *GetLeaderboardInput) (*GetLeaderboardOutput, error)
    
    // ResetGame clears all drink assignments in a game
    ResetGame(ctx context.Context, input *ResetGameInput) (*ResetGameOutput, error)
    
    // EndGame concludes a game session
    EndGame(ctx context.Context, input *EndGameInput) (*EndGameOutput, error)
    
    // HandleRollOff manages roll-offs for tied players
    HandleRollOff(ctx context.Context, input *HandleRollOffInput) (*HandleRollOffOutput, error)
}
```

## Data Models

The service will work with these primary data structures:

1. **Game** - Represents a game session
   - ID
   - Channel ID
   - Players
   - Status (waiting, active, roll-off, completed)
   - Parent Game ID (for roll-offs)
   - Creation time
   - Last activity time

2. **Player** - Represents a participant
   - ID (Discord user ID)
   - Name
   - Current game ID
   - Last roll

3. **Roll** - Represents a dice roll
   - Value
   - Player ID
   - Game ID
   - Timestamp
   - Is critical hit/fail
   - Is lowest roll

4. **DrinkLedger** - Records drink assignments
   - ID
   - From Player ID
   - To Player ID
   - Game ID
   - Reason (critical hit, critical fail, lowest roll)
   - Timestamp

5. **Leaderboard** - Game standings
   - Player rankings
   - Drink tallies (given and received)
   - Statistics

## Implementation Notes

- Games are identified by Discord channel ID
- Only one active game per channel
- Players can only be in one game at a time
- Critical hit (6) allows assigning a drink
- Critical fail (1) results in taking a drink
- Lowest roll also results in taking a drink
- After all players roll, check for ties:
  - Tied highest rolls trigger a winner roll-off
  - Tied lowest rolls trigger a loser roll-off
- Roll-offs can be nested if ties persist
- Drink ledger provides detailed history of all drink assignments
- Consider timeout for inactive games
