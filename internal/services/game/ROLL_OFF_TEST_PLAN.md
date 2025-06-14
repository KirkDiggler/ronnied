# Roll-Off Testing Plan

This document outlines the testing strategy for the roll-off functionality in the Ronnied Discord bot. The tests are designed to verify that the service layer correctly handles roll-off state transitions and provides the necessary information for the UI rendering.

## Test Objectives

1. Verify that roll-offs are correctly initiated when players tie
2. Ensure that only players in a roll-off can roll during that state
3. Confirm that roll-off results are correctly processed
4. Test that the game returns to the appropriate state after a roll-off
5. Verify that multiple rounds of roll-offs work correctly
6. Ensure that the output contains all necessary information for UI rendering

## Test Cases

### 1. Highest Roll-Off Flow

Tests the complete flow of a highest-value roll-off:

- Starting a roll-off with tied players
- Players rolling in the roll-off
- Verifying participant status changes
- Completing the roll-off when all players have rolled
- Returning to active game state

**Key Assertions:**
- Game status changes to `GameStatusRollOff` with `RollOffTypeHighest`
- Correct players are added to `RollOffPlayerIDs`
- Player statuses change to `ParticipantStatusInRollOff` and then `ParticipantStatusRolledInRollOff`
- Roll-off output contains `IsRollOffRoll = true` and correct `AllPlayersRolled` flag
- Game returns to `GameStatusActive` when complete

### 2. Lowest Roll-Off Flow

Tests the complete flow of a lowest-value roll-off:

- Starting a roll-off with tied players
- Players rolling in the roll-off
- Verifying participant status changes
- Completing the roll-off when all players have rolled
- Returning to active game state

**Key Assertions:**
- Game status changes to `GameStatusRollOff` with `RollOffTypeLowest`
- Correct players are added to `RollOffPlayerIDs`
- Player statuses change appropriately
- Game returns to `GameStatusActive` when complete

### 3. Multiple Round Roll-Off

Tests a scenario where players continue to tie in roll-offs:

- Starting a roll-off with tied players
- Players rolling and tying again
- Initiating a second round of roll-offs
- Verifying `RollOffRound` increments
- Eventually resolving the tie and returning to active state

**Key Assertions:**
- `RollOffRound` increments correctly
- Player statuses reset between rounds
- Game maintains roll-off state until tie is resolved

### 4. Player Not In Roll-Off

Tests error handling when a player who is not part of a roll-off attempts to roll:

- Starting a roll-off with specific players
- Having a non-roll-off player attempt to roll
- Verifying the appropriate error is returned

**Key Assertions:**
- `ErrPlayerNotInRollOff` is returned
- No state changes occur

## UI Rendering Information

The tests verify that the following information is available for UI rendering:

1. **Game Status**: The current state of the game (`GameStatusRollOff`, etc.)
2. **Roll-Off Type**: Whether it's a highest or lowest roll-off
3. **Roll-Off Players**: Which players are participating in the roll-off
4. **Roll-Off Round**: The current round number for multi-round roll-offs
5. **Player Statuses**: Whether players have rolled in the current roll-off
6. **Roll Results**: The values players have rolled

This information is essential for the handler layer to correctly render the UI, showing:

- Which players are in a roll-off
- Who needs to roll
- Who has already rolled
- What type of roll-off is happening
- What round of the roll-off we're in

## Future Test Considerations

1. **Edge Cases**:
   - Players disconnecting during roll-offs
   - Game being abandoned during a roll-off
   - Extremely long chains of roll-offs

2. **Performance Testing**:
   - Testing with maximum number of players
   - Testing with multiple concurrent roll-offs

3. **Integration Testing**:
   - End-to-end tests with the handler layer
   - Verifying UI components render correctly based on roll-off state
