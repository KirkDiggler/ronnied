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

The roll-off functionality is implemented as a set of methods within the game orchestrator. This approach treats roll-offs as nested games linked to a parent game:

1. **StartRollOff**: Creates a new roll-off game linked to the parent game
2. **RollDice**: Handles rolling in both main games and roll-offs
3. **CompleteRollOff**: Finalizes a roll-off and updates the parent game
4. **FindActiveRollOffForPlayer**: Determines if a player is currently in a roll-off

This design allows for:
- Clean separation between main games and roll-offs
- Support for nested roll-offs (roll-offs of roll-offs)
- Reuse of existing game logic for roll-offs
- Simplified testing of roll-off functionality

## Testing Strategy

The orchestrator pattern facilitates comprehensive testing:

1. **Unit Tests**: Test each method in isolation with mocked dependencies
2. **Integration Tests**: Test the interaction between different components
3. **End-to-End Tests**: Test complete workflows from start to finish

## Migration Plan

The migration from the current service-based approach to the orchestrator pattern will be done incrementally:

1. Create the orchestrator interface and basic implementation
2. Migrate one method at a time from the service to the orchestrator
3. Update tests to use the orchestrator
4. Refactor the service to use the orchestrator
5. Remove duplicate code from the service

This approach ensures that the system remains functional throughout the migration process.
