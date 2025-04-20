package discord

import (
	"context"
	"fmt"
	"log"
	"sort"

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

			components = append(components, discordgo.SelectMenu(playerSelect))
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
	// Determine color based on roll result
	var embedColor int
	if output.IsCriticalHit {
		embedColor = 0x2ecc71 // Green for critical hits
	} else if output.RollValue == 1 {
		embedColor = 0xe74c3c // Red for critical fails
	} else {
		embedColor = 0x3498db // Blue for normal rolls
	}

	ctx := context.Background()
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

	// Get a dynamic roll result message from the messaging service
	// Determine color based on roll result
	var embedColor int
	if output.IsCriticalHit {
		embedColor = 0x2ecc71 // Green for critical hits
	} else if output.RollValue == 1 {
		embedColor = 0xe74c3c // Red for critical fails
	} else {
		embedColor = 0x3498db // Blue for normal rolls
	}

	ctx := context.Background()
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

// renderGameMessage renders the game message with the current state
func renderGameMessage(s *discordgo.Session, game *models.Game, leaderboard *game.GetSessionLeaderboardOutput) error {
	var embeds []*discordgo.MessageEmbed
	var components []discordgo.MessageComponent

	// Create the base embed
	embed := &discordgo.MessageEmbed{
		Title: "Ronnied Drinking Game",
		Color: 0x3498db, // Blue color
	}

	// Add fields based on game status
	switch game.Status {
	case models.GameStatusWaiting:
		embed.Description = "Waiting for players to join..."
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  "Waiting",
				Inline: true,
			},
			{
				Name:   "Players",
				Value:  fmt.Sprintf("%d", len(game.Participants)),
				Inline: true,
			},
		}

		// Add join and begin buttons
		joinButton := discordgo.Button{
			Label:    "Join Game",
			Style:    discordgo.SuccessButton,
			CustomID: ButtonJoinGame,
			Emoji: discordgo.ComponentEmoji{
				Name: "🎲",
			},
		}

		beginButton := discordgo.Button{
			Label:    "Begin Game",
			Style:    discordgo.PrimaryButton,
			CustomID: ButtonBeginGame,
			Emoji: discordgo.ComponentEmoji{
				Name: "▶️",
			},
		}

		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				joinButton,
				beginButton,
			},
		})

	case models.GameStatusActive:
		embed.Description = "Game in progress! Each player should roll their dice."
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  "Active",
				Inline: true,
			},
			{
				Name:   "Players",
				Value:  fmt.Sprintf("%d", len(game.Participants)),
				Inline: true,
			},
		}

	case models.GameStatusRollOff:
		embed.Description = "🔄 **ROLL-OFF IN PROGRESS!** Players in the roll-off need to roll again to break the tie."
		embed.Color = 0xff9900 // Orange color for roll-offs to make them stand out

		// Add fields for roll-off status
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  "⚔️ Roll-Off",
				Inline: true,
			},
			{
				Name:   "Players",
				Value:  fmt.Sprintf("%d", len(game.Participants)),
				Inline: true,
			},
		}

		// If this is a roll-off game, add info about the parent game
		if game.ParentGameID != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Roll-Off Type",
				Value: "This is a tie-breaker roll-off",
			})
		}

		// Add a special field highlighting who needs to roll
		var pendingRollers string
		for _, p := range game.Participants {
			if p.RollTime == nil {
				pendingRollers += fmt.Sprintf("• **%s** - NEEDS TO ROLL! 🎲\n", p.PlayerName)
			} else {
				pendingRollers += fmt.Sprintf("• %s - Already rolled ✅\n", p.PlayerName)
			}
		}

		if pendingRollers != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Roll-Off Participants",
				Value: pendingRollers,
			})
		}

		rollButton := discordgo.Button{
			Label:    "Roll Dice",
			Style:    discordgo.DangerButton, // Red to make it stand out
			CustomID: ButtonRollDice,
			Emoji: discordgo.ComponentEmoji{
				Name: "🎲",
			},
		}
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				rollButton,
			},
		})

	case models.GameStatusCompleted:
		embed.Description = "Game completed! Here are the final results."
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  "Completed",
				Inline: true,
			},
			{
				Name:   "Players",
				Value:  fmt.Sprintf("%d", len(game.Participants)),
				Inline: true,
			},
		}

		// Add start new game button
		startNewGameButton := discordgo.Button{
			Label:    "Start New Game",
			Style:    discordgo.SuccessButton,
			CustomID: ButtonStartNewGame,
			Emoji: discordgo.ComponentEmoji{
				Name: "🎮",
			},
		}

		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				startNewGameButton,
			},
		})
	}

	// Add participant list
	var participantList string
	for _, p := range game.Participants {
		var rollInfo string
		if p.RollValue > 0 {
			rollInfo = fmt.Sprintf(" (Rolled: %d)", p.RollValue)
		} else {
			rollInfo = " (Not rolled yet)"
		}
		participantList += fmt.Sprintf("• %s%s\n", p.PlayerName, rollInfo)
	}

	if participantList != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Participants",
			Value: participantList,
		})
	}

	// Add drink leaderboard if available
	if leaderboard != nil && len(leaderboard.Entries) > 0 {
		// Sort entries by drink count (descending)
		sort.Slice(leaderboard.Entries, func(i, j int) bool {
			return leaderboard.Entries[i].DrinkCount > leaderboard.Entries[j].DrinkCount
		})

		var leaderboardText string
		var totalDrinks int
		var totalPaid int

		// Create a visual progress bar for the session
		for _, entry := range leaderboard.Entries {
			totalDrinks += entry.DrinkCount
			totalPaid += entry.PaidCount

			// Show drinks owed and paid for each player
			remaining := entry.DrinkCount - entry.PaidCount
			var statusEmoji string

			// Select appropriate emoji based on payment status
			if remaining == 0 && entry.DrinkCount > 0 {
				statusEmoji = "🎉" // Celebration emoji for all paid
			} else if entry.DrinkCount > 0 && float64(entry.PaidCount)/float64(entry.DrinkCount) >= 0.75 {
				statusEmoji = "🔥" // Fire emoji for almost done
			} else if entry.DrinkCount > 0 && float64(entry.PaidCount)/float64(entry.DrinkCount) >= 0.5 {
				statusEmoji = "👍" // Thumbs up for good progress
			} else if entry.DrinkCount > 0 && float64(entry.PaidCount)/float64(entry.DrinkCount) >= 0.25 {
				statusEmoji = "🍺" // Beer emoji for some progress
			} else if entry.DrinkCount > 0 {
				statusEmoji = "💪" // Flexed arm for just starting
			} else {
				statusEmoji = "😇" // Angel for no drinks
			}

			// Format the leaderboard entry
			if entry.DrinkCount > 0 {
				leaderboardText += fmt.Sprintf("• %s: %d owed, %d paid, %d remaining %s\n",
					entry.PlayerName, entry.DrinkCount, entry.PaidCount, remaining, statusEmoji)
			} else {
				leaderboardText += fmt.Sprintf("• %s: No drinks owed %s\n", entry.PlayerName, statusEmoji)
			}
		}

		// Add session progress bar if there are any drinks
		if totalDrinks > 0 {
			sessionProgress := createDrinkProgressBar(totalPaid, totalDrinks)
			leaderboardText += fmt.Sprintf("\n**Session Progress**: %s", sessionProgress)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Drink Leaderboard",
			Value: leaderboardText,
		})
	}

	embeds = append(embeds, embed)

	// Edit the message
	messageEdit := &discordgo.MessageEdit{
		Channel: game.ChannelID,
		ID:      game.MessageID,
		Embeds:  embeds,
	}

	// Only set Components if we have any
	if len(components) > 0 {
		log.Printf("Setting %d components for game %s", len(components), game.ID)
		messageEdit.Components = components
	} else {
		log.Printf("No components to set for message edit for game %s (status: %s)", game.ID, game.Status)
		// Explicitly set to nil to remove any existing components
		var emptyComponents []discordgo.MessageComponent
		messageEdit.Components = emptyComponents
		log.Printf("Set empty components array for game %s to clear buttons", game.ID)
	}

	_, err := s.ChannelMessageEditComplex(messageEdit)
	return err
}

