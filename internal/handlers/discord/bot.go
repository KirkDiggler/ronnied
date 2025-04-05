package discord

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/KirkDiggler/ronnied/internal/models"
	"github.com/KirkDiggler/ronnied/internal/services/game"
	"github.com/bwmarrin/discordgo"
)

// Bot represents the Discord bot instance
type Bot struct {
	session      *discordgo.Session
	commands     map[string]CommandHandler
	commandIDs   map[string]string // Maps command name to command ID
	gameService  game.Service
	config       *Config
}

// Config holds the configuration for the bot
type Config struct {
	// Discord bot token
	Token string
	
	// Application ID for the bot
	ApplicationID string
	
	// Optional guild ID for development (server-specific commands)
	GuildID string
	
	// Game service
	GameService game.Service
}

// New creates a new Discord bot
func New(cfg *Config) (*Bot, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	if cfg.Token == "" {
		return nil, errors.New("token cannot be empty")
	}

	if cfg.GameService == nil {
		return nil, errors.New("game service cannot be nil")
	}

	// Create a new Discord session
	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	bot := &Bot{
		session:     session,
		commands:    make(map[string]CommandHandler),
		commandIDs:  make(map[string]string),
		gameService: cfg.GameService,
		config:      cfg,
	}

	// Register the interaction handler
	session.AddHandler(bot.handleInteraction)

	return bot, nil
}

// Start initializes the Discord connection and registers commands
func (b *Bot) Start() error {
	// Open the websocket connection to Discord
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord connection: %w", err)
	}

	// Register the ronnied command
	ronniedCmd := NewRonniedCommand(b.gameService)
	if err := b.RegisterCommand(ronniedCmd); err != nil {
		return fmt.Errorf("failed to register ronnied command: %w", err)
	}

	log.Println("Bot is now running. Press CTRL-C to exit.")
	return nil
}

// Stop gracefully shuts down the Discord connection
func (b *Bot) Stop() error {
	// Remove all commands
	appID := b.config.ApplicationID
	if appID == "" {
		appID = b.session.State.User.ID
	}
	
	guildID := ""
	if b.config.GuildID != "" {
		guildID = b.config.GuildID
	}
	
	for cmdName, cmdID := range b.commandIDs {
		if err := b.session.ApplicationCommandDelete(appID, guildID, cmdID); err != nil {
			log.Printf("Failed to delete command %s (ID: %s): %v", cmdName, cmdID, err)
		} else {
			log.Printf("Successfully deleted command %s (ID: %s)", cmdName, cmdID)
		}
	}

	return b.session.Close()
}

// RegisterCommand registers a command with Discord
func (b *Bot) RegisterCommand(cmd CommandHandler) error {
	// Register the command with Discord
	appID := b.config.ApplicationID
	if appID == "" {
		// Fall back to session user ID if application ID is not provided
		appID = b.session.State.User.ID
	}
	
	// If guild ID is provided, register command for that specific guild
	// Otherwise, register it globally
	guildID := ""
	if b.config.GuildID != "" {
		guildID = b.config.GuildID
		log.Printf("Registering command %s for guild %s", cmd.GetName(), guildID)
	} else {
		log.Printf("Registering command %s globally", cmd.GetName())
	}
	
	createdCmd, err := b.session.ApplicationCommandCreate(appID, guildID, cmd.GetCommand())
	if err != nil {
		return fmt.Errorf("failed to create command %s: %w", cmd.GetName(), err)
	}

	// Store the command handler and its ID
	b.commands[cmd.GetName()] = cmd
	b.commandIDs[cmd.GetName()] = createdCmd.ID
	log.Printf("Registered command: %s with ID: %s", cmd.GetName(), createdCmd.ID)
	
	return nil
}

// ButtonHandler defines a function type for handling button interactions
type ButtonHandler func(s *discordgo.Session, i *discordgo.InteractionCreate) error

// Button IDs
const (
	ButtonJoinGame    = "join_game"
	ButtonBeginGame   = "begin_game"
	ButtonRollDice    = "roll_dice"
	ButtonStartNewGame = "start_new_game"
	
	// Select menu custom IDs
	SelectAssignDrink = "assign_drink"
)

