package discord

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/KirkDiggler/ronnied/internal/models"
	"github.com/KirkDiggler/ronnied/internal/services/game"
	"github.com/KirkDiggler/ronnied/internal/services/messaging"
)

// Bot represents the Discord bot instance
type Bot struct {
	session          *discordgo.Session
	gameService      game.Service
	messagingService messaging.Service
	commands         map[string]CommandHandler
	commandIDs       map[string]string // Maps command name to command ID
	config           *Config
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

	// Messaging service
	MessagingService messaging.Service
}

// New creates a new Discord bot
func New(cfg *Config) (*Bot, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.Token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	if cfg.GameService == nil {
		return nil, fmt.Errorf("game service cannot be nil")
	}

	if cfg.MessagingService == nil {
		return nil, fmt.Errorf("messaging service cannot be nil")
	}

	// Create a new Discord session
	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	bot := &Bot{
		session:          session,
		gameService:      cfg.GameService,
		messagingService: cfg.MessagingService,
		commands:         make(map[string]CommandHandler),
		commandIDs:       make(map[string]string),
		config:           cfg,
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
	ButtonJoinGame     = "join_game"
	ButtonBeginGame    = "begin_game"
	ButtonRollDice     = "roll_dice"
	ButtonStartNewGame = "start_new_game"
	ButtonPayDrink     = "pay_drink"

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
	case ButtonPayDrink:
		// Handle pay drink button
		return b.handlePayDrinkButton(s, i, channelID, userID)
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
	joinOutput, err := b.gameService.JoinGame(ctx, &game.JoinGameInput{
		GameID:     existingGame.Game.ID,
		PlayerID:   userID,
		PlayerName: username,
	})
	if err != nil {
		log.Printf("Error joining game: %v", err)

		// Map the error to an error type for the messaging service
		var errorType string
		switch err {
		case game.ErrGameActive:
			errorType = "game_active"
		case game.ErrGameRollOff:
			errorType = "game_roll_off"
		case game.ErrGameCompleted:
			errorType = "game_completed"
		case game.ErrGameFull:
			errorType = "game_full"
		case game.ErrInvalidGameState:
			errorType = "invalid_game_state"
		default:
			// For any other error, just return the error message
			return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Failed to join game: %v", err))
		}

		// Get a friendly error message from the messaging service
		errorMsgOutput, msgErr := b.messagingService.GetErrorMessage(ctx, &messaging.GetErrorMessageInput{
			ErrorType: errorType,
		})
		if msgErr != nil {
			// If messaging service fails, use a generic message
			return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Failed to join game: %v", err))
		}
		return RespondWithEphemeralMessage(s, i, errorMsgOutput.Message)
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

	// Get a join game message from the messaging service
	joinMsgOutput, err := b.messagingService.GetJoinGameMessage(ctx, &messaging.GetJoinGameMessageInput{
		PlayerName:    username,
		GameStatus:    existingGame.Game.Status,
		AlreadyJoined: joinOutput.AlreadyJoined,
	})

	if err != nil {
		// Fallback message if the messaging service fails
		log.Printf("Error getting join game message: %v", err)
		joinMsgOutput = &messaging.GetJoinGameMessageOutput{
			Message: "You've joined the game!",
		}
	}

	log.Printf("Player %s joined game %s with status %s (already joined: %v)",
		username, existingGame.Game.ID, existingGame.Game.Status, joinOutput.AlreadyJoined)

	// Respond with success message
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: joinMsgOutput.Message,
			Flags:   discordgo.MessageFlagsEphemeral,
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

	// Get a dynamic game started message from the messaging service
	startMsgOutput, err := b.messagingService.GetGameStartedMessage(ctx, &messaging.GetGameStartedMessageInput{
		CreatorName: existingGame.Game.GetCreatorName(),
		PlayerCount: len(existingGame.Game.Participants),
	})

	// Default message if the messaging service fails
	gameStartedMessage := "Game Started! Click the button below to roll your dice."
	if err == nil {
		gameStartedMessage = startMsgOutput.Message
	} else {
		log.Printf("Error getting game started message: %v", err)
	}

	// Send an ephemeral message to the user who started the game
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: gameStartedMessage,
			Flags:   discordgo.MessageFlagsEphemeral,
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

	// Acknowledge the interaction immediately to prevent timeout
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error acknowledging interaction: %v", err)
		return err
	}

	// Get the game in this channel
	existingGame, err := b.gameService.GetGameByChannel(ctx, &game.GetGameByChannelInput{
		ChannelID: channelID,
	})

	// Handle errors or missing game
	if err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: stringPtr("No active game found in this channel. Use `/ronnied start` to create a new game."),
			})
			return err
		}
		log.Printf("Error getting game: %v", err)
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr(fmt.Sprintf("%v", err)),
		})
		return err
	}

	// Check if the game is in a state where players can roll
	if existingGame.Game.Status == models.GameStatusWaiting {
		log.Printf("Player %s is rolling in waiting state for game %s", userID, existingGame.Game.ID)
	}

	// For roll-offs, check if this player is eligible to roll
	if existingGame.Game.Status == models.GameStatusRollOff {
		// Check if this player is part of the roll-off
		participant := existingGame.Game.GetParticipant(userID)
		if participant == nil {
			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: stringPtr("You are not part of the current roll-off."),
			})
			return err
		}
	}

	// If the game is completed, return a specific message
	if existingGame.Game.Status == models.GameStatusCompleted {
		// Get a friendly error message from the messaging service
		errorMsgOutput, msgErr := b.messagingService.GetErrorMessage(ctx, &messaging.GetErrorMessageInput{
			ErrorType: "game_completed",
		})
		if msgErr != nil {
			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: stringPtr("This game is already completed. Start a new game to roll again."),
			})
			return err
		}
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr(errorMsgOutput.Message),
		})
		return err
	}

	// Roll the dice
	rollOutput, err := b.gameService.RollDice(ctx, &game.RollDiceInput{
		GameID:   existingGame.Game.ID,
		PlayerID: userID,
	})
	if err != nil {
		log.Printf("Error rolling dice: %v", err)

		// Check if the error is about being in a roll-off game
		if strings.Contains(err.Error(), "roll-off game") {
			// Extract the roll-off game ID from the error message
			parts := strings.Split(err.Error(), ":")
			if len(parts) > 1 {
				rollOffGameID := strings.TrimSpace(parts[1])

				// Get the roll-off game
				rollOffGameOutput, err := b.gameService.GetGame(ctx, &game.GetGameInput{
					GameID: rollOffGameID,
				})

				if err == nil && rollOffGameOutput.Game != nil {
					_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: stringPtr("You need to roll in the roll-off game. Check the game message for details."),
					})
					return err
				}
			}
		}

		// Map the error to an error type for the messaging service
		var errorType string
		switch err {
		case game.ErrGameActive:
			errorType = "game_active"
		case game.ErrGameRollOff:
			errorType = "game_roll_off"
		case game.ErrGameCompleted:
			errorType = "game_completed"
		case game.ErrInvalidGameState:
			errorType = "invalid_game_state"
		case game.ErrPlayerNotInGame:
			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: stringPtr("You are not part of this game."),
			})
			return err
		default:
			// For any other error, just return the error message
			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: stringPtr(fmt.Sprintf("Failed to roll dice: %v", err)),
			})
			return err
		}

		// Get a friendly error message from the messaging service
		errorMsgOutput, msgErr := b.messagingService.GetErrorMessage(ctx, &messaging.GetErrorMessageInput{
			ErrorType: errorType,
		})
		if msgErr != nil {
			// If messaging service fails, use a generic message
			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: stringPtr(fmt.Sprintf("Failed to roll dice: %v", err)),
			})
			return err
		}
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr(errorMsgOutput.Message),
		})
		return err
	}

	// Update the game message in the channel
	b.updateGameMessage(s, channelID, existingGame.Game.ID)

	// Check if the player should be redirected to a roll-off game
	if rollOutput.ActiveRollOffGameID != "" {
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: stringPtr("You need to roll in the roll-off game. Check the game message for details."),
		})
		return err
	}

	// Render the roll response
	return renderRollDiceResponseEdit(s, i, rollOutput, b.messagingService)
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

	// Create roll button for the next roll
	rollButton := discordgo.Button{
		Label:    "Roll Again",
		Style:    discordgo.PrimaryButton,
		CustomID: ButtonRollDice,
		Emoji: &discordgo.ComponentEmoji{
			Name: "🎲",
		},
	}

	// Get a pay drink button
	payDrinkButton := discordgo.Button{
		Label:    "Pay Drink",
		Style:    discordgo.SuccessButton,
		CustomID: ButtonPayDrink,
		Emoji: &discordgo.ComponentEmoji{
			Name: "💸",
		},
	}

	// Update the current message with a confirmation and a roll button
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("You assigned a drink to %s! 🍻", targetPlayerName),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{rollButton, payDrinkButton},
				},
			},
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
			Title:       "Ronnie D Rollem get in here!",
			Description: "Let's get ready to RRROOOLLL!.",
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

	// Acknowledge the interaction without sending a message
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}