// renderGameMessage renders the game message based on the current game state
func (b *Bot) renderGameMessage(game *models.Game, drinkRecords []*models.DrinkLedger, leaderboardEntries []game.LeaderboardEntry, sessionLeaderboardEntries []game.LeaderboardEntry, rollOffGame *models.Game, parentGame *models.Game) (*discordgo.MessageEdit, error) {
	// Create the embed
	embed := &discordgo.MessageEmbed{
		Title: "Ronnied Drinking Game",
		Color: 0x3498db, // Blue color
	}

	// Add fields based on game status
	switch game.Status {
	case models.GameStatusWaiting:
		embed.Description = "Waiting for players to join..."
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  "Waiting",
				Inline: true,
			},
			{
				Name:   "Players",
				Value:  fmt.Sprintf("%d", len(game.Participants)),
				Inline: true,
			},
		}

	case models.GameStatusActive:
		embed.Description = "Game in progress! Each player should roll their dice."
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  "Active",
				Inline: true,
			},
			{
				Name:   "Players",
				Value:  fmt.Sprintf("%d", len(game.Participants)),
				Inline: true,
			},
		}

	case models.GameStatusRollOff:
		embed.Description = "🔄 **ROLL-OFF IN PROGRESS!** Players in the roll-off need to roll again to break the tie."
		embed.Color = 0xff9900 // Orange color for roll-offs to make them stand out

		// Add fields for roll-off status
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  "⚔️ Roll-Off",
				Inline: true,
			},
			{
				Name:   "Players",
				Value:  fmt.Sprintf("%d", len(game.Participants)),
				Inline: true,
			},
		}

		// If this is a roll-off game, add info about the parent game
		if parentGame != nil {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Roll-Off Type",
				Value: "This is a tie-breaker roll-off",
			})
		}

		// Add a special field highlighting who needs to roll
		var pendingRollers string
		for _, p := range game.Participants {
			if p.RollTime == nil {
				pendingRollers += fmt.Sprintf("• **%s** - NEEDS TO ROLL! 🎲\n", p.PlayerName)
			} else {
				pendingRollers += fmt.Sprintf("• %s - Already rolled ✅\n", p.PlayerName)
			}
		}

		if pendingRollers != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Roll-Off Participants",
				Value: pendingRollers,
			})
		}

	case models.GameStatusCompleted:
		embed.Description = "Game completed! Here are the final results."
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  "Completed",
				Inline: true,
			},
			{
				Name:   "Players",
				Value:  fmt.Sprintf("%d", len(game.Participants)),
				Inline: true,
			},
		}
	}

	// Add participant list
	var participantList string
	for _, p := range game.Participants {
		var rollInfo string
		if p.RollValue > 0 {
			rollInfo = fmt.Sprintf(" (Rolled: %d)", p.RollValue)
		} else {
			rollInfo = " (Not rolled yet)"
		}
		participantList += fmt.Sprintf("• %s%s\n", p.PlayerName, rollInfo)
	}

	if participantList != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Participants",
			Value: participantList,
		})
	}

	// Add drink leaderboard if available
	if len(sessionLeaderboardEntries) > 0 {
		// Sort entries by drink count (descending)
		sort.Slice(sessionLeaderboardEntries, func(i, j int) bool {
			return sessionLeaderboardEntries[i].DrinkCount > sessionLeaderboardEntries[j].DrinkCount
		})

		var leaderboardText string
		var totalDrinks int
		var totalPaid int

		// Create a visual progress bar for the session
		for _, entry := range sessionLeaderboardEntries {
			totalDrinks += entry.DrinkCount
			totalPaid += entry.PaidCount

			// Show drinks owed and paid for each player
			remaining := entry.DrinkCount - entry.PaidCount
			var statusEmoji string

			// Select appropriate emoji based on payment status
			if remaining == 0 && entry.DrinkCount > 0 {
				statusEmoji = "🎉" // Celebration emoji for all paid
			} else if entry.DrinkCount > 0 && float64(entry.PaidCount)/float64(entry.DrinkCount) >= 0.75 {
				statusEmoji = "🔥" // Fire emoji for almost done
			} else if entry.DrinkCount > 0 && float64(entry.PaidCount)/float64(entry.DrinkCount) >= 0.5 {
				statusEmoji = "👍" // Thumbs up for good progress
			} else if entry.DrinkCount > 0 && float64(entry.PaidCount)/float64(entry.DrinkCount) >= 0.25 {
				statusEmoji = "🍺" // Beer emoji for some progress
			} else if entry.DrinkCount > 0 {
				statusEmoji = "💪" // Flexed arm for just starting
			} else {
				statusEmoji = "😇" // Angel for no drinks
			}

			// Format the leaderboard entry
			if entry.DrinkCount > 0 {
				leaderboardText += fmt.Sprintf("• %s: %d owed, %d paid, %d remaining %s\n",
					entry.PlayerName, entry.DrinkCount, entry.PaidCount, remaining, statusEmoji)
			} else {
				leaderboardText += fmt.Sprintf("• %s: No drinks owed %s\n", entry.PlayerName, statusEmoji)
			}
		}

		// Add session progress bar if there are any drinks
		if totalDrinks > 0 {
			sessionProgress := createProgressBar(totalPaid, totalDrinks)
			leaderboardText += fmt.Sprintf("\n**Session Progress**: %s", sessionProgress)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Drink Leaderboard",
			Value: leaderboardText,
		})
	} else if len(leaderboardEntries) > 0 {
		// If no session leaderboard, fall back to game leaderboard
		var leaderboardText string
		for _, entry := range leaderboardEntries {
			leaderboardText += fmt.Sprintf("• %s: %d drinks\n", entry.PlayerName, entry.DrinkCount)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Drink Leaderboard",
			Value: leaderboardText,
		})
	}

	// Create embeds array
	embeds := []*discordgo.MessageEmbed{embed}

	// Create components based on game status
	var components []discordgo.MessageComponent

	switch game.Status {
	case models.GameStatusWaiting:
		// Add join and begin buttons
		joinButton := discordgo.Button{
			Label:    "Join Game",
			Style:    discordgo.SuccessButton,
			CustomID: ButtonJoinGame,
			Emoji: discordgo.ComponentEmoji{
				Name: "🎲",
			},
		}

		beginButton := discordgo.Button{
			Label:    "Begin Game",
			Style:    discordgo.PrimaryButton,
			CustomID: ButtonBeginGame,
			Emoji: discordgo.ComponentEmoji{
				Name: "▶️",
			},
		}

		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				joinButton,
				beginButton,
			},
		})

	case models.GameStatusRollOff:
		rollButton := discordgo.Button{
			Label:    "Roll Dice",
			Style:    discordgo.DangerButton, // Red to make it stand out
			CustomID: ButtonRollDice,
			Emoji: discordgo.ComponentEmoji{
				Name: "🎲",
			},
		}
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				rollButton,
			},
		})

	case models.GameStatusCompleted:
		// Add start new game button
		startNewGameButton := discordgo.Button{
			Label:    "Start New Game",
			Style:    discordgo.SuccessButton,
			CustomID: ButtonStartNewGame,
			Emoji: discordgo.ComponentEmoji{
				Name: "🎮",
			},
		}

		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				startNewGameButton,
			},
		})
	}

	// Create the message edit
	messageEdit := &discordgo.MessageEdit{
		Channel: game.ChannelID,
		ID:      game.MessageID,
		Embeds:  embeds,
	}

	// Only set Components if we have any
	if len(components) > 0 {
		log.Printf("Setting %d components for game %s", len(components), game.ID)
		messageEdit.Components = components
	} else {
		log.Printf("No components to set for message edit for game %s (status: %s)", game.ID, game.Status)
		// Explicitly set to nil to remove any existing components
		var emptyComponents []discordgo.MessageComponent
		messageEdit.Components = emptyComponents
		log.Printf("Set empty components array for game %s to clear buttons", game.ID)
	}

	return messageEdit, nil
}