// handleInteraction handles Discord interactions
func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Handle different interaction types
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		// Handle slash commands
		if h, ok := b.commands[i.ApplicationCommandData().Name]; ok {
			if err := h.Handle(s, i); err != nil {
				log.Printf("Error handling command %s: %v", i.ApplicationCommandData().Name, err)
			}
		}
	case discordgo.InteractionMessageComponent:
		// Handle buttons and other components
		if err := b.handleComponentInteraction(s, i); err != nil {
			log.Printf("Error handling component interaction: %v", err)
		}
	}
}

// handleComponentInteraction handles button clicks and other component interactions
func (b *Bot) handleComponentInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Get the custom ID of the component
	customID := i.MessageComponentData().CustomID
	
	// Get channel and user info
	channelID := i.ChannelID
	userID := i.Member.User.ID
	username := i.Member.User.Username
	if i.Member.Nick != "" {
		username = i.Member.Nick
	}
	
	// Handle different button actions
	switch customID {
	case ButtonJoinGame:
		// Handle join game button
		return b.handleJoinGameButton(s, i, channelID, userID, username)
	case ButtonBeginGame:
		// Handle begin game button
		return b.handleBeginGameButton(s, i, channelID, userID)
	case ButtonRollDice:
		// Handle roll dice button
		return b.handleRollDiceButton(s, i, channelID, userID)
	case SelectAssignDrink:
		// Handle assign drink dropdown
		return b.handleAssignDrinkSelect(s, i, channelID, userID)
	case ButtonStartNewGame:
		// Handle start new game button
		return b.handleStartNewGameButton(s, i, channelID, userID, username)
	default:
		return RespondWithError(s, i, fmt.Sprintf("Unknown button: %s", customID))
	}
}

// handleJoinGameButton handles the join game button click
func (b *Bot) handleJoinGameButton(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID, username string) error {
	ctx := context.Background()
	
	// Get the game in this channel
	existingGame, err := b.gameService.GetGameByChannel(ctx, &game.GetGameByChannelInput{
		ChannelID: channelID,
	})
	
	if err != nil {
		log.Printf("Error getting game: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Error: %v", err))
	}
	
	// Join the game
	_, err = b.gameService.JoinGame(ctx, &game.JoinGameInput{
		GameID:     existingGame.Game.ID,
		PlayerID:   userID,
		PlayerName: username,
	})
	if err != nil {
		log.Printf("Error joining game: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Failed to join game: %v", err))
	}
	
	// Update the game message
	b.updateGameMessage(s, channelID, existingGame.Game.ID)
	
	// Create roll button for when the game starts
	rollButton := discordgo.Button{
		Label:    "Roll Dice",
		Style:    discordgo.PrimaryButton,
		CustomID: ButtonRollDice,
		Emoji: &discordgo.ComponentEmoji{
			Name: "🎲",
		},
	}
	
	// Respond with success message
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "You've joined the game! Wait for the creator to start the game.",
			Flags: discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{rollButton},
				},
			},
		},
	})
}

// handleBeginGameButton handles the begin game button click
func (b *Bot) handleBeginGameButton(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID string) error {
	ctx := context.Background()
	
	// Get the game in this channel
	existingGame, err := b.gameService.GetGameByChannel(ctx, &game.GetGameByChannelInput{
		ChannelID: channelID,
	})
	
	if err != nil {
		log.Printf("Error getting game: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Error: %v", err))
	}
	
	// Start the game
	startOutput, err := b.gameService.StartGame(ctx, &game.StartGameInput{
		GameID:   existingGame.Game.ID,
		PlayerID: userID,
	})
	if err != nil {
		log.Printf("Error starting game: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Failed to start game: %v", err))
	}
	
	if !startOutput.Success {
		return RespondWithEphemeralMessage(s, i, "Failed to start the game. Make sure you are the creator of the game.")
	}
	
	// Update the game message
	b.updateGameMessage(s, channelID, existingGame.Game.ID)
	
	// Create roll button
	rollButton := discordgo.Button{
		Label:    "Roll Dice",
		Style:    discordgo.PrimaryButton,
		CustomID: ButtonRollDice,
		Emoji: &discordgo.ComponentEmoji{
			Name: "🎲",
		},
	}
	
	// Send an ephemeral message to the user who started the game
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Game Started! Click the button below to roll your dice.",
			Flags: discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{rollButton},
				},
			},
		},
	})
}

