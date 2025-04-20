package discord

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/KirkDiggler/ronnied/internal/models"
	"github.com/KirkDiggler/ronnied/internal/services/game"
	"github.com/KirkDiggler/ronnied/internal/services/messaging"
	"github.com/bwmarrin/discordgo"
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
		return b.handlePayDrinkButton(s, i)
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
		Emoji: discordgo.ComponentEmoji{
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
		GameID:     existingGame.Game.ID,
		PlayerID:   userID,
		ForceStart: true, // Always try to force start, service layer will decide if it's allowed
	})
	if err != nil {
		log.Printf("Error starting game: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Failed to start game: %v", err))
	}

	if !startOutput.Success {
		return RespondWithEphemeralMessage(s, i, "Failed to start the game. Make sure you are the creator of the game.")
	}

	// If the game was force-started, add a metadata field to the game
	if startOutput.ForceStarted && startOutput.CreatorName != "" {
		// Create a special message for the shared game message
		forceStartMsg := fmt.Sprintf("⚠️ Game force-started by %s! %s took too long to start the game and has been assigned a drink.", 
			s.State.User.Username, startOutput.CreatorName)
		
		// Update the game message with the force-start information
		b.updateGameMessageWithForceStart(s, channelID, existingGame.Game.ID, forceStartMsg)
	} else {
		// Update the game message normally
		b.updateGameMessage(s, channelID, existingGame.Game.ID)
	}

	// Create roll button
	rollButton := discordgo.Button{
		Label:    "Roll Dice",
		Style:    discordgo.PrimaryButton,
		CustomID: ButtonRollDice,
		Emoji: discordgo.ComponentEmoji{
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
	
	// If the game was force-started, add information about the original creator
	if startOutput.ForceStarted && startOutput.CreatorName != "" {
		gameStartedMessage = fmt.Sprintf("Game force-started! %s took too long to start the game and has been assigned a drink. Click the button below to roll your dice.", startOutput.CreatorName)
	} else if err == nil {
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

	// First, acknowledge the interaction with a deferred update
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
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
			_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "No active game found in this channel.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return err
		}
		log.Printf("Error getting game: %v", err)
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("Error getting game: %v", err),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return err
	}

	// Check if there's an active roll-off game for this player
	rollOffGame, rollOffErr := b.gameService.FindActiveRollOffGame(ctx, userID, existingGame.Game.ID)
	if rollOffErr == nil && rollOffGame != nil {
		// Player should be rolling in the roll-off game instead
		log.Printf("Player %s should be rolling in roll-off game %s instead of main game %s", 
			userID, rollOffGame.ID, existingGame.Game.ID)
		
		// Roll the dice in the roll-off game
		rollOutput, err := b.gameService.RollDice(ctx, &game.RollDiceInput{
			GameID:   rollOffGame.ID,
			PlayerID: userID,
		})
		
		if err != nil {
			_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: fmt.Sprintf("Failed to roll in roll-off game: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return err
		}
		
		// Update both the roll-off game message and the main game message
		b.updateGameMessage(s, channelID, rollOffGame.ID)
		b.updateGameMessage(s, channelID, existingGame.Game.ID)
		
		// Create a response for the player
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("⚔️ **ROLL-OFF ROLL!** You rolled a **%d** in the tie-breaker! Check the game message to see if you won the roll-off.", rollOutput.RollValue),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return err
	}

	// Roll the dice in the main game
	rollOutput, err := b.gameService.RollDice(ctx, &game.RollDiceInput{
		GameID:   existingGame.Game.ID,
		PlayerID: userID,
	})

	// Handle errors
	if err != nil {
		// Check if we need to redirect to a roll-off game
		if errors.Is(err, game.ErrPlayerNotInGame) || errors.Is(err, game.ErrPlayerAlreadyRolled) || errors.Is(err, game.ErrInvalidGameState) {
			// Check again for an active roll-off game (in case one was created between our earlier check and now)
			rollOffGame, rollOffErr := b.gameService.FindActiveRollOffGame(ctx, userID, existingGame.Game.ID)
			if rollOffErr == nil && rollOffGame != nil {
				// Player should be rolling in a roll-off game
				_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "You need to roll in a roll-off game! Use the Roll button on the game message to continue.",
					Flags:   discordgo.MessageFlagsEphemeral,
				})
				
				// Update the game message to make the roll-off more visible
				b.updateGameMessage(s, channelID, existingGame.Game.ID)
				return err
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
			_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "You are not part of this game.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return err
		default:
			// For any other error, just return the error message
			_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: fmt.Sprintf("Failed to roll dice: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return err
		}

		// Get a friendly error message from the messaging service
		errorMsgOutput, msgErr := b.messagingService.GetErrorMessage(ctx, &messaging.GetErrorMessageInput{
			ErrorType: errorType,
		})
		if msgErr != nil {
			// If messaging service fails, use a generic message
			_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: fmt.Sprintf("Failed to roll dice: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return err
		}
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: errorMsgOutput.Message,
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return err
	}

	// Update the game message in the channel
	b.updateGameMessage(s, channelID, existingGame.Game.ID)

	// Check if the player should be redirected to a roll-off game
	if rollOutput.ActiveRollOffGameID != "" {
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "⚔️ **ROLL-OFF REQUIRED!** You need to roll again to break the tie. Use the Roll button to continue.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return err
	}

	// Determine color based on roll result
	var embedColor int
	if rollOutput.IsCriticalHit {
		embedColor = 0x2ecc71 // Green for critical hits
	} else if rollOutput.RollValue == 1 {
		embedColor = 0xe74c3c // Red for critical fails
	} else {
		embedColor = 0x3498db // Blue for normal rolls
	}

	// Get a dynamic roll result message from the messaging service
	rollResultOutput, err := b.messagingService.GetRollResultMessage(ctx, &messaging.GetRollResultMessageInput{
		PlayerName:        rollOutput.PlayerName,
		RollValue:         rollOutput.RollValue,
		IsCriticalHit:     rollOutput.IsCriticalHit,
		IsCriticalFail:    rollOutput.RollValue == 1, // Assuming 1 is critical fail
		IsPersonalMessage: true,                      // This is an ephemeral message to the player
	})

	// Get a supportive whisper message from Ronnie
	rollWhisperOutput, whisperErr := b.messagingService.GetRollWhisperMessage(ctx, &messaging.GetRollWhisperMessageInput{
		PlayerName:     rollOutput.PlayerName,
		RollValue:      rollOutput.RollValue,
		IsCriticalHit:  rollOutput.IsCriticalHit,
		IsCriticalFail: rollOutput.RollValue == 1, // Assuming 1 is critical fail
	})

	// Create embeds - either with messaging service output or fallback to static content
	var embeds []*discordgo.MessageEmbed
	var contentText string

	if err != nil {
		// Fallback to static description if messaging service fails
		embeds = []*discordgo.MessageEmbed{
			{
				Title:       rollOutput.Result,
				Description: rollOutput.Details,
				Color:       embedColor,
			},
		}
		contentText = rollOutput.Result
	} else {
		// Use the fun message from the messaging service
		contentText = rollResultOutput.Title

		// Create an embed with the fun message
		embed := &discordgo.MessageEmbed{
			Title:       rollResultOutput.Title,
			Description: rollResultOutput.Message,
			Color:       embedColor,
		}

		embeds = append(embeds, embed)
	}

	// Add the whisper message as a second embed if available
	if whisperErr == nil {
		whisperEmbed := &discordgo.MessageEmbed{
			Title:       "Ronnie whispers...",
			Description: rollWhisperOutput.Message,
			Color:       0x95a5a6, // Gray color for whispers
			Footer: &discordgo.MessageEmbedFooter{
				Text:    "Just between us...",
				IconURL: "https://cdn.discordapp.com/emojis/839903382661799966.png", // Optional: Add a whisper emoji
			},
		}
		embeds = append(embeds, whisperEmbed)
	}

	// Build components based on the roll result
	var components []discordgo.MessageComponent
	if rollOutput.IsCriticalHit {
		// Create player selection dropdown for critical hits
		if len(rollOutput.EligiblePlayers) > 0 {
			var playerOptions []discordgo.SelectMenuOption

			for _, player := range rollOutput.EligiblePlayers {
				playerOptions = append(playerOptions, discordgo.SelectMenuOption{
					Label:       player.PlayerName,
					Value:       player.PlayerID,
					Description: "Assign a drink to this player",
					Emoji: discordgo.ComponentEmoji{
						Name: "🍺",
					},
				})
			}

			playerSelect := discordgo.SelectMenu{
				CustomID:    SelectAssignDrink,
				Placeholder: "Select a player to drink",
				Options:     playerOptions,
			}

			components = append(components, playerSelect)
		}
	} else {
		// Create roll again button for non-critical hits
		rollButton := discordgo.Button{
			Label:    "Roll Again",
			Style:    discordgo.PrimaryButton,
			CustomID: ButtonRollDice,
			Emoji: discordgo.ComponentEmoji{
				Name: "🎲",
			},
		}

		// Add Pay Drink button
		payDrinkButton := discordgo.Button{
			Label:    "Pay Drink",
			Style:    discordgo.SuccessButton,
			CustomID: ButtonPayDrink,
			Emoji: discordgo.ComponentEmoji{
				Name: "💸",
			},
		}

		// Add both buttons
		components = append(components, rollButton, payDrinkButton)
	}

	// Create action row for components if we have any
	var messageComponents []discordgo.MessageComponent
	if len(components) > 0 {
		messageComponents = append(messageComponents, discordgo.ActionsRow{
			Components: components,
		})
	}

	// Edit the original message with the updated content
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:    &contentText,
		Embeds:     &embeds,
		Components: &messageComponents,
	})

	return err
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
		Emoji: discordgo.ComponentEmoji{
			Name: "🎲",
		},
	}

	// Create pay drink button
	payDrinkButton := discordgo.Button{
		Label:    "Pay Drink",
		Style:    discordgo.SuccessButton,
		CustomID: ButtonPayDrink,
		Emoji: discordgo.ComponentEmoji{
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
		Emoji: discordgo.ComponentEmoji{
			Name: "🎲",
		},
	}

	// Create begin button
	beginButton := discordgo.Button{
		Label:    "Begin Game",
		Style:    discordgo.PrimaryButton,
		CustomID: ButtonBeginGame,
		Emoji: discordgo.ComponentEmoji{
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
func (b *Bot) handlePayDrinkButton(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Get the user ID and channel ID
	userID := i.Member.User.ID
	channelID := i.ChannelID
	ctx := context.Background()

	// First, acknowledge the interaction with a deferred update
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
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
			_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "No active game found in this channel.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return err
		}
		log.Printf("Error getting game: %v", err)
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("Error getting game: %v", err),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return err
	}

	// Get player info
	var playerName string
	for _, participant := range existingGame.Game.Participants {
		if participant.PlayerID == userID {
			playerName = participant.PlayerName
			break
		}
	}

	// Pay the drink
	_, err = b.gameService.PayDrink(ctx, &game.PayDrinkInput{
		GameID:   existingGame.Game.ID,
		PlayerID: userID,
	})
	if err != nil {
		log.Printf("Error paying drink: %v", err)
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("Failed to pay drink: %v", err),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return err
	}

	// Update the game message in the channel to show the drink payment
	b.updateGameMessage(s, channelID, existingGame.Game.ID)

	// Get the session ID from the game's channel
	sessionOutput, err := b.gameService.GetSessionLeaderboard(ctx, &game.GetSessionLeaderboardInput{
		ChannelID: channelID,
	})

	// Calculate remaining drinks for the player
	var remainingDrinks int
	var drinkStats string
	var progressBar string
	var statusEmoji string
	var motivationalMsg string
	if err == nil && sessionOutput != nil {
		for _, entry := range sessionOutput.Entries {
			if entry.PlayerID == userID {
				remainingDrinks = entry.DrinkCount - entry.PaidCount

				// Create a detailed drink stats message
				if entry.DrinkCount > 0 {
					// Select appropriate emoji based on payment status
					if remainingDrinks == 0 {
						statusEmoji = "🎉" // Celebration emoji for all paid
						motivationalMsg = "All paid up! You're a champion!"
					} else if float64(entry.PaidCount)/float64(entry.DrinkCount) >= 0.75 {
						statusEmoji = "🔥" // Fire emoji for almost done
						motivationalMsg = "Almost there! Just a few more to go!"
					} else if float64(entry.PaidCount)/float64(entry.DrinkCount) >= 0.5 {
						statusEmoji = "👍" // Thumbs up for good progress
						motivationalMsg = "Halfway there! Keep it up!"
					} else if float64(entry.PaidCount)/float64(entry.DrinkCount) >= 0.25 {
						statusEmoji = "🍺" // Beer emoji for some progress
						motivationalMsg = "Good start! Keep those drinks flowing!"
					} else {
						statusEmoji = "💪" // Flexed arm for just starting
						motivationalMsg = "Just getting started! You can do this!"
					}

					drinkStats = fmt.Sprintf("**Drink Stats** %s\nTotal: %d | Paid: %d | Remaining: %d",
						statusEmoji, entry.DrinkCount, entry.PaidCount, remainingDrinks)

					// Create a visual progress bar
					progressBar = createProgressBar(entry.PaidCount, entry.DrinkCount)
				}
				break
			}
		}
	}

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
		Emoji: discordgo.ComponentEmoji{
			Name: "🎲",
		},
	}

	// Create pay drink button
	payDrinkButton := discordgo.Button{
		Label:    "Pay Drink",
		Style:    discordgo.SuccessButton,
		CustomID: ButtonPayDrink,
		Emoji: discordgo.ComponentEmoji{
			Name: "💸",
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

	// Add remaining drinks to the embed description
	if remainingDrinks > 0 {
		if len(embeds) > 0 {
			embeds[0].Description += fmt.Sprintf("\nYou still owe %d drinks!", remainingDrinks)
		} else {
			contentText += fmt.Sprintf("\nYou still owe %d drinks!", remainingDrinks)
		}
	} else {
		if len(embeds) > 0 {
			embeds[0].Description += "\nYou've paid all your drinks!"
		} else {
			contentText += "\nYou've paid all your drinks!"
		}
	}

	// Add drink stats to the embed description
	if drinkStats != "" {
		if len(embeds) > 0 {
			embeds[0].Description += "\n" + drinkStats
		} else {
			contentText += "\n" + drinkStats
		}
	}

	// Add progress bar to the embed description
	if progressBar != "" {
		if len(embeds) > 0 {
			embeds[0].Description += "\n" + progressBar
		} else {
			contentText += "\n" + progressBar
		}
	}

	// Add motivational message to the embed description
	if motivationalMsg != "" {
		if len(embeds) > 0 {
			embeds[0].Description += "\n" + motivationalMsg
		} else {
			contentText += "\n" + motivationalMsg
		}
	}

	// Create message components
	messageComponents := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				rollButton,
				payDrinkButton,
			},
		},
	}

	// Edit the original message with the updated content
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:    &contentText,
		Embeds:     &embeds,
		Components: &messageComponents,
	})

	return err
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

// updateGameMessageWithForceStart updates the main game message in the channel with force-start information
func (b *Bot) updateGameMessageWithForceStart(s *discordgo.Session, channelID string, gameID string, forceStartMsg string) {
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

	// Add the force-start message to the game message
	if messageEdit.Embeds != nil && len(messageEdit.Embeds) > 0 {
		messageEdit.Embeds[0].Description = forceStartMsg + "\n\n" + messageEdit.Embeds[0].Description
	} else if messageEdit.Content != nil {
		// Create a new content string with the force-start message
		newContent := forceStartMsg + "\n\n" + *messageEdit.Content
		messageEdit.Content = &newContent
	} else {
		// If there's no content, create a new one
		messageEdit.Content = &forceStartMsg
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
