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

### Prerequisites
- Go 1.16 or higher
- Redis server
- Discord account with developer access

### Discord Bot Setup
1. Go to the [Discord Developer Portal](https://discord.com/developers/applications)
2. Click "New Application" and give it a name (e.g., "Ronnied")
3. Navigate to the "Bot" tab and click "Add Bot"
4. Under the "TOKEN" section, click "Copy" to copy your bot token
5. Under "Bot Permissions" or "Privileged Gateway Intents", enable the necessary intents:
   - If available, enable "Server Members Intent"
   - If available, enable "Message Content Intent"
6. Navigate to the "OAuth2" tab, then to "URL Generator"
7. In the "Scopes" section, select:
   - `bot`
   - `applications.commands`
8. In the "Bot Permissions" section, select:
   - "Send Messages"
   - "Embed Links"
   - "Read Message History"
   - "Use Slash Commands"
9. Copy the generated URL and open it in your browser to add the bot to your server

### Local Setup
1. Clone this repository
2. Install dependencies: `go mod tidy`
3. Create a `.env` file in the project root with the following content:
   ```
   # Discord Bot Configuration
   DISCORD_TOKEN=your_discord_token_here
   APPLICATION_ID=your_application_id_here
   GUILD_ID=your_guild_id_here
   
   # Redis Configuration
   REDIS_ADDR=localhost:6379
   REDIS_PASSWORD=
   
   # Game Configuration
   MAX_PLAYERS=10
   DICE_SIDES=6
   CRITICAL_HIT_VALUE=6
   CRITICAL_FAIL_VALUE=1
   ```
4. Replace `your_discord_token_here` with your bot token
5. Replace `your_application_id_here` with your application ID (found in the "General Information" tab)
6. Replace `your_guild_id_here` with your server ID (right-click on your server and select "Copy ID")
7. Start Redis server
8. Run the bot: `go run main.go`

### Development Notes
- During development, the bot will register commands only in the specified guild (faster updates)
- For production, leave the `GUILD_ID` empty to register commands globally

## Commands

- `/ronnied start`: Start a new game session
- `/ronnied join`: Join the current game
- `/ronnied roll`: Roll your dice
- `/ronnied leaderboard`: Display the drink leaderboard

## Development Roadmap

- Basic dice rolling functionality
- Drink assignment system
- Persistent leaderboards
- Game session management
- Additional game modes