// handlePayDrinkButton handles the pay drink button click
func (b *Bot) handlePayDrinkButton(s *discordgo.Session, i *discordgo.InteractionCreate, channelID, userID string) error {
	ctx := context.Background()

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

	// Get player info
	var playerName string
	for _, participant := range existingGame.Game.Participants {
		if participant.PlayerID == userID {
			playerName = participant.PlayerName
			break
		}
	}

	// Get the player's unpaid drinks
	drinkRecordsOutput, err := b.gameService.GetDrinkRecords(ctx, &game.GetDrinkRecordsInput{
		GameID: existingGame.Game.ID,
	})
	if err != nil {
		log.Printf("Error getting drink records: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Failed to get drink records: %v", err))
	}

	// Find the first unpaid drink assigned to this player
	var drinkToPay *models.DrinkLedger
	for _, drink := range drinkRecordsOutput.Records {
		if drink.ToPlayerID == userID && !drink.Paid {
			drinkToPay = drink
			break
		}
	}

	// Check if there are any unpaid drinks
	if drinkToPay == nil {
		return RespondWithEphemeralMessage(s, i, "You don't have any unpaid drinks to pay!")
	}

	// Pay the drink
	_, err = b.gameService.PayDrink(ctx, &game.PayDrinkInput{
		GameID:   existingGame.Game.ID,
		PlayerID: userID,
		DrinkID:  drinkToPay.ID,
	})
	if err != nil {
		log.Printf("Error paying drink: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Failed to pay drink: %v", err))
	}

	// Update the game message in the channel to show the drink payment
	b.updateGameMessage(s, channelID, existingGame.Game.ID)

	// Get a fun message for the drink payment
	payDrinkMsgOutput, err := b.messagingService.GetPayDrinkMessage(ctx, &messaging.GetPayDrinkMessageInput{
		PlayerName: playerName,
		DrinkCount: 1, // For now, we're just paying one drink at a time
	})

	// Create roll button for the next roll
	rollButton := discordgo.Button{
		Label:    "Roll Again",
		Style:    discordgo.PrimaryButton,
		CustomID: ButtonRollDice,
		Emoji: &discordgo.ComponentEmoji{
			Name: "🎲",
		},
	}

	// Create embeds for the response
	var embeds []*discordgo.MessageEmbed
	var contentText string

	if err != nil {
		// Fallback to static message if messaging service fails
		contentText = "You paid your drink!"
	} else {
		// Use the fun message from the messaging service
		contentText = payDrinkMsgOutput.Title
		
		// Create an embed with the fun message
		embed := &discordgo.MessageEmbed{
			Title:       payDrinkMsgOutput.Title,
			Description: payDrinkMsgOutput.Message,
			Color:       0x2ecc71, // Green for success
		}
		
		embeds = append(embeds, embed)
	}

	// Update the current message with a confirmation and a roll button
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    contentText,
			Embeds:     embeds,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{rollButton},
				},
			},
		},
	})
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

	// Get related data needed for rendering
	var rollOffGame, parentGame *models.Game
	var drinkRecords []*models.DrinkLedger
	var leaderboardEntries, sessionLeaderboardEntries []game.LeaderboardEntry

	// Check if this is a roll-off game
	if gameOutput.Game.Status.IsRollOff() && gameOutput.Game.ParentGameID != "" {
		// Get the parent game
		parentGameOutput, err := b.gameService.GetGame(ctx, &game.GetGameInput{
			GameID: gameOutput.Game.ParentGameID,
		})
		if err == nil {
			parentGame = parentGameOutput.Game
		}
	}

	// Check if this game has a roll-off in progress
	if gameOutput.Game.RollOffGameID != "" {
		// Get the roll-off game
		rollOffGameOutput, err := b.gameService.GetGame(ctx, &game.GetGameInput{
			GameID: gameOutput.Game.RollOffGameID,
		})
		if err == nil {
			rollOffGame = rollOffGameOutput.Game
		}
	}

	// Get drink records
	drinkRecordsOutput, err := b.gameService.GetDrinkRecords(ctx, &game.GetDrinkRecordsInput{
		GameID: gameID,
	})
	if err == nil && drinkRecordsOutput != nil {
		drinkRecords = drinkRecordsOutput.Records
	}

	// Get leaderboard for completed games
	if gameOutput.Game.Status.IsCompleted() {
		leaderboardOutput, err := b.gameService.GetLeaderboard(ctx, &game.GetLeaderboardInput{
			GameID: gameID,
		})
		if err == nil && leaderboardOutput != nil {
			leaderboardEntries = leaderboardOutput.Entries
		}

		// Get session leaderboard for completed games
		sessionOutput, err := b.gameService.GetSessionLeaderboard(ctx, &game.GetSessionLeaderboardInput{
			ChannelID: channelID,
		})
		if err == nil && sessionOutput != nil {
			sessionLeaderboardEntries = sessionOutput.Entries
		}
	}

	// Render the game message
	messageEdit, err := b.renderGameMessage(gameOutput.Game, drinkRecords, leaderboardEntries, sessionLeaderboardEntries, rollOffGame, parentGame)
	if err != nil {
		log.Printf("Error rendering game message: %v", err)
		return
	}

	// Send the message edit
	_, err = s.ChannelMessageEditComplex(messageEdit)
	if err != nil {
		log.Printf("Error updating game message: %v", err)
	}
}

// Helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}
