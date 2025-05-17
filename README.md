# Ronnied - Discord Dice Drinking Game Bot

Ronnied is a Discord bot written in Go that implements a dice rolling drinking game. Players join a game session, roll dice, and based on the results, drinks are assigned to players according to specific rules.

## Codebase Review and Architecture

This document provides a comprehensive review of the current codebase, focusing on architecture, identified issues, and recommended improvements.

## Game Rules

- Players join a game session in a Discord channel
- Each player rolls a dice (default: 6-sided)
- Rolling a 6 (critical hit): Player can assign a drink to another player
- Rolling a 1 (critical fail): Player must take a drink
- After all players have rolled, the player with the lowest roll takes a drink
- If there's a tie for highest or lowest roll, a roll-off occurs
- The game tracks a leaderboard of drinks owed and received
- Sessions can span multiple games to track drinking totals over time

## Architecture Overview

The project follows a clean three-layer architecture:

### 1. Handler Layer (`/internal/handlers/discord`)
- **Purpose**: Manages Discord interactions and UI rendering
- **Components**:
  - `bot.go`: Core Discord bot setup and interaction handling
  - `command.go`: Command handler interfaces and response utilities
  - `ronnied_command.go`: Implementation of game commands
  - `render.go`: UI rendering logic for game state

### 2. Service Layer (`/internal/services`)
- **Purpose**: Contains the core business logic
- **Components**:
  - `game/`: Game service implementing game rules and logic
    - `interface.go`: Service interface definition
    - `service.go`: Core implementation of game logic
    - `types.go`: Input/output types and enums
    - `errors.go`: Service-specific errors
  - `messaging/`: Handles message generation for UI

### 3. Repository Layer (`/internal/repositories`)
- **Purpose**: Data persistence and retrieval
- **Components**:
  - `game/`: Game data storage
  - `player/`: Player information storage
  - `drink_ledger/`: Drink assignment tracking

### 4. Shared Models (`/internal/models`)
- **Purpose**: Common data structures used across layers
- **Components**:
  - Game, Player, Roll, DrinkLedger, and other shared types

### 5. Utilities (`/internal/common`, `/internal/dice`)
- **Purpose**: Shared utilities and services
- **Components**:
  - `dice/`: Dice rolling functionality
  - `common/`: Shared utilities like UUID generation and time

## Current Issues and Challenges

### 1. Roll-Off Functionality Issues

The roll-off mechanism has several problems:

- **Complex State Management**: Roll-offs create nested game states that are difficult to track and render correctly
- **UI Confusion**: Players often don't understand when they're in a roll-off vs. the main game
- **Chain Management**: The parent-child relationship between games and roll-offs is not clearly maintained
- **Edge Cases**: Multiple simultaneous roll-offs (both highest and lowest) create complex state transitions
- **Incomplete Roll-Off Handling**: Some roll-offs don't properly resolve or assign drinks

### 2. Handler Layer Issues

The handler layer has accumulated business logic that should be in the service layer:

- **Game State Logic**: The handler is making decisions about game state that should be handled by the service
- **UI State Management**: Complex UI state is managed in the handler rather than being provided by the service
- **Redundant Logic**: Similar logic is duplicated across multiple handler functions
- **Error Handling Inconsistency**: Error handling varies across different handlers
- **Button/Component Logic**: Game flow decisions are made based on UI components rather than game state

### 3. Other Issues

- **Session Management**: Session functionality is incomplete and not fully integrated
- **Error Handling**: Some errors are not properly propagated or displayed to users
- **Game Cleanup**: Abandoned games can leave orphaned data
- **Message Updates**: Game message updates are sometimes inconsistent

## Detailed Component Analysis

### Game Service (`/internal/services/game`)

The game service implements the core game logic with the following key components:

- **Game Status Types**:
  - `waiting`: Game created, waiting for players to join
  - `active`: Game in progress, players are rolling
  - `roll_off`: A roll-off is in progress to break a tie
  - `completed`: Game has ended