// createDrinkProgressBar creates a visual progress bar for drink payments
func createDrinkProgressBar(paidCount int, totalDrinks int) string {
	// Handle edge cases
	if totalDrinks == 0 {
		return "No drinks to pay"
	}

	// Calculate progress
	progress := float64(paidCount) / float64(totalDrinks)

	// Select appropriate bar characters based on Discord's rendering
	filledChar := "🟩" // Green square for paid drinks
	emptyChar := "⬜"  // White square for unpaid drinks

	// For small numbers of drinks (≤ 10), show one character per drink
	if totalDrinks <= 10 {
		var progressBar string
		for i := 0; i < totalDrinks; i++ {
			if i < paidCount {
				progressBar += filledChar
			} else {
				progressBar += emptyChar
			}
		}
		return progressBar + fmt.Sprintf(" (%d/%d)", paidCount, totalDrinks)
	}

	// For larger numbers, create a 10-segment bar
	const segments = 10
	filledSegments := int(progress * segments)

	var progressBar string
	for i := 0; i < segments; i++ {
		if i < filledSegments {
			progressBar += filledChar
		} else {
			progressBar += emptyChar
		}
	}

	// Add percentage to the progress bar
	progressBar += fmt.Sprintf(" (%.0f%%)", progress*100)

	return progressBar
}

// createProgressBar creates a visual progress bar for drink payments
func createProgressBar(paidCount int, totalDrinks int) string {
	// Handle edge cases
	if totalDrinks == 0 {
		return "No drinks to pay"
	}

	// Calculate progress
	progress := float64(paidCount) / float64(totalDrinks)

	// Select appropriate bar characters based on Discord's rendering
	filledChar := "🟩" // Green square for paid drinks
	emptyChar := "⬜"  // White square for unpaid drinks

	// For small numbers of drinks (≤ 10), show one character per drink
	if totalDrinks <= 10 {
		var progressBar string
		for i := 0; i < totalDrinks; i++ {
			if i < paidCount {
				progressBar += filledChar
			} else {
				progressBar += emptyChar
			}
		}
		return progressBar + fmt.Sprintf(" (%d/%d)", paidCount, totalDrinks)
	}

	// For larger numbers, create a 10-segment bar
	const segments = 10
	filledSegments := int(progress * segments)

	var progressBar string
	for i := 0; i < segments; i++ {
		if i < filledSegments {
			progressBar += filledChar
		} else {
			progressBar += emptyChar
		}
	}

	// Add percentage to the progress bar
	progressBar += fmt.Sprintf(" (%.0f%%)", progress*100)

	return progressBar
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
