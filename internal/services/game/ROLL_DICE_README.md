# RollDice Function Logic

## Current Implementation

The current implementation of `RollDice` has complex logic for handling roll-offs:

### Main Flow
1. Validate input (gameID, playerID)
2. Get the game by ID
3. Determine if the player should be in a roll-off game:
   - If the game is NOT a roll-off game:
     - Check if there's an active roll-off game for this player
     - If found, silently redirect to that roll-off game
   - If the game IS a roll-off game:
     - Set flags to indicate it's a roll-off
     - Check if there's a nested roll-off game for this player
     - If found, silently redirect to that nested roll-off game
4. Validate game state for rolling
5. Check if player is in the game
6. Check if player has already rolled
7. Perform the roll and update player state
8. Handle critical hits/fails
9. Save the game
10. Check if all players have rolled and if the game can be ended
11. Return the roll result

### Issues with Current Implementation
1. Error handling is inconsistent - sometimes errors are returned, sometimes logged and ignored
2. Silent redirection to roll-off games can be confusing
3. Nested conditionals make the flow hard to follow
4. The function is doing too many things at once

## Proposed Simplified Logic

### Main Flow
1. Validate input
2. Get the game by ID
3. Handle roll-off redirection:
   ```go
   // Find if player should be in a roll-off game
   rollOffGame, err := s.FindActiveRollOffGame(ctx, input.PlayerID, input.GameID)
   if err != nil && !errors.Is(err, ErrRollOffGameNotFound) {
       return nil, fmt.Errorf("failed to check roll-off status: %w", err)
   }
   
   // If a roll-off game exists, use that instead
   if rollOffGame != nil {
       return s.RollDiceInGame(ctx, input, rollOffGame)
   }
   
   // Continue with normal roll in the original game
   return s.RollDiceInGame(ctx, input, game)
   ```

4. Create a helper function `RollDiceInGame` that handles the actual rolling:
   ```go
   func (s *service) RollDiceInGame(ctx context.Context, input *RollDiceInput, game *models.Game) (*RollDiceOutput, error) {
       // Check if game is in a valid state for rolling
       if !isValidGameStateForRolling(game.Status) {
           return nil, fmt.Errorf("%w: game status is %s", ErrInvalidGameState, game.Status)
       }
       
       // Find the participant in the game
       participant := game.GetParticipant(input.PlayerID)
       if participant == nil {
           return nil, ErrPlayerNotInGame
       }
       
       // Check if the participant has already rolled
       if participant.RollTime != nil {
           return nil, fmt.Errorf("player %s has already rolled in this game", participant.PlayerName)
       }
       
       // Perform the roll and handle the rest of the logic...
   }
   ```

### Benefits of New Approach
1. Clear separation of concerns
2. Explicit error handling at each step
3. No silent redirections
4. Easier to understand and maintain
5. More testable

## Questions to Discuss
1. Should we keep the automatic redirection to roll-off games or return an error?
2. How should we handle errors from downstream operations like saving drink records?
3. Should we split the function into smaller, more focused functions?
4. How should we handle the game ending logic?
