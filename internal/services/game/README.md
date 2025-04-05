# Game Service

The Game Service manages the core game logic for the Ronnied dice drinking game.

## Game Flow

1. **Game Initiation**
   - A user initiates a new game in a Discord channel
   - Multiple games can exist in a channel, with each game represented by a message
   - Games start in "waiting" status

2. **Joining Phase**
   - Players can join the game while it's in "waiting" status
   - The game remains in this state until the initiator starts it

3. **Active Game Phase**
   - Game status changes to "active"
   - Each player rolls the dice once
   - Critical hits (6) allow assigning drinks to others
   - Critical fails (1) result in taking a drink

4. **Roll-off Phase** (if needed)
   - If there are ties for highest/lowest rolls, a roll-off occurs
   - Roll-offs are treated as sub-games with their own state
   - Players in the roll-off roll again to determine the winner/loser
   - Multiple roll-offs might be needed until ties are resolved

5. **Game Completion**
   - When all rolls and roll-offs are completed, the game ends
   - Drinks are calculated and added to the ledger
   - Leaderboard is displayed
   - Game status changes to "completed"
   - New games can be started in the channel

## File Structure

- `types.go` - Contains all data structures and request/response types
- `interface.go` - Defines the service interface
- `service.go` - Implements the service functionality

## Responsibilities

1. **Game Session Management**
   - Create and track game sessions by Discord channel
   - Handle player joining and leaving
   - Maintain game state (waiting, active, roll-off, completed)

2. **Game Mechanics**
   - Process player dice rolls
   - Apply game rules (critical hits/fails)
   - Track which players have rolled
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
    
    // Repository dependencies
    GameRepo            gameRepo.Repository
    PlayerRepo          playerRepo.Repository
    DrinkLedgerRepo     ledgerRepo.Repository
    
    // Service dependencies
    DiceRoller          *dice.Roller
    Clock               clock.Clock
    UUID                uuid.UUID
}
```

## Dependencies

The Game Service depends on:
- Player Repository - For storing player information and drink tallies
- Game Repository - For persisting game state
- Drink Ledger Repository - For tracking drink assignments
- Dice functionality - For generating random dice rolls
- Clock - For consistent time tracking
- UUID - For generating unique identifiers

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
    
    // StartGame transitions a game from waiting to active state
    StartGame(ctx context.Context, input *StartGameInput) (*StartGameOutput, error)
    
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

## Roll-Off Implementation

Roll-offs occur when multiple players tie for either the highest roll (critical hit) or the lowest roll (critical fail). The roll-off process determines which player(s) will assign or receive drinks.

### Roll-Off Design

1. **Data Structure**
   - Roll-offs are implemented as separate Game objects with their own state
   - Each roll-off has a ParentGameID linking it to the main game
   - Roll-offs have GameStatusRollOff status
   - Only the tied players participate in a roll-off

2. **Roll-Off Flow**
   - When all players have rolled in the main game, check for ties
   - If ties exist, create a roll-off game with the tied players
   - Players roll again in the roll-off
   - If another tie occurs, create another roll-off (recursive)
   - Once a winner/loser is determined, update the main game

3. **Roll-Off Types**
   - Highest Roll-Off: For critical hits (6), determines who gets to assign a drink
   - Lowest Roll-Off: For critical fails (1) or lowest rolls, determines who takes a drink

4. **User Experience**
   - From the user's perspective, roll-offs appear as part of the main game flow
   - Players don't need to explicitly join roll-offs - they're automatically included if they were tied
   - The Discord message updates to show roll-off status and participants

### Implementation Details

1. **Creating Roll-Offs**
   - When ties are detected, the RollDice method creates a roll-off game
   - The roll-off game inherits channel and creator from the parent game
   - Only tied players are added as participants to the roll-off

2. **Handling Roll-Offs**
   - The HandleRollOff method processes the results of a roll-off
   - It determines if a clear winner/loser exists or if another roll-off is needed
   - For highest roll-offs, it returns the winner(s) who can assign drinks
   - For lowest roll-offs, it assigns drinks to the loser(s)

3. **Resolving Roll-Offs**
   - When a roll-off is resolved, the roll-off game status changes to completed
   - The main game continues its flow based on roll-off results
   - Drink assignments are recorded in the drink ledger

4. **Edge Cases**
   - Players leaving during roll-offs
   - Multiple simultaneous roll-offs (both highest and lowest ties)
   - Nested roll-offs (ties within roll-offs)

## Data Models

The service will work with these primary data structures:

1. **Game** - Represents a game session
   - ID
   - Channel ID
   - Players
   - Creator ID (the user who initiated the game)
   - Status (waiting, active, roll-off, completed)
   - Parent Game ID (for roll-offs)
   - Players who have rolled
   - Creation time
   - Last activity time

2. **Player** - Represents a participant
   - ID (Discord user ID)
   - Name
   - Current game ID
   - Last roll
   - Last roll time

3. **Roll** - Represents a dice roll
   - ID
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
   - Paid status

5. **Leaderboard** - Game standings
   - Player rankings
   - Drink tallies (given and received)
   - Statistics

## Implementation Notes

- Multiple games can exist in a channel simultaneously
- Each game is represented by a message in the Discord channel
- Players can only be in one game at a time
- Critical hit (6) allows assigning a drink
- Critical fail (1) results in taking a drink
- Lowest roll also results in taking a drink
- After all players roll, check for ties:
  - Tied highest rolls trigger a winner roll-off
  - Tied lowest rolls trigger a loser roll-off
- Roll-offs are treated as sub-games with their own state
- Roll-offs can be nested if ties persist
- Drink ledger provides detailed history of all drink assignments
- Consider timeout for inactive games
