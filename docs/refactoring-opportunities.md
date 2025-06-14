# Refactoring Opportunities

This document outlines potential refactoring opportunities for the Ronnied Discord bot to improve separation of concerns and reduce leaky abstractions.

## Handler Layer Refactoring

The following areas in the handler layer contain business logic that could be moved to the service layer:

### Error Handling and Mapping
- Error type mapping in `handleJoinGameButton`
- Error message parsing in `handleRollDiceButton`

### Game State Validation
- Game state checks in `handleRollDiceButton`
- Participant eligibility validation in various handlers

### Player Information Extraction
- Player name extraction from game data in `handleAssignDrinkSelect`

### UI Component Creation
- Button and component creation directly in handlers

## Implementation Recommendations

1. **Create Service Methods for Validation**
   - Move game state validation to the game service
   - Add methods like `CanPlayerRoll(gameID, playerID)` or `IsPlayerInRollOff(gameID, playerID)`

2. **Enhance Error Types**
   - Define more specific error types in the game service
   - Include necessary context in the errors (like roll-off game IDs)

3. **Add Player Information Methods**
   - Create helper methods in the game service to get player information

4. **Move UI Logic to Rendering Layer**
   - Expand the rendering functions to handle all UI components

## Benefits

- Cleaner handler code focused on coordination
- Better separation of concerns
- Reduced duplication
- More testable components
- Easier maintenance and extension

This refactoring should be approached incrementally, starting with the most critical areas first.
