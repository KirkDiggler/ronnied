# Game Orchestrator

## Overview

The Game Orchestrator is a core component of the Ronnied Discord bot that manages all game-related operations. It provides a clean, focused interface for game management, separating the business logic from the service layer and making the codebase more maintainable and testable.

## Design Principles

1. **Single Responsibility**: The orchestrator focuses solely on game logic and operations
2. **Clean Interface**: Provides a clear, well-defined API for all game operations
3. **Dependency Injection**: All dependencies are injected for better testability
4. **Stateless Design**: The orchestrator doesn't maintain state between calls

## Key Components

### Game Orchestrator Interface

The `GameOrchestrator` interface defines all operations that can be performed on games:

```go
type GameOrchestrator interface {
    // Game Lifecycle - User-facing operations
    CreateGame(ctx context.Context, input *CreateGameInput) (*CreateGameOutput, error)
    StartGame(ctx context.Context, input *StartGameInput) (*StartGameOutput, error)
    JoinGame(ctx context.Context, input *JoinGameInput) (*JoinGameOutput, error)
    EndGame(ctx context.Context, input *EndGameInput) (*EndGameOutput, error)
    AbandonGame(ctx context.Context, input *AbandonGameInput) (*AbandonGameOutput, error)
    
    // Player Actions - What players can do in a game
    RollDice(ctx context.Context, input *RollDiceInput) (*RollDiceOutput, error)
    AssignDrink(ctx context.Context, input *AssignDrinkInput) (*AssignDrinkOutput, error)
    PayDrink(ctx context.Context, input *PayDrinkInput) (*PayDrinkOutput, error)
    
    // Game Queries - Getting information about games
    GetGame(ctx context.Context, input *GetGameInput) (*GetGameOutput, error)
    GetGameByChannel(ctx context.Context, input *GetGameByChannelInput) (*GetGameByChannelOutput, error)
    
    // Game Management - Administrative operations
    UpdateGameMessage(ctx context.Context, input *UpdateGameMessageInput) (*UpdateGameMessageOutput, error)
}
```

### Implementation

The orchestrator implementation encapsulates all the game logic:

```go
type gameOrchestrator struct {
    // Configuration
    maxPlayers         int
    diceSides          int
    criticalHitValue   int
    criticalFailValue  int
    maxConcurrentGames int

    // Dependencies
    gameRepo        gameRepo.Repository
    playerRepo      playerRepo.Repository
    drinkLedgerRepo ledgerRepo.Repository
    diceRoller      dice.Roller
    clock           clock.Clock
    uuid            uuid.Generator
}
```

## Benefits of the Orchestrator Pattern

1. **Improved Testability**: Each method can be tested in isolation
2. **Better Separation of Concerns**: Clear boundaries between different parts of the system
3. **Easier Maintenance**: Changes to one part of the system don't affect others
4. **Enhanced Readability**: Code is organized by functionality
5. **Simplified Debugging**: Issues can be isolated to specific components

## Roll-Off Implementation

The roll-off functionality is implemented as an internal detail of the game orchestrator. Roll-offs are treated as separate game entities linked to a parent game:

### Roll-Off Game Structure

1. **Same Game Model**: Roll-off games use the same `Game` model as regular games
2. **Parent Relationship**: Roll-off games have a `ParentGameID` field that links to their parent game
3. **Nested Roll-Offs**: This approach naturally supports nested roll-offs (roll-of fs of roll-offs)

### Roll-Off Management

1. **Internal Management**: Roll-offs are handled internally by the orchestrator
2. **Transparent to Users**: Players simply roll dice, and the orchestrator determines if they should roll in the main game or a roll-off
3. **Helper Functions**: The orchestrator includes helper functions to:
   - Find roll-off games by parent ID
   - Determine if a player should roll in a specific game
   - Check if all players have rolled in a roll-off

### Player Flow

1. Player calls `RollDice` on a game
2. Orchestrator checks if there are any roll-off games with that game as parent
3. If the player is a participant in a roll-off game, they roll there
4. Otherwise, they roll in the main game

This design allows for:
- Clean separation between game logic and roll-off handling
- Simplified API for clients (handlers don't need to know about roll-offs)
- Improved testability with focused test cases
- Modular approach that avoids leaky abstractions

## Testing Strategy

The orchestrator pattern facilitates comprehensive testing:

1. **Unit Tests**: Test each method in isolation with mocked dependencies
2. **Integration Tests**: Test the interaction between different components
3. **End-to-End Tests**: Test complete workflows from start to finish

## Migration Plan

The migration from the current service-based approach to the orchestrator pattern will be done incrementally:

1. Create the orchestrator interface and basic implementation
2. Implement the core game functionality
3. Add roll-off handling as an internal feature
4. Create adapters to integrate with the existing service
5. Gradually migrate handlers to use the orchestrator directly
