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
)

// Bot represents the Discord bot instance
type Bot struct {
	session     *discordgo.Session
	commands    map[string]CommandHandler
	commandIDs  map[string]string // Maps command name to command ID
	gameService game.Service
	config      *Config
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
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.Token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	if cfg.GameService == nil {
		return nil, fmt.Errorf("game service cannot be nil")
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
	ButtonJoinGame     = "join_game"
	ButtonBeginGame    = "begin_game"
	ButtonRollDice     = "roll_dice"
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

	// Send a direct message to each participant with a roll button
	// We can't send multiple interaction responses, so we'll use direct messages instead
	for _, participant := range existingGame.Game.Participants {
		// Skip the user who started the game, they'll get the interaction response
		if participant.PlayerID == userID {
			continue
		}

		// Create a DM channel with the participant
		dmChannel, err := s.UserChannelCreate(participant.PlayerID)
		if err != nil {
			log.Printf("Error creating DM channel with %s: %v", participant.PlayerName, err)
			continue
		}

		// Send a message with the roll button
		_, err = s.ChannelMessageSendComplex(dmChannel.ID, &discordgo.MessageSend{
			Content: "A game has started! Click the button below to roll your dice.",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{rollButton},
				},
			},
		})
		if err != nil {
			log.Printf("Error sending DM to %s: %v", participant.PlayerName, err)
		}
	}

	// Send an ephemeral message to the user who started the game
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Game Started! Click the button below to roll your dice.",
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

	// Get the game in this channel
	existingGame, err := b.gameService.GetGameByChannel(ctx, &game.GetGameByChannelInput{
		ChannelID: channelID,
	})

	// Handle errors or missing game
	if err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			return RespondWithEphemeralMessage(s, i, "No active game found in this channel. Use `/ronnied start` to create a new game.")
		}
		log.Printf("Error getting game: %v", err)
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Error getting game: %v", err))
	}

	// Check if the game is in a state where players can roll
	if existingGame.Game.Status != models.GameStatusActive && existingGame.Game.Status != models.GameStatusRollOff {
		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Wait for %s to start the game.", existingGame.Game.GetCreatorName()))
	}

	// For roll-offs, check if this player is eligible to roll
	if existingGame.Game.Status == models.GameStatusRollOff {
		// Check if this player is part of the roll-off
		isInRollOff := false
		for _, participant := range existingGame.Game.Participants {
			if participant.PlayerID == userID {
				isInRollOff = true
				break
			}
		}

		if !isInRollOff {
			return RespondWithEphemeralMessage(s, i, "You are not part of the current roll-off.")
		}
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
					return RespondWithEphemeralMessage(s, i, fmt.Sprintf("You need to roll in the roll-off game. Check the game message for details."))
				}
			}
		}

		return RespondWithEphemeralMessage(s, i, fmt.Sprintf("Failed to roll dice: %v", err))
	}

	// Build the response message based on the roll result
	var title string
	var description string
	var components []discordgo.MessageComponent

	// Handle critical hit (assign a drink)
	if rollOutput.IsCriticalHit {
		title = "Assign a Drink"
		description = "Select a player to assign a drink:"

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
	} else {
		// For all other rolls, just show a simple message with a roll button
		title = "Roll Recorded"
		description = "Your roll has been recorded."
		
		// Add roll dice button for next roll
		rollButton := discordgo.Button{
			Label:    "Roll Again",
			Style:    discordgo.PrimaryButton,
			CustomID: ButtonRollDice,
			Emoji: &discordgo.ComponentEmoji{
				Name: "🎲",
			},
		}
		
		components = append(components, rollButton)
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

	// Check if this is a component interaction (button click)
	if i.Type == discordgo.InteractionMessageComponent {
		// Update the existing message instead of sending a new one
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: title,
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       title,
						Description: description,
						Color:       0x00ff00, // Green color
					},
				},
				Components: messageComponents,
			},
		})
	} else {
		// For the initial interaction, create a new ephemeral message
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: title,
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       title,
						Description: description,
						Color:       0x00ff00, // Green color
					},
				},
				Components: messageComponents,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}
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

	// Update the current message with a confirmation and a roll button
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("You assigned a drink to %s! 🍻", targetPlayerName),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{rollButton},
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

	// Acknowledge the interaction without sending a message
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
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

	// Check if this is a roll-off game
	isRollOff := gameOutput.Game.Status == models.GameStatusRollOff

	// Check if this game has a roll-off in progress
	hasRollOffInProgress := false
	if gameOutput.Game.RollOffGameID != "" {
		// Get the roll-off game to check its status
		rollOffGameOutput, err := b.gameService.GetGame(ctx, &game.GetGameInput{
			GameID: gameOutput.Game.RollOffGameID,
		})
		if err == nil && rollOffGameOutput.Game.Status == models.GameStatusRollOff {
			hasRollOffInProgress = true
		}
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

		// If this is a roll-off game, add information about the roll-off
		if isRollOff {
			// Get the parent game to show what this roll-off is for
			if gameOutput.Game.ParentGameID != "" {
				parentGameOutput, err := b.gameService.GetGame(ctx, &game.GetGameInput{
					GameID: gameOutput.Game.ParentGameID,
				})
				if err == nil {
					// Determine roll-off type based on participants
					rollOffType := "Lowest Roll"
					for _, participant := range parentGameOutput.Game.Participants {
						// Check if any participants in the roll-off had a critical fail in the parent game
						if participant.RollValue == 1 {
							for _, rollOffParticipant := range gameOutput.Game.Participants {
								if rollOffParticipant.PlayerID == participant.PlayerID {
									rollOffType = "Critical Fail"
									break
								}
							}
							if rollOffType == "Critical Fail" {
								break
							}
						}
					}

					// Add roll-off information field
					rollOffInfo := fmt.Sprintf("This is a roll-off for **%s**\n", rollOffType)
					rollOffInfo += "Players in the roll-off:\n"

					// Add player mentions for those who need to roll
					for _, participant := range gameOutput.Game.Participants {
						// Check if the participant hasn't rolled yet
						if participant.RollTime == nil {
							rollOffInfo += fmt.Sprintf("- <@%s> **%s** (needs to roll)\n", participant.PlayerID, participant.PlayerName)
						} else {
							rollOffInfo += fmt.Sprintf("- **%s** (rolled a %d)\n", participant.PlayerName, participant.RollValue)
						}
					}

					fields = append(fields, &discordgo.MessageEmbedField{
						Name:   "Roll-Off Information",
						Value:  rollOffInfo,
						Inline: false,
					})
				}
			}
		} else if hasRollOffInProgress {
			// If this game has a roll-off in progress, show that information
			rollOffGameOutput, err := b.gameService.GetGame(ctx, &game.GetGameInput{
				GameID: gameOutput.Game.RollOffGameID,
			})
			if err == nil {
				// Create a list of players in the roll-off
				var rollOffPlayerNames []string
				var playersNeedingToRoll []string
				
				for _, participant := range rollOffGameOutput.Game.Participants {
					if participant.RollTime == nil {
						playersNeedingToRoll = append(playersNeedingToRoll, fmt.Sprintf("<@%s>", participant.PlayerID))
						rollOffPlayerNames = append(rollOffPlayerNames, fmt.Sprintf("**%s** (needs to roll)", participant.PlayerName))
					} else {
						rollOffPlayerNames = append(rollOffPlayerNames, fmt.Sprintf("**%s** (rolled a %d)", participant.PlayerName, participant.RollValue))
					}
				}

				// Add roll-off information field
				rollOffInfo := "A roll-off is in progress with the following players:\n"
				for _, name := range rollOffPlayerNames {
					rollOffInfo += fmt.Sprintf("- %s\n", name)
				}
				
				// Add a call to action if players still need to roll
				if len(playersNeedingToRoll) > 0 {
					rollOffInfo += "\n**Waiting for rolls from:** " + strings.Join(playersNeedingToRoll, ", ") + "\n"
					rollOffInfo += "Please click the 'Roll for Tie-Breaker' button to roll."
				}

				fields = append(fields, &discordgo.MessageEmbedField{
					Name:   "Roll-Off In Progress",
					Value:  rollOffInfo,
					Inline: false,
				})
			}
		}

		// Get the drink records for the game to show detailed assignments
		drinkRecords, err := b.gameService.GetDrinkRecords(ctx, &game.GetDrinkRecordsInput{
			GameID: gameID,
		})

		if err == nil && len(drinkRecords.Records) > 0 {
			// Create a field for detailed drink assignments
			drinkAssignments := ""

			// Create a map to track player names
			playerNames := make(map[string]string)
			for _, participant := range gameOutput.Game.Participants {
				playerNames[participant.PlayerID] = participant.PlayerName
			}

			// Group drink records by reason
			criticalHitAssignments := ""
			criticalFailAssignments := ""
			lowestRollAssignments := ""

			for _, record := range drinkRecords.Records {
				fromName := playerNames[record.FromPlayerID]
				toName := playerNames[record.ToPlayerID]

				switch record.Reason {
				case models.DrinkReasonCriticalHit:
					criticalHitAssignments += fmt.Sprintf("**%s** passed the drink to **%s** 🎯\n", fromName, toName)
				case models.DrinkReasonCriticalFail:
					criticalFailAssignments += fmt.Sprintf("**%s** rolled a 1, that's a drink 🍻\n", toName)
				case models.DrinkReasonLowestRoll:
					lowestRollAssignments += fmt.Sprintf("**%s** had the lowest roll 🍻\n", toName)
				}
			}

			// Combine all assignments
			if criticalHitAssignments != "" {
				drinkAssignments += "**Critical Hit Assignments:**\n" + criticalHitAssignments + "\n"
			}

			if criticalFailAssignments != "" {
				drinkAssignments += "**Critical Fails:**\n" + criticalFailAssignments + "\n"
			}

			if lowestRollAssignments != "" {
				drinkAssignments += "**Lowest Rolls:**\n" + lowestRollAssignments + "\n"
			}

			if drinkAssignments != "" {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:   "Drink Assignments",
					Value:  drinkAssignments,
					Inline: false,
				})
			}
		}

		// Only show the leaderboard for completed games with no roll-offs in progress
		if gameOutput.Game.Status == models.GameStatusCompleted && !hasRollOffInProgress {
			// Get the leaderboard to show total drinks per player
			leaderboardOutput, err := b.gameService.GetLeaderboard(ctx, &game.GetLeaderboardInput{
				GameID: gameID,
			})

			if err == nil && len(leaderboardOutput.Entries) > 0 {
				// Create a field for drink totals
				drinkTotals := ""
				for _, entry := range leaderboardOutput.Entries {
					if entry.DrinkCount > 0 {
						drinkTotals += fmt.Sprintf("**%s**: %d drink(s)\n", entry.PlayerName, entry.DrinkCount)
					}
				}

				if drinkTotals != "" {
					fields = append(fields, &discordgo.MessageEmbedField{
						Name:   "Final Drink Tally",
						Value:  drinkTotals,
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

	// Get the game status description
	description := getGameStatusDescription(gameOutput.Game.Status)
	
	// Add additional information for active games
	if gameOutput.Game.Status == models.GameStatusActive || gameOutput.Game.Status == models.GameStatusRollOff {
		description += "\n\n**Players:** Check your DMs for a roll button message."
	}

	// Create the embed
	var components []discordgo.MessageComponent

	if gameOutput.Game.Status == models.GameStatusWaiting {
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
				Name: "▶️",
			},
		}

		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				joinButton,
				beginButton,
			},
		})
	} else if gameOutput.Game.Status == models.GameStatusActive || gameOutput.Game.Status == models.GameStatusRollOff {
		// No buttons for active or roll-off games
	} else if gameOutput.Game.Status == models.GameStatusCompleted {
		// Add new game button
		newGameButton := discordgo.Button{
			Label:    "Start New Game",
			Style:    discordgo.SuccessButton,
			CustomID: ButtonStartNewGame,
			Emoji: &discordgo.ComponentEmoji{
				Name: "🎮",
			},
		}

		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				newGameButton,
			},
		})
	}

	// Edit the message
	_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: gameOutput.Game.ChannelID,
		ID:      gameOutput.Game.MessageID,
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       getGameStatusTitle(gameOutput.Game.Status),
				Description: description,
				Color:       0x00ff00, // Green color
				Fields:      fields,
			},
		},
		Components: &components,
	})

	if err != nil {
		log.Printf("Error updating game message: %v", err)
	}
}

func getGameStatusTitle(status models.GameStatus) string {
	switch status {
	case models.GameStatusWaiting:
		return "Game Waiting"
	case models.GameStatusActive:
		return "Game Active"
	case models.GameStatusRollOff:
		return "Roll-Off"
	case models.GameStatusCompleted:
		return "Game Completed"
	default:
		return "Unknown Status"
	}
}

func getGameStatusDescription(status models.GameStatus) string {
	switch status {
	case models.GameStatusWaiting:
		return "Players are waiting to join the game."
	case models.GameStatusActive:
		return "The game is active. Players can roll their dice."
	case models.GameStatusRollOff:
		return "A roll-off is in progress."
	case models.GameStatusCompleted:
		return "The game is completed."
	default:
		return "Unknown status"
	}
}