- **Drink Reason Types**:
  - `critical_hit`: Drink assigned due to rolling a 6
  - `critical_fail`: Drink assigned due to rolling a 1
  - `lowest_roll`: Drink assigned for having the lowest roll

- **Roll-Off Types**:
  - `highest`: Roll-off for players tied with highest roll
  - `lowest`: Roll-off for players tied with lowest roll

- **Key Methods**:
  - `CreateGame`: Creates a new game session
  - `JoinGame`: Adds a player to a game
  - `StartGame`: Transitions from waiting to active
  - `RollDice`: Handles dice rolling and outcomes
  - `HandleRollOff`: Manages roll-offs for tied players
  - `AssignDrink`: Records drink assignments
  - `EndGame`: Concludes a game and processes results

### Discord Handler (`/internal/handlers/discord`)

The Discord handler manages user interactions and UI rendering:

- **Command Handling**:
  - `/ronnied start`: Creates a new game
  - `/ronnied leaderboard`: Shows session leaderboard
  - `/ronnied newsession`: Starts a new drinking session
  - `/ronnied abandon`: Abandons the current game

- **Button Interactions**:
  - `join_game`: Allows players to join
  - `begin_game`: Starts the game
  - `roll_dice`: Rolls dice for the player
  - `assign_drink`: Assigns drinks after critical hits

- **UI Components**:
  - Game status messages with embeds
  - Player action buttons
  - Leaderboards and drink tallies

## Recommended Improvements

### 1. Roll-Off Functionality Refactoring

- **Simplify Roll-Off Model**: Integrate roll-offs as a state within the main game rather than separate games
- **Improve State Transitions**: Clearly define and enforce valid state transitions
- **Enhance UI Clarity**: Make it obvious to players when they're in a roll-off
- **Consolidate Logic**: Move all roll-off handling to dedicated service methods

### 2. Handler Layer Cleanup

- **Move Business Logic to Service**: Extract game flow decisions from handlers to service layer
- **Standardize Response Handling**: Create consistent patterns for handling service responses
- **Simplify UI Updates**: Create a unified approach to updating game messages
- **Improve Error Handling**: Standardize error handling and user feedback

### 3. General Improvements

- **Complete Session Management**: Finish implementing session tracking features
- **Add Comprehensive Testing**: Increase test coverage, especially for edge cases
- **Improve Documentation**: Add more code comments and documentation
- **Enhance User Experience**: Improve message clarity and game flow

## Commands and Interactions

### Slash Commands
- `/ronnied start`: Create a new game in the current channel
- `/ronnied leaderboard`: Show the current session leaderboard
- `/ronnied newsession`: Start a new drinking session
- `/ronnied abandon`: Abandon the current game

### Button Interactions
- **Join Game**: Allows players to join a waiting game
- **Begin Game**: Starts the game when all players have joined
- **Roll Dice**: Rolls dice for the current player
- **Assign Drink**: Allows selection of a player to receive a drink after rolling a critical hit
- **Pay Drink**: Marks a drink as paid in the leaderboard

## Setup and Installation

### Prerequisites
- Go 1.16 or higher
- Redis server (for persistence)
- Discord account with developer access

### Discord Bot Setup
1. Go to the [Discord Developer Portal](https://discord.com/developers/applications)
2. Click "New Application" and give it a name (e.g., "Ronnied")
3. Navigate to the "Bot" tab and click "Add Bot"
4. Under the "TOKEN" section, click "Copy" to copy your bot token
5. Enable necessary intents (Server Members Intent, Message Content Intent)
6. Navigate to the "OAuth2" tab, then to "URL Generator"
7. Select appropriate scopes (`bot`, `applications.commands`)
8. Select required permissions
9. Use the generated URL to add the bot to your server

### Local Setup
1. Clone this repository
2. Install dependencies: `go mod tidy`
3. Create a `.env` file with appropriate configuration
4. Start Redis server
5. Run the bot: `go run cmd/bot/main.go`

## Development Roadmap

- Refactor roll-off functionality
- Clean up handler layer
- Complete session management features
- Improve error handling and user feedback
- Add comprehensive testing
- Enhance documentation
