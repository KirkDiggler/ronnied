# Ronnied - Discord Dice Drinking Game Bot

Ronnied is a Discord bot written in Go that manages a dice rolling drinking game. Players join a game, roll dice, and based on the results, drinks are assigned.

## Game Rules

- Players join a game session
- Each player rolls a dice
- Rolling a 6 (critical hit): Assign a drink to another player
- Rolling a 1 (critical fail): Take a drink
- After each round, a leaderboard shows who owes drinks

## Project Structure

The project follows a three-layer architecture:

1. **Handler Layer**: Manages Discord interactions and commands
   - Handles Discord events and bot setup
   - Processes user commands

2. **Service Layer**: Contains the core game logic
   - Manages game sessions and player interactions
   - Handles dice rolling mechanics and results

3. **Repository Layer**: Manages data persistence
   - Stores player information and drink tallies

## Setup and Installation

1. Clone this repository
2. Install dependencies: `go mod tidy`
3. Create a `.env` file with your Discord bot token
4. Run the bot: `go run main.go`

## Commands

- `/join`: Join the current game session
- `/roll`: Roll your dice
- `/drinks`: View the current drink tally
- `/leaderboard`: Display the leaderboard
- `/reset`: Reset the current game session

## Development Roadmap

- Basic dice rolling functionality
- Drink assignment system
- Persistent leaderboards
- Game session management
- Additional game modes
