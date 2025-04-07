package discord

import (
	"fmt"
	"strings"

	"github.com/KirkDiggler/ronnied/internal/models"
	"github.com/KirkDiggler/ronnied/internal/services/game"
	"github.com/bwmarrin/discordgo"
)

// renderRollDiceResponse renders the response for a roll dice action
func renderRollDiceResponse(s *discordgo.Session, i *discordgo.InteractionCreate, output *game.RollDiceOutput) error {
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

	// Create embeds
	embeds := []*discordgo.MessageEmbed{
		{
			Title:       output.Result,
			Description: output.Details,
			Color:       0x00ff00, // Green color
		},
	}

	// Check if this is a component interaction (button click)
	if i.Type == discordgo.InteractionMessageComponent {
		// Update the existing message instead of sending a new one
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    output.Result,
				Embeds:     embeds,
				Components: messageComponents,
			},
		})
	} else {
		// For the initial interaction, create a new ephemeral message
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    output.Result,
				Embeds:     embeds,
				Components: messageComponents,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

// renderGameMessage renders the game message based on the current game state
func renderGameMessage(game *models.Game, drinkRecords []*models.DrinkLedger, leaderboardEntries []game.LeaderboardEntry, rollOffGame *models.Game, parentGame *models.Game) (*discordgo.MessageEdit, error) {
	// Create fields for the message
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Status",
			Value:  string(game.Status),
			Inline: true,
		},
		{
			Name:   "Players",
			Value:  fmt.Sprintf("%d", len(game.Participants)),
			Inline: true,
		},
	}

	// Check if this is a roll-off game
	isRollOff := game.Status == models.GameStatusRollOff

	// Check if this game has a roll-off in progress
	hasRollOffInProgress := rollOffGame != nil && rollOffGame.Status == models.GameStatusRollOff

	// Add player names and their rolls if the game is active or completed
	if game.Status.IsActive() || game.Status.IsCompleted() {

		// Create a field for player rolls
		playerRolls := ""
		for _, participant := range game.Participants {
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

			// Add roll-off information field
			rollOffInfo := fmt.Sprintf("This is a roll-off for **%s**\n", rollOffType)
			rollOffInfo += "Players in the roll-off:\n"

			// Add player mentions for those who need to roll
			for _, participant := range game.Participants {
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
		} else if hasRollOffInProgress {
			// If this game has a roll-off in progress, show that information
			// Create a list of players in the roll-off
			var rollOffPlayerNames []string
			var playersNeedingToRoll []string

			for _, participant := range rollOffGame.Participants {
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
					criticalHitAssignments += fmt.Sprintf("**%s** passed the drink to **%s** 🎯\n", fromName, toName)
				case models.DrinkReasonCriticalFail:
					criticalFailAssignments += fmt.Sprintf("**%s** rolled a 1, that's a drink 🍻\n", toName)
				case models.DrinkReasonLowestRoll:
					lowestRollAssignments += fmt.Sprintf("**%s** had the lowest roll 🍻\n", toName)
				}
			}

			// Combine all assignments
			if criticalHitAssignments != "" {
				drinkAssignments += "**Crits!:**\n" + criticalHitAssignments + "\n"
			}

			if criticalFailAssignments != "" {
				drinkAssignments += "**Fails!:**\n" + criticalFailAssignments + "\n"
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
		if game.Status.IsCompleted() && !hasRollOffInProgress && len(leaderboardEntries) > 0 {
			// Create a field for drink totals
			drinkTotals := ""
			for _, entry := range leaderboardEntries {
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
	} else {
		// Just show player names for waiting games
		playerNames := ""
		for _, participant := range game.Participants {
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
	description := game.Status.Description()

	// Add additional information for active games
	if game.Status.IsActive() {
		description += "\n\n**Players:** Check your DMs for a roll button message."
	}

	// Create the embed
	var components []discordgo.MessageComponent

	if game.Status.IsWaiting() {
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
	} else if game.Status.IsActive() || game.Status.IsRollOff() {
		// Explicitly set components to an empty slice for active or roll-off games
		// This ensures any previous components are removed
		components = []discordgo.MessageComponent{}
	} else if game.Status.IsCompleted() {
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

	// Create the message edit
	messageEdit := &discordgo.MessageEdit{
		Channel: game.ChannelID,
		ID:      game.MessageID,
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       game.Status.DisplayTitle(),
				Description: description,
				Color:       0x00ff00, // Green color
				Fields:      fields,
			},
		},
	}
	
	// Only set Components if we have any
	if len(components) > 0 {
		messageEdit.Components = &components
	} else {
		// Explicitly set to nil to remove any existing components
		var emptyComponents []discordgo.MessageComponent
		messageEdit.Components = &emptyComponents
	}

	return messageEdit, nil
}
