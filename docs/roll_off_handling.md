# Roll-Off Handling in Ronnied

## Current Behavior

Currently, the game handles roll-offs in the following way:

1. When a game ends, we check for ties with the highest roll first
   - If multiple players tie for the highest roll, a roll-off is created
   - The main game waits for this roll-off to complete before proceeding

2. If there are no ties for the highest roll, we check for ties with the lowest roll
   - If multiple players tie for the lowest roll, a roll-off is created
   - The main game waits for this roll-off to complete before proceeding

3. Roll-offs can be nested (roll-offs of roll-offs) when ties continue to occur

## Edge Cases and Improvements

### Edge Case: Everyone Has the Same Roll

If all players roll the same value, we currently create a roll-off for the highest roll. This is technically correct, but it means everyone is in the roll-off, which might be confusing to players.

### Edge Case: Ties for Both Highest and Lowest

If there are exactly two distinct roll values (e.g., three players roll 6 and two players roll 1), we currently only handle the highest roll-off first. The lowest roll-off would only happen after the highest roll-off is completed.

## Proposed Improvements

1. **Parallel Roll-Offs**:
   - When a game ends, check for both highest and lowest roll ties simultaneously
   - Create both roll-offs at the same time if needed
   - Players could be in both roll-offs if there are only two distinct roll values
   - Each player would need to complete all their active roll-offs

2. **Clear Messaging**:
   - Improve messaging to clearly indicate when a player is in multiple roll-offs
   - Show which roll-off is currently active for the player
   - Provide a way to switch between roll-offs if a player is in multiple

3. **Roll-Off Prioritization**:
   - Establish a priority order for roll-offs (e.g., highest roll-offs before lowest)
   - Ensure players complete roll-offs in the correct order

4. **Special Case Handling**:
   - If everyone has the same roll, consider special handling:
     - Option 1: Skip roll-offs entirely and declare no winners/losers
     - Option 2: Still do a roll-off but with clearer messaging about why
     - Option 3: Add a "everyone drinks" rule for this rare case

5. **Roll-Off Tracking**:
   - Enhance the data model to track all active roll-offs for a player
   - Allow players to see their pending roll-offs and current status

## Implementation Considerations

1. **Data Model Changes**:
   - Add a list of active roll-off game IDs to the Player model
   - Track roll-off type (highest/lowest) in the Game model

2. **Service Layer Changes**:
   - Modify EndGame to create multiple roll-offs when needed
   - Update RollDice to check for all active roll-offs
   - Enhance FindActiveRollOffGame to handle multiple active roll-offs

3. **UI/UX Improvements**:
   - Add indicators for multiple roll-offs
   - Provide clear instructions on which roll-off to complete first

4. **Testing Scenarios**:
   - Test with all players rolling the same value
   - Test with exactly two distinct roll values
   - Test nested roll-offs (roll-offs of roll-offs)
   - Test a player being in multiple roll-offs simultaneously