// handleRollDiceButton handles the roll dice button click
func (b *Bot) handleRollDiceButton(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID string) error {
	ctx := context.Background()
	
	// Get the game in this channel
	existingGame, err := b.gameService.GetGameByChannel(ctx, &game.GetGameByChannelInput{
		ChannelID: channelID,
	})
	
	// Handle errors or missing game
	if err != nil {
		if err == game.ErrGameNotFound {
			return RespondWithEphemeralMessage(s, i, "No active game found in this channel. Use `/ronnied start` to create a new game.")
		}
		log.Printf("Error getting game: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Error getting game: %v", err))
	}
	
	// Check if the game is in a state where players can roll
	if existingGame.Game.Status != models.GameStatusActive {
		return RespondWithEphemeralMessage(s, i, "Cannot roll dice. Game is not active.")
	}
	
	// Roll the dice
	rollOutput, err := b.gameService.RollDice(ctx, &game.RollDiceInput{
		GameID:   existingGame.Game.ID,
		PlayerID: userID,
	})
	if err != nil {
		log.Printf("Error rolling dice: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Failed to roll dice: %v", err))
	}
	
	// Build the response message based on the roll result
	var title string
	var description string
	var fields []*discordgo.MessageEmbedField
	var components []discordgo.MessageComponent
	
	// Add roll value field
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Roll Value",
		Value:  fmt.Sprintf("%d", rollOutput.Value),
		Inline: true,
	})
	
	// Handle critical hit (assign a drink)
	if rollOutput.IsCriticalHit {
		title = "Critical Hit! 🎯"
		description = "You rolled a critical hit! Select a player below to assign them a drink."
		
		// Get players for dropdown
		var playerOptions []discordgo.SelectMenuOption
		
		// Check if there are other players
		hasOtherPlayers := false
		for _, participant := range existingGame.Game.Participants {
			// Skip the current player initially
			if participant.PlayerID == userID {
				continue
			}
			
			hasOtherPlayers = true
			
			// Add player to options
			playerOptions = append(playerOptions, discordgo.SelectMenuOption{
				Label:       participant.PlayerName,
				Value:       participant.PlayerID,
				Description: "Assign a drink to this player",
				Emoji: &discordgo.ComponentEmoji{
					Name: "🍺",
				},
			})
		}
		
		// If there are no other players, include the current player
		if !hasOtherPlayers {
			// Find the current player
			for _, participant := range existingGame.Game.Participants {
				if participant.PlayerID == userID {
					playerOptions = append(playerOptions, discordgo.SelectMenuOption{
						Label:       participant.PlayerName + " (You)",
						Value:       participant.PlayerID,
						Description: "Assign a drink to yourself (no choice!)",
						Emoji: &discordgo.ComponentEmoji{
							Name: "🍺",
						},
					})
					break
				}
			}
			
			description += "\n\nYou're the only player, so you'll have to drink yourself!"
		}
		
		// Create dropdown for player selection
		if len(playerOptions) > 0 {
			playerSelect := discordgo.SelectMenu{
				CustomID:    SelectAssignDrink,
				Placeholder: "Select a player to drink",
				Options:     playerOptions,
			}
			
			components = append(components, discordgo.SelectMenu(playerSelect))
		}
	} else if rollOutput.IsCriticalFail {
		// Handle critical fail (take a drink)
		title = "Critical Fail! 🍻"
		description = "You rolled a critical fail! Take a drink."
	} else {
		// Regular roll
		title = "Dice Roll"
		description = fmt.Sprintf("You rolled a %d.", rollOutput.Value)
	}
	
	// Create action row for components if we have any
	var messageComponents []discordgo.MessageComponent
	if len(components) > 0 {
		messageComponents = append(messageComponents, discordgo.ActionsRow{
			Components: components,
		})
	}
	
	// Update the game message in the channel
	b.updateGameMessage(s, channelID, existingGame.Game.ID)
	
	// Check if all players have rolled and we need to end the game
	if rollOutput.AllPlayersRolled {
		// Get the leaderboard
		_, err := b.gameService.GetLeaderboard(ctx, &game.GetLeaderboardInput{
			GameID: existingGame.Game.ID,
		})
		if err != nil {
			log.Printf("Error getting leaderboard: %v", err)
		} else {
			// End the game if all players have rolled and no roll-off is needed
			if !rollOutput.NeedsRollOff {
				_, err := b.gameService.EndGame(ctx, &game.EndGameInput{
					GameID: existingGame.Game.ID,
				})
				if err != nil {
					log.Printf("Error ending game: %v", err)
				}
				// Update the game message one more time to show the final results
				b.updateGameMessage(s, channelID, existingGame.Game.ID)
			}
		}
	}
	
	// Respond with the roll result
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       title,
					Description: description,
					Color:       0x00ff00, // Green color
					Fields:      fields,
				},
			},
			Components: messageComponents,
			Flags:      discordgo.MessageFlagsEphemeral, // Make the message ephemeral
		},
	})
}

