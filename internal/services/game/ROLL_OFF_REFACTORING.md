# Roll-Off Functionality Refactoring

This document outlines the plan for simplifying the roll-off functionality in the Ronnied Discord bot game service.

## Current Implementation Issues

The current roll-off implementation has several problems:

1. **Separate Game Entities**: Roll-offs are implemented as separate game entities with their own IDs, participants, and states
2. **Complex State Management**: Managing the relationship between main games and roll-off games is difficult
3. **Confusing User Experience**: Players don't clearly understand when they're in a roll-off vs. the main game
4. **Fragile Chain Management**: Parent-child relationships between games can break or become inconsistent
5. **Edge Case Handling**: Multiple simultaneous roll-offs (highest and lowest) create complex state transitions

## Proposed Solution

We will simplify roll-offs by integrating them as a state within the main game rather than as separate game entities.

### Key Changes

1. **Unified Game Model**:
   - Add roll-off state directly to the main game model
   - Track roll-off participants within the main game
   - Store roll-off results in the main game

2. **State Management**:
   - Add a new `RollOffState` field to the game model
   - Define clear state transitions between regular gameplay and roll-offs
   - Ensure the game can only be in one roll-off state at a time

3. **Participant Tracking**:
   - Add a `ParticipantStatus` field to track who is in a roll-off
   - Track which participants have completed their roll-off rolls

4. **Roll-Off Resolution**:
   - Simplify the logic for determining roll-off winners
   - Automatically assign drinks based on roll-off results
   - Transition back to the main game state after roll-off completion

## Implementation Plan

### 1. Model Changes

```go
// Add to models.GameStatus
const (
    GameStatusRollOffHighest GameStatus = "roll_off_highest"
    GameStatusRollOffLowest  GameStatus = "roll_off_lowest"
)

// Add to models.ParticipantStatus
const (
    ParticipantStatusInRollOff      ParticipantStatus = "in_roll_off"
    ParticipantStatusRolledInRollOff ParticipantStatus = "rolled_in_roll_off"
)

// Add to models.Game
type Game struct {
    // Existing fields...

    // Roll-off related fields
    RollOffPlayerIDs []string          // Players participating in the current roll-off
    RollOffType      game.RollOffType  // Type of roll-off (highest or lowest)
    RollOffRound     int               // Track multiple rounds of roll-offs if needed
}
```

### 2. Service Interface Changes

We'll simplify the interface by removing the separate roll-off game management methods and integrating roll-off handling into the main game flow:

```go
// Remove these methods
FindActiveRollOffGame(ctx context.Context, playerID string, mainGameID string) (*models.Game, error)
HandleRollOff(ctx context.Context, input *HandleRollOffInput) (*HandleRollOffOutput, error)

// Add these methods
StartRollOff(ctx context.Context, input *StartRollOffInput) (*StartRollOffOutput, error)
CompleteRollOff(ctx context.Context, input *CompleteRollOffInput) (*CompleteRollOffOutput, error)
```

### 3. Implementation Changes

#### RollDice Method

The `RollDice` method will be updated to:
- Check if the game is in a roll-off state
- Validate if the player is part of the roll-off
- Process the roll appropriately based on game state
- Determine if all roll-off participants have rolled
- Automatically resolve the roll-off when all participants have rolled

#### Roll-Off Resolution

When all roll-off participants have rolled:
1. Determine winners based on roll-off type (highest or lowest)
2. Assign drinks as needed
3. Update participant statuses
4. Transition the game back to active state or to another roll-off if needed

### 4. Repository Changes

The repository layer will be simplified by removing roll-off game creation and management methods:

```go
// Remove these methods
CreateRollOffGame(ctx context.Context, input *CreateRollOffGameInput) (*CreateRollOffGameOutput, error)
GetRollOffGames(ctx context.Context, input *GetRollOffGamesInput) (*GetRollOffGamesOutput, error)
```

## Benefits

1. **Simplified State Management**: Roll-offs are now part of the main game flow
2. **Improved User Experience**: Clearer indication of roll-off state
3. **Reduced Complexity**: No need to manage separate game entities
4. **More Robust**: Fewer opportunities for inconsistent state
5. **Easier to Extend**: Adding new roll-off types or rules will be simpler

## Migration Strategy

1. Implement the new roll-off handling in the service layer
2. Update the handler layer to work with the new implementation
3. Add a migration path for any existing games with separate roll-off games
4. Test thoroughly with various game scenarios
5. Deploy the changes

## Potential Challenges

1. **Backward Compatibility**: Ensuring existing games can be migrated
2. **UI Updates**: The handler layer will need significant updates
3. **Edge Cases**: Handling complex scenarios like multiple roll-offs
