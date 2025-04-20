package discord

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/KirkDiggler/ronnied/internal/models"
	"github.com/KirkDiggler/ronnied/internal/services/game"
	"github.com/KirkDiggler/ronnied/internal/services/messaging"
	"github.com/bwmarrin/discordgo"
)

// renderRollDiceResponse renders the response for a roll dice action
func renderRollDiceResponse(s *discordgo.Session, i *discordgo.InteractionCreate, output *game.RollDiceOutput, messagingService messaging.Service) error {
	var components []discordgo.MessageComponent

	// Build components based on the roll result
	if output.IsCriticalHit {
		// Create player selection dropdown for critical hits
		if len(output.EligiblePlayers) > 0 {
			var playerOptions []discordgo.SelectMenuOption

			for _, player := range output.EligiblePlayers {
				playerOptions = append(playerOptions, discordgo.SelectMenuOption{
					Label:       player.PlayerName,
					Value:       player.PlayerID,
					Description: "Assign a drink to this player",
					Emoji: &discordgo.ComponentEmoji{
						Name: "🍺",
					},
				})
			}

			playerSelect := discordgo.SelectMenu{
				CustomID:    SelectAssignDrink,
				Placeholder: "Select a player to drink",
				Options:     playerOptions,
			}

			components = append(components, discordgo.SelectMenu(playerSelect))
		}
	} else {
		// Create roll again button for non-critical hits
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

	// Get a dynamic roll result message from the messaging service
	ctx := context.Background()

	// Determine color based on roll result
	var embedColor int
	if output.IsCriticalHit {
		embedColor = 0x2ecc71 // Green for critical hits
	} else if output.RollValue == 1 {
		embedColor = 0xe74c3c // Red for critical fails
	} else {
		embedColor = 0x3498db // Blue for normal rolls
	}

	rollResultOutput, err := messagingService.GetRollResultMessage(ctx, &messaging.GetRollResultMessageInput{
		PlayerName:        output.PlayerName,
		RollValue:         output.RollValue,
		IsCriticalHit:     output.IsCriticalHit,
		IsCriticalFail:    output.RollValue == 1, // Assuming 1 is critical fail
		IsPersonalMessage: true,                  // This is an ephemeral message to the player
	})

	// Get a supportive whisper message from Ronnie
	rollWhisperOutput, whisperErr := messagingService.GetRollWhisperMessage(ctx, &messaging.GetRollWhisperMessageInput{
		PlayerName:     output.PlayerName,
		RollValue:      output.RollValue,
		IsCriticalHit:  output.IsCriticalHit,
		IsCriticalFail: output.RollValue == 1, // Assuming 1 is critical fail
	})

	// Create embeds - either with messaging service output or fallback to static content
	var embeds []*discordgo.MessageEmbed
	var contentText string

	if err != nil {
		log.Printf("Failed to get roll result message: %v", err)
		// Fallback to static description if messaging service fails
		embeds = []*discordgo.MessageEmbed{
			{
				Title:       output.Result,
				Description: output.Details,
				Color:       embedColor,
			},
		}
		contentText = output.Result
	} else {
		embeds = []*discordgo.MessageEmbed{
			{
				Title:       rollResultOutput.Title,
				Description: rollResultOutput.Message,
				Color:       embedColor,
			},
		}
		contentText = rollResultOutput.Title
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

	// Check if this is a component interaction (button click)
	if i.Type == discordgo.InteractionMessageComponent {
		// Update the existing message instead of sending a new one
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    contentText,
				Embeds:     embeds,
				Components: messageComponents,
			},
		})
	} else {
		// For the initial interaction, create a new ephemeral message
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    contentText,
				Embeds:     embeds,
				Components: messageComponents,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

// renderRollDiceResponseEdit renders the response for a roll dice action by editing the deferred message
func renderRollDiceResponseEdit(s *discordgo.Session, i *discordgo.InteractionCreate, output *game.RollDiceOutput, messagingService messaging.Service) error {
	var components []discordgo.MessageComponent

	// Build components based on the roll result
	if output.IsCriticalHit {
		// Create player selection dropdown for critical hits
		if len(output.EligiblePlayers) > 0 {
			var playerOptions []discordgo.SelectMenuOption

			for _, player := range output.EligiblePlayers {
				playerOptions = append(playerOptions, discordgo.SelectMenuOption{
					Label:       player.PlayerName,
					Value:       player.PlayerID,
					Description: "Assign a drink to this player",
					Emoji: &discordgo.ComponentEmoji{
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
			Emoji: &discordgo.ComponentEmoji{
				Name: "🎲",
			},
		}

		// Add Pay Drink button
		payDrinkButton := discordgo.Button{
			Label:    "Pay Drink",
			Style:    discordgo.SuccessButton,
			CustomID: ButtonPayDrink,
			Emoji: &discordgo.ComponentEmoji{
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

	// Get a dynamic roll result message from the messaging service
	ctx := context.Background()

	// Determine color based on roll result
	var embedColor int
	if output.IsCriticalHit {
		embedColor = 0x2ecc71 // Green for critical hits
	} else if output.RollValue == 1 {
		embedColor = 0xe74c3c // Red for critical fails
	} else {
		embedColor = 0x3498db // Blue for normal rolls
	}

	rollResultOutput, err := messagingService.GetRollResultMessage(ctx, &messaging.GetRollResultMessageInput{
		PlayerName:        output.PlayerName,
		RollValue:         output.RollValue,
		IsCriticalHit:     output.IsCriticalHit,
		IsCriticalFail:    output.RollValue == 1, // Assuming 1 is critical fail
		IsPersonalMessage: true,                  // This is an ephemeral message to the player
	})

	// Get a supportive whisper message from Ronnie
	rollWhisperOutput, whisperErr := messagingService.GetRollWhisperMessage(ctx, &messaging.GetRollWhisperMessageInput{
		PlayerName:     output.PlayerName,
		RollValue:      output.RollValue,
		IsCriticalHit:  output.IsCriticalHit,
		IsCriticalFail: output.RollValue == 1, // Assuming 1 is critical fail
	})

	// Create embeds - either with messaging service output or fallback to static content
	var embeds []*discordgo.MessageEmbed
	var contentText string

	if err != nil {
		log.Printf("Failed to get roll result message: %v", err)
		// Fallback to static description if messaging service fails
		embeds = []*discordgo.MessageEmbed{
			{
				Title:       output.Result,
				Description: output.Details,
				Color:       embedColor,
			},
		}
		contentText = output.Result
	} else {
		embeds = []*discordgo.MessageEmbed{
			{
				Title:       rollResultOutput.Title,
				Description: rollResultOutput.Message,
				Color:       embedColor,
			},
		}
		contentText = rollResultOutput.Title
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

	// Edit the deferred message
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:    &contentText,
		Embeds:     &embeds,
		Components: &messageComponents,
	})
	return err
}

// renderGameMessage renders the game message based on the current game state
func (b *Bot) renderGameMessage(game *models.Game, drinkRecords []*models.DrinkLedger, leaderboardEntries []game.LeaderboardEntry, sessionLeaderboardEntries []game.LeaderboardEntry, rollOffGame *models.Game, parentGame *models.Game) (*discordgo.MessageEdit, error) {
	ctx := context.Background()

	// Create a slice to hold all embeds
	var embeds []*discordgo.MessageEmbed

	// Define color codes for different game states
	var primaryColor int

	switch game.Status {
	case models.GameStatusWaiting:
		primaryColor = 0x3498db // Blue
	case models.GameStatusActive:
		primaryColor = 0x2ecc71 // Green
	case models.GameStatusRollOff:
		primaryColor = 0xe67e22 // Orange
	case models.GameStatusCompleted:
		primaryColor = 0x9b59b6 // Purple
	default:
		primaryColor = 0x95a5a6 // Gray (fallback)
	}

	// Get a dynamic game status message from the messaging service
	statusMsgOutput, err := b.messagingService.GetGameStatusMessage(ctx, &messaging.GetGameStatusMessageInput{
		GameStatus:       game.Status,
		ParticipantCount: len(game.Participants),
	})

	var description string
	if err != nil {
		// Fallback to static description if messaging service fails
		description = game.Status.Description()
	} else {
		description = statusMsgOutput.Message
	}

	// Add additional information for active games
	if game.Status.IsActive() {
		description += "\n\n**Players:** Take your action!"
	}

	// Create the main embed with game info
	mainEmbed := &discordgo.MessageEmbed{
		Title:       getGameTitle(game),
		Description: description,
		Color:       primaryColor,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🎮 Game Info",
				Value:  fmt.Sprintf("**Status:** %s\n**Players:** %d", string(game.Status), len(game.Participants)),
				Inline: true,
			},
			{
				Name:   "🕒 Created",
				Value:  game.CreatedAt.Format("Jan 2, 2006 3:04 PM"),
				Inline: true,
			},
		},
	}

	// Add the main embed
	embeds = append(embeds, mainEmbed)

	// Check if this is a roll-off game
	isRollOff := game.Status == models.GameStatusRollOff

	// Check if this game has a roll-off in progress
	hasRollOffInProgress := rollOffGame != nil && rollOffGame.Status == models.GameStatusRollOff

	// Add player section based on game state
	if game.Status.IsWaiting() {
		// Just show player names for waiting games with a waiting emoji
		if len(game.Participants) > 0 {
			playerNames := ""
			for i, participant := range game.Participants {
				playerNames += fmt.Sprintf("%d. %s\n", i+1, participant.PlayerName)
			}

			// Create a player list embed
			playerEmbed := &discordgo.MessageEmbed{
				Title:       "👥 Participants",
				Description: playerNames,
				Color:       primaryColor,
			}

			embeds = append(embeds, playerEmbed)
		}
	} else if game.Status.IsActive() || game.Status.IsCompleted() {
		// Create a field for player rolls with appropriate emojis
		if len(game.Participants) > 0 {
			playerRolls := ""
			for i, participant := range game.Participants {
				rollInfo := "❓ Not rolled yet"
				if participant.RollValue > 0 {
					rollInfo = fmt.Sprintf("🎲 **%d**", participant.RollValue)

					// Add emoji indicators for critical hits and fails
					if participant.RollValue == 6 { // Assuming 6 is critical hit
						rollInfo += " 🎯 Critical Hit!"

						// Get a funny message for this roll
						rollMsgOutput, err := b.messagingService.GetRollResultMessage(ctx, &messaging.GetRollResultMessageInput{
							PlayerName:        participant.PlayerName,
							RollValue:         participant.RollValue,
							IsCriticalHit:     true,
							IsCriticalFail:    false,
							IsPersonalMessage: false, // This is for the group message
						})

						if err == nil {
							rollInfo += fmt.Sprintf("\n> *%s*", rollMsgOutput.Message)
						}
					} else if participant.RollValue == 1 { // Assuming 1 is critical fail
						rollInfo += " 💀 Critical Fail!"

						// Get a funny message for this roll
						rollMsgOutput, err := b.messagingService.GetRollResultMessage(ctx, &messaging.GetRollResultMessageInput{
							PlayerName:        participant.PlayerName,
							RollValue:         participant.RollValue,
							IsCriticalHit:     false,
							IsCriticalFail:    true,
							IsPersonalMessage: false, // This is for the group message
						})

						if err == nil {
							rollInfo += fmt.Sprintf("\n> *%s*", rollMsgOutput.Message)
						}
					} else {
						// Get a funny message for normal rolls too
						rollMsgOutput, err := b.messagingService.GetRollResultMessage(ctx, &messaging.GetRollResultMessageInput{
							PlayerName:        participant.PlayerName,
							RollValue:         participant.RollValue,
							IsCriticalHit:     false,
							IsCriticalFail:    false,
							IsPersonalMessage: false, // This is for the group message
						})

						if err == nil {
							rollInfo += fmt.Sprintf("\n> *%s*", rollMsgOutput.Message)
						}
					}
				}
				playerRolls += fmt.Sprintf("%d. **%s**: %s\n", i+1, participant.PlayerName, rollInfo)
			}

			// Create a player rolls embed
			rollsEmbed := &discordgo.MessageEmbed{
				Title:       "🎲 Player Rolls",
				Description: playerRolls,
				Color:       primaryColor,
			}

			embeds = append(embeds, rollsEmbed)
		}

		// Handle roll-off information with better formatting
		if isRollOff && parentGame != nil {
			// Determine roll-off type based on participants
			rollOffType := "Lowest Roll"
			for _, participant := range parentGame.Participants {
				// Check if any participants in the roll-off had a critical fail in the parent game
				if participant.RollValue == 1 {
					for _, rollOffParticipant := range game.Participants {
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

			// Add roll-off information field with better formatting
			rollOffInfo := fmt.Sprintf("⚔️ This is a roll-off for **%s**\n\n", rollOffType)
			rollOffInfo += "Players in the roll-off:\n"

			// Add player mentions for those who need to roll
			for i, participant := range game.Participants {
				// Check if the participant hasn't rolled yet
				if participant.RollTime == nil {
					rollOffInfo += fmt.Sprintf("%d. <@%s> **%s** ❓ (needs to roll)\n", i+1, participant.PlayerID, participant.PlayerName)
				} else {
					rollOffInfo += fmt.Sprintf("%d. **%s** 🎲 (rolled a **%d**)\n", i+1, participant.PlayerName, participant.RollValue)
				}
			}

			// Create a roll-off embed
			rollOffEmbed := &discordgo.MessageEmbed{
				Title:       "⚔️ Roll-Off Information",
				Description: rollOffInfo,
				Color:       0xe67e22, // Orange for roll-offs
			}

			embeds = append(embeds, rollOffEmbed)
		} else if hasRollOffInProgress {
			// If this game has a roll-off in progress, show that information with better formatting
			var rollOffPlayerNames []string
			var playersNeedingToRoll []string

			for i, participant := range rollOffGame.Participants {
				if participant.RollTime == nil {
					playersNeedingToRoll = append(playersNeedingToRoll, fmt.Sprintf("<@%s>", participant.PlayerID))
					rollOffPlayerNames = append(rollOffPlayerNames, fmt.Sprintf("%d. **%s** ❓ (needs to roll)", i+1, participant.PlayerName))
				} else {
					rollOffPlayerNames = append(rollOffPlayerNames, fmt.Sprintf("%d. **%s** 🎲 (rolled a **%d**)", i+1, participant.PlayerName, participant.RollValue))
				}
			}

			// Add roll-off information field
			rollOffInfo := "⚔️ A roll-off is in progress with the following players:\n\n"
			for _, name := range rollOffPlayerNames {
				rollOffInfo += name + "\n"
			}

			// Add a call to action if players still need to roll
			if len(playersNeedingToRoll) > 0 {
				rollOffInfo += "\n**Waiting for rolls from:** " + strings.Join(playersNeedingToRoll, ", ") + "\n"
				rollOffInfo += "Please check your DMs for the 'Roll for Tie-Breaker' button."
			}

			// Create a roll-off in progress embed
			rollOffProgressEmbed := &discordgo.MessageEmbed{
				Title:       "⚔️ Roll-Off In Progress",
				Description: rollOffInfo,
				Color:       0xe67e22, // Orange for roll-offs
			}

			embeds = append(embeds, rollOffProgressEmbed)
		}

		// Add drink assignments with better formatting
		if len(drinkRecords) > 0 {
			// Create a field for detailed drink assignments
			drinkAssignments := ""

			// Create a map to track player names
			playerNames := make(map[string]string)
			for _, participant := range game.Participants {
				playerNames[participant.PlayerID] = participant.PlayerName
			}

			// Group drink records by reason
			criticalHitAssignments := ""
			criticalFailAssignments := ""
			lowestRollAssignments := ""

			for _, record := range drinkRecords {
				fromName := playerNames[record.FromPlayerID]
				toName := playerNames[record.ToPlayerID]

				switch record.Reason {
				case models.DrinkReasonCriticalHit:
					criticalHitAssignments += fmt.Sprintf("• **%s** passed the drink to **%s** 🎯\n", fromName, toName)
				case models.DrinkReasonCriticalFail:
					criticalFailAssignments += fmt.Sprintf("• **%s** rolled a 1, that's a drink 💀\n", toName)
				case models.DrinkReasonLowestRoll:
					lowestRollAssignments += fmt.Sprintf("• **%s** had the lowest roll 🍺\n", toName)
				}
			}

			// Combine all assignments with section dividers
			if criticalHitAssignments != "" {
				drinkAssignments += "🎯 **Critical Hits:**\n" + criticalHitAssignments + "\n"
			}

			if criticalFailAssignments != "" {
				drinkAssignments += "💀 **Critical Fails:**\n" + criticalFailAssignments + "\n"
			}

			if lowestRollAssignments != "" {
				drinkAssignments += "🍺 **Lowest Rolls:**\n" + lowestRollAssignments + "\n"
			}

			if drinkAssignments != "" {
				// Create a drink assignments embed
				drinkEmbed := &discordgo.MessageEmbed{
					Title:       "🍻 Drink Assignments",
					Description: drinkAssignments,
					Color:       0xf39c12, // Amber for drinks
				}

				embeds = append(embeds, drinkEmbed)
			}
		}

		// Only show the leaderboard for completed games with no roll-offs in progress
		if game.Status.IsCompleted() && !hasRollOffInProgress && len(leaderboardEntries) > 0 {
			// Create a field for drink totals with better formatting
			drinkTotals := ""

			// Sort leaderboard entries by drink count (descending)
			sort.Slice(leaderboardEntries, func(i, j int) bool {
				return leaderboardEntries[i].DrinkCount > leaderboardEntries[j].DrinkCount
			})

			for i, entry := range leaderboardEntries {
				if entry.DrinkCount > 0 {
					// Add medal emojis for top 3
					var prefix string
					if i == 0 {
						prefix = "🥇"
					} else if i == 1 {
						prefix = "🥈"
					} else if i == 2 {
						prefix = "🥉"
					} else {
						prefix = "•"
					}

					drinkTotals += fmt.Sprintf("%s **%s**: %d drink(s)\n", prefix, entry.PlayerName, entry.DrinkCount)

					// Get a funny message for this leaderboard entry
					leaderboardMsgOutput, err := b.messagingService.GetLeaderboardMessage(ctx, &messaging.GetLeaderboardMessageInput{
						PlayerName:   entry.PlayerName,
						DrinkCount:   entry.DrinkCount,
						Rank:         i,
						TotalPlayers: len(leaderboardEntries),
					})

					if err == nil {
						drinkTotals += fmt.Sprintf("> *%s*\n", leaderboardMsgOutput.Message)
					}
				}
			}

			if drinkTotals != "" {
				// Create a leaderboard embed
				leaderboardEmbed := &discordgo.MessageEmbed{
					Title:       "🏆 Final Drink Tally",
					Description: drinkTotals,
					Color:       0x9b59b6, // Purple for completed games
				}

				embeds = append(embeds, leaderboardEmbed)
			}

			// Add session leaderboard if available
			if len(sessionLeaderboardEntries) > 0 {
				sessionDrinkTotals := ""

				// Sort the session leaderboard entries by drink count (descending)
				sort.Slice(sessionLeaderboardEntries, func(i, j int) bool {
					return sessionLeaderboardEntries[i].DrinkCount > sessionLeaderboardEntries[j].DrinkCount
				})

				for i, entry := range sessionLeaderboardEntries {
					// Add rank emoji
					var prefix string
					switch i {
					case 0:
						prefix = "🥇"
					case 1:
						prefix = "🥈"
					case 2:
						prefix = "🥉"
					default:
						prefix = "🍺"
					}

					// Show both total drinks and paid drinks
					sessionDrinkTotals += fmt.Sprintf("%s **%s**: %d drink(s) owed, %d paid\n",
						prefix, entry.PlayerName, entry.DrinkCount, entry.PaidCount)
				}

				if sessionDrinkTotals != "" {
					// Create a session leaderboard embed
					sessionLeaderboardEmbed := &discordgo.MessageEmbed{
						Title:       "🏆 Session Leaderboard",
						Description: "Drinks across all games in this session:\n\n" + sessionDrinkTotals,
						Color:       0xf1c40f, // Gold for session stats
						Footer: &discordgo.MessageEmbedFooter{
							Text: "Use /ronnied newsession to start a new drinking session",
						},
					}

					embeds = append(embeds, sessionLeaderboardEmbed)
				}
			}
		}
	}

	// Create the message components
	var components []discordgo.MessageComponent

	log.Printf("Rendering game message for game %s with status %s", game.ID, game.Status)

	// Always show the Join button for waiting and active games
	if game.Status.IsWaiting() || game.Status.IsActive() {
		log.Printf("Game %s is waiting or active, adding join button", game.ID)

		// Create the Join button
		joinButton := discordgo.Button{
			Label:    "Join Game",
			Style:    discordgo.SuccessButton,
			CustomID: ButtonJoinGame,
			Emoji: &discordgo.ComponentEmoji{
				Name: "🎲",
			},
		}

		// For waiting games, also add the Begin button
		if game.Status.IsWaiting() {
			log.Printf("Game %s is waiting, also adding begin button", game.ID)

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
		} else {
			// For active games, just add the Join button
			components = append(components, discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					joinButton,
				},
			})
		}
	} else if game.Status.IsCompleted() {
		log.Printf("Game %s is completed, adding new game button", game.ID)
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
	} else {
		log.Printf("Game %s is roll-off, no buttons needed", game.ID)
	}

	// Create the message edit
	messageEdit := &discordgo.MessageEdit{
		Channel: game.ChannelID,
		ID:      game.MessageID,
		Embeds:  &embeds,
	}

	// Only set Components if we have any
	if len(components) > 0 {
		log.Printf("Setting %d components for game %s", len(components), game.ID)
		messageEdit.Components = &components
	} else {
		log.Printf("No components to set for message edit for game %s (status: %s)", game.ID, game.Status)
		// Explicitly set to nil to remove any existing components
		var emptyComponents []discordgo.MessageComponent
		messageEdit.Components = &emptyComponents
		log.Printf("Set empty components array for game %s to clear buttons", game.ID)
	}

	return messageEdit, nil
}

// getGameTitle returns a dynamic title based on game status
func getGameTitle(game *models.Game) string {
	switch game.Status {
	case models.GameStatusWaiting:
		return "🎲 Ronnied Drinking Game - Waiting for Players"
	case models.GameStatusActive:
		return "🎲 Ronnied Drinking Game - Roll the Dice!"
	case models.GameStatusRollOff:
		return "⚔️ Ronnied Drinking Game - Roll-Off in Progress"
	case models.GameStatusCompleted:
		return "🏆 Ronnied Drinking Game - Game Complete"
	default:
		return "🎲 Ronnied Drinking Game"
	}
}