// handleAssignDrinkSelect handles the assign drink dropdown selection
func (b *Bot) handleAssignDrinkSelect(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID string) error {
	ctx := context.Background()
	
	// Get the selected player ID from the interaction data
	var targetPlayerID string
	if i.MessageComponentData().Values != nil && len(i.MessageComponentData().Values) > 0 {
		targetPlayerID = i.MessageComponentData().Values[0]
	}
	
	if targetPlayerID == "" {
		return RespondWithEphemeralMessage(s, i, "No player selected")
	}
	
	// Get the game in this channel
	existingGame, err := b.gameService.GetGameByChannel(ctx, &game.GetGameByChannelInput{
		ChannelID: channelID,
	})
	
	// Handle errors or missing game
	if err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			return RespondWithEphemeralMessage(s, i, "No active game found in this channel.")
		}
		log.Printf("Error getting game: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Error getting game: %v", err))
	}
	
	// Get target player name before assigning the drink
	targetPlayerName := ""
	for _, participant := range existingGame.Game.Participants {
		if participant.PlayerID == targetPlayerID {
			targetPlayerName = participant.PlayerName
			break
		}
	}
	
	// Assign the drink
	_, err = b.gameService.AssignDrink(ctx, &game.AssignDrinkInput{
		GameID:       existingGame.Game.ID,
		FromPlayerID: userID,
		ToPlayerID:   targetPlayerID,
		Reason:       game.DrinkReasonCriticalHit,
	})
	if err != nil {
		log.Printf("Error assigning drink: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Failed to assign drink: %v", err))
	}
	
	// Update the game message in the channel to show the drink assignment
	b.updateGameMessage(s, channelID, existingGame.Game.ID)
	
	// Create roll button for the next round if needed
	rollButton := discordgo.Button{
		Label:    "Roll Dice",
		Style:    discordgo.PrimaryButton,
		CustomID: ButtonRollDice,
		Emoji: &discordgo.ComponentEmoji{
			Name: "🎲",
		},
	}
	
	// Respond with success message and include the roll button
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("You assigned a drink to %s! 🍻", targetPlayerName),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{rollButton},
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

// handleStartNewGameButton handles the start new game button click
func (b *Bot) handleStartNewGameButton(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID, username string) error {
	ctx := context.Background()
	
	// Check if there's an existing game in this channel
	existingGame, err := b.gameService.GetGameByChannel(ctx, &game.GetGameByChannelInput{
		ChannelID: channelID,
	})
	
	// Only allow creating a new game if no game exists or the existing game is completed
	if err == nil && existingGame != nil && existingGame.Game != nil {
		if existingGame.Game.Status != models.GameStatusCompleted {
			return RespondWithEphemeralMessage(s, i, "There's already an active game in this channel. Use `/ronnied abandon` if you want to abandon the current game.")
		}
	}
	
	// Create a new game
	createOutput, err := b.gameService.CreateGame(ctx, &game.CreateGameInput{
		ChannelID:   channelID,
		CreatorID:   userID,
		CreatorName: username,
	})
	if err != nil {
		log.Printf("Error creating game: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Failed to create game: %v", err))
	}
	
	// Join the creator to the game
	_, err = b.gameService.JoinGame(ctx, &game.JoinGameInput{
		GameID:     createOutput.GameID,
		PlayerID:   userID,
		PlayerName: username,
	})
	if err != nil {
		log.Printf("Error joining game: %v", err)
		// Not critical, continue
	}
	
	// Create join button
	joinButton := discordgo.Button{
		Label:    "Join Game",
		Style:    discordgo.SuccessButton,
		CustomID: ButtonJoinGame,
		Emoji: &discordgo.ComponentEmoji{
			Name: "🎲",
		},
	}
	
	// Create begin button
	beginButton := discordgo.Button{
		Label:    "Begin Game",
		Style:    discordgo.PrimaryButton,
		CustomID: ButtonBeginGame,
		Emoji: &discordgo.ComponentEmoji{
			Name: "🎮",
		},
	}
	
	// Create the embed
	embeds := []*discordgo.MessageEmbed{
		{
			Title:       "New Game Started",
			Description: "A new game has been created! Click the Join button to join the game. Once everyone has joined, the creator can click Begin to start the game.",
			Color:       0x00ff00, // Green color
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Status",
					Value:  "waiting",
					Inline: true,
				},
				{
					Name:   "Players",
					Value:  "1",
					Inline: true,
				},
				{
					Name:   "Participants",
					Value:  username,
					Inline: false,
				},
			},
		},
	}
	
	// Send the message to the channel
	msg, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds: embeds,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{joinButton, beginButton},
			},
		},
	})
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Failed to send game message: %v", err))
	}
	
	// Update the game with the message ID
	_, err = b.gameService.UpdateGameMessage(ctx, &game.UpdateGameMessageInput{
		GameID:    createOutput.GameID,
		MessageID: msg.ID,
	})
	if err != nil {
		log.Printf("Error updating game message: %v", err)
		// Not critical, continue
	}
	
	// Respond with an ephemeral message
	return RespondWithEphemeralMessage(s, i, "New game created! You've been automatically added as the first player.")
}

// updateGameMessage updates the main game message in the channel
func (b *Bot) updateGameMessage(s *discordgo.Session, channelID string, gameID string) {
	ctx := context.Background()
	
	// Get the game
	gameOutput, err := b.gameService.GetGame(ctx, &game.GetGameInput{
		GameID: gameID,
	})
	if err != nil {
		log.Printf("Error getting game for message update: %v", err)
		return
	}
	
	if gameOutput.Game.MessageID == "" {
		log.Printf("Game has no message ID, cannot update")
		return
	}
	
	// Create fields for the message
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Status",
			Value:  string(gameOutput.Game.Status),
			Inline: true,
		},
		{
			Name:   "Players",
			Value:  fmt.Sprintf("%d", len(gameOutput.Game.Participants)),
			Inline: true,
		},
	}
	
	// Add player names and their rolls if the game is active or completed
	if gameOutput.Game.Status == models.GameStatusActive || 
	   gameOutput.Game.Status == models.GameStatusRollOff || 
	   gameOutput.Game.Status == models.GameStatusCompleted {
		
		// Create a field for player rolls
		playerRolls := ""
		for _, participant := range gameOutput.Game.Participants {
			rollInfo := "Not rolled yet"
			if participant.RollValue > 0 {
				rollInfo = fmt.Sprintf("%d", participant.RollValue)
				
				// Add emoji indicators for critical hits and fails
				if participant.RollValue == 6 { // Assuming 6 is critical hit
					rollInfo += " 🎯"
				} else if participant.RollValue == 1 { // Assuming 1 is critical fail
					rollInfo += " 🍻"
				}
			}
			playerRolls += fmt.Sprintf("**%s**: %s\n", participant.PlayerName, rollInfo)
		}
		
		if playerRolls != "" {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Player Rolls",
				Value:  playerRolls,
				Inline: false,
			})
		}
		
		// Get the drink assignments
		leaderboardOutput, err := b.gameService.GetLeaderboard(ctx, &game.GetLeaderboardInput{
			GameID: gameID,
		})
		
		if err == nil && len(leaderboardOutput.Entries) > 0 {
			// Create a field for drink assignments
			drinkAssignments := ""
			for _, entry := range leaderboardOutput.Entries {
				if entry.DrinkCount > 0 {
					drinkAssignments += fmt.Sprintf("**%s**: %d drink(s)\n", entry.PlayerName, entry.DrinkCount)
				}
			}
			
			if drinkAssignments != "" {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:   "Drink Assignments",
					Value:  drinkAssignments,
					Inline: false,
				})
			}
		}
		
		// If the game is completed, add the final drink leaderboard
		if gameOutput.Game.Status == models.GameStatusCompleted {
			// Get the leaderboard
			leaderboardOutput, err := b.gameService.GetLeaderboard(ctx, &game.GetLeaderboardInput{
				GameID: gameID,
			})
			
			if err == nil && len(leaderboardOutput.Entries) > 0 {
				drinkLeaderboard := ""
				for _, entry := range leaderboardOutput.Entries {
					drinkLeaderboard += fmt.Sprintf("**%s**: %d drink(s)\n", entry.PlayerName, entry.DrinkCount)
				}
				
				if drinkLeaderboard != "" {
					fields = append(fields, &discordgo.MessageEmbedField{
						Name:   "Final Drink Tally",
						Value:  drinkLeaderboard,
						Inline: false,
					})
				}
			}
		}
	} else {
		// Just show player names for waiting games
		playerNames := ""
		for _, participant := range gameOutput.Game.Participants {
			playerNames += participant.PlayerName + "\n"
		}
		
		if playerNames != "" {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Participants",
				Value:  playerNames,
				Inline: false,
			})
		}
	}
	
	// Create the embed
	var title, description string
	var components []discordgo.MessageComponent
	
	if gameOutput.Game.Status == models.GameStatusWaiting {
		title = "Game Waiting for Players"
		description = "Click the Join button to join the game. Once everyone has joined, the creator can click Begin to start the game."
		
		// Add join and begin buttons
		joinButton := discordgo.Button{
			Label:    "Join Game",
			Style:    discordgo.SuccessButton,
			CustomID: ButtonJoinGame,
			Emoji: &discordgo.ComponentEmoji{
				Name: "🎲",
			},
		}
		
		beginButton := discordgo.Button{
			Label:    "Begin Game",
			Style:    discordgo.PrimaryButton,
			CustomID: ButtonBeginGame,
			Emoji: &discordgo.ComponentEmoji{
				Name: "🎮",
			},
		}
		
		components = []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{joinButton, beginButton},
			},
		}
	} else if gameOutput.Game.Status == models.GameStatusActive {
		title = "Game in Progress"
		description = "The game has begun! Each player has received a private message with a button to roll their dice."
		
		// Count how many players have rolled
		playersRolled := 0
		for _, participant := range gameOutput.Game.Participants {
			if participant.RollValue > 0 {
				playersRolled++
			}
		}
		
		if playersRolled > 0 {
			description += fmt.Sprintf("\n\n%d out of %d players have rolled.", 
				playersRolled, len(gameOutput.Game.Participants))
		}
		
	} else if gameOutput.Game.Status == models.GameStatusRollOff {
		title = "Roll-Off in Progress"
		description = "There was a tie! Players in the roll-off have received a private message to roll again."
		
		// Identify which players are in the roll-off
		var rollOffPlayers []string
		for _, participant := range gameOutput.Game.Participants {
			// In a roll-off, participants with Status = WaitingToRoll are part of the roll-off
			if participant.Status == models.ParticipantStatusWaitingToRoll {
				rollOffPlayers = append(rollOffPlayers, participant.PlayerName)
			}
		}
		
		if len(rollOffPlayers) > 0 {
			description += "\n\nPlayers in roll-off: " + strings.Join(rollOffPlayers, ", ")
		}
		
	} else if gameOutput.Game.Status == models.GameStatusCompleted {
		title = "Game Completed"
		description = "This game has ended. Click the button below to start a new game."
		
		// Add a new game button
		newGameButton := discordgo.Button{
			Label:    "Start New Game",
			Style:    discordgo.SuccessButton,
			CustomID: ButtonStartNewGame,
			Emoji: &discordgo.ComponentEmoji{
				Name: "🎮",
			},
		}
		
		components = []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{newGameButton},
			},
		}
	}
	
	// Create the embed
	embeds := []*discordgo.MessageEmbed{
		{
			Title:       title,
			Description: description,
			Color:       0x00ff00, // Green color
			Fields:      fields,
			Timestamp:   time.Now().Format(time.RFC3339),
		},
	}
	
	// Update the message
	_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    channelID,
		ID:         gameOutput.Game.MessageID,
		Embeds:     &embeds,
		Components: &components,
	})
	if err != nil {
		log.Printf("Error updating game message: %v", err)
	}
}
