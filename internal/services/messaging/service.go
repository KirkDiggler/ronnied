package messaging

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
	
	"github.com/KirkDiggler/ronnied/internal/models"
)

// MessageTones
const (
	MessageToneNeutral     MessageTone = "neutral"
	MessageToneFunny       MessageTone = "funny"
	MessageToneSarcastic   MessageTone = "sarcastic"
	MessageToneEncouraging MessageTone = "encouraging"
)

// service implements the Service interface
type service struct {
	// We'll add a repository here later when we implement message storage
	// repository Repository
	
	// Random number generator for selecting random messages
	rand *rand.Rand
}

// NewService creates a new messaging service
func NewService(config *ServiceConfig) (Service, error) {
	// Create a new random source with the current time as seed
	source := rand.NewSource(time.Now().UnixNano())
	
	return &service{
		// repository: config.Repository,
		rand: rand.New(source),
	}, nil
}

// GetJoinGameMessage returns a message for when a player joins a game
func (s *service) GetJoinGameMessage(ctx context.Context, input *GetJoinGameMessageInput) (*GetJoinGameMessageOutput, error) {
	// In the future, we could fetch these from a repository
	var messages []string
	var tone MessageTone
	
	// Set default tone if not specified
	if input.PreferredTone == "" {
		tone = ToneFunny
	} else {
		tone = input.PreferredTone
	}
	
	// Select messages based on game status and whether player already joined
	if input.AlreadyJoined {
		if input.GameStatus.IsWaiting() {
			messages = []string{
				"You're already in the game, eager beaver! Just hang tight while we wait for everyone.",
				"Patience, grasshopper! You're already in this game.",
				"Double-dipping, are we? You're already in this game!",
				"You're already on the roster! Take a seat and grab a drink.",
			}
		} else if input.GameStatus.IsActive() {
			messages = []string{
				"Ready to roll? Here's your dice button again. Don't lose it this time! 😉",
				"Found your dice button again! Try not to drop it this time.",
				"Look who came back for their dice! Roll away, my friend.",
				"The dice await your command... again. Let's hope they're luckier this time!",
			}
		} else if input.GameStatus.IsRollOff() {
			messages = []string{
				"Patience, young padawan! The roll-off is in progress. Your turn will come.",
				"Hold your horses! There's a roll-off happening.",
				"Roll-off in progress! This is where legends (and hangovers) are made.",
				"The tension builds during this roll-off! Stay tuned for your turn.",
			}
		} else {
			messages = []string{
				"You're already in this game! Did you forget? 🤔",
				"Having memory issues? You're already in this game!",
				"You're part of this game already. Maybe have one less drink?",
				"Already on the team! No need to join twice.",
			}
		}
	} else {
		// New player joining
		messages = []string{
			"Welcome to the party! 🎉 Get ready to roll when the game begins.",
			"Fresh meat! Er, I mean... welcome to the game!",
			"A new challenger appears! Get ready to roll those dice.",
			"Look who decided to join! The dice gods await your tribute.",
		}
	}
	
	// Select a random message
	selectedMessage := messages[s.rand.Intn(len(messages))]
	
	return &GetJoinGameMessageOutput{
		Message: selectedMessage,
		Tone:    tone,
	}, nil
}

// GetJoinGameErrorMessage returns an error message for when a player fails to join a game
func (s *service) GetJoinGameErrorMessage(ctx context.Context, input *GetJoinGameErrorMessageInput) (*GetJoinGameErrorMessageOutput, error) {
	// Set default tone if not specified
	tone := input.Tone
	if tone == "" {
		tone = MessageToneFunny // Default to funny tone
	}
	
	var messages []string
	
	// Select messages based on error type
	switch input.ErrorType {
	case "game_active":
		messages = []string{
			fmt.Sprintf("Sorry, %s! The game is already in progress. Wait for the next round to show off your dice skills.", input.PlayerName),
			fmt.Sprintf("Whoa there, %s! This train has already left the station. Catch the next game!", input.PlayerName),
			fmt.Sprintf("Too late, %s! The dice are already rolling. Next time, be quicker on the draw!", input.PlayerName),
		}
	case "game_completed":
		messages = []string{
			fmt.Sprintf("%s, this game is over! But don't worry, there's always another chance to lose... I mean play!", input.PlayerName),
			fmt.Sprintf("Game's done, %s! You missed all the fun. Start a new one?", input.PlayerName),
			fmt.Sprintf("Sorry %s, you can't join a finished game. That's like trying to board a plane that's already landed!", input.PlayerName),
		}
	case "game_roll_off":
		messages = []string{
			fmt.Sprintf("%s, there's a roll-off in progress! Only the tied players get to participate in this showdown.", input.PlayerName),
			fmt.Sprintf("Hold your horses, %s! This is a special tie-breaker round. Wait for the next full game.", input.PlayerName),
			fmt.Sprintf("Nice try, %s, but roll-offs are invitation-only events. Wait for the next game to start!", input.PlayerName),
		}
	case "already_joined":
		messages = []string{
			fmt.Sprintf("%s, you're already in this game! One player, one set of dice - those are the rules.", input.PlayerName),
			fmt.Sprintf("Easy there, %s! You're already part of this game. No need to join twice!", input.PlayerName),
			fmt.Sprintf("Whoa, %s! You can't join twice. We know you're excited, but save some enthusiasm for the actual game!", input.PlayerName),
		}
	default:
		messages = []string{
			fmt.Sprintf("Sorry %s, you can't join the game right now. Try again later!", input.PlayerName),
			fmt.Sprintf("Hmm, something went wrong, %s. The dice gods are not smiling upon your join attempt.", input.PlayerName),
			fmt.Sprintf("No dice, %s! Something's preventing you from joining. Try again or wait for a new game.", input.PlayerName),
		}
	}
	
	// Select a random message
	selectedMessage := messages[s.rand.Intn(len(messages))]
	
	return &GetJoinGameErrorMessageOutput{
		Title:   "Error Joining Game",
		Message: selectedMessage,
	}, nil
}

// GetGameStatusMessage returns a dynamic message based on the game status
func (s *service) GetGameStatusMessage(ctx context.Context, input *GetGameStatusMessageInput) (*GetGameStatusMessageOutput, error) {
	// Set default tone if not specified
	tone := input.Tone
	if tone == "" {
		tone = MessageToneFunny // Default to funny tone
	}
	
	var messages []string
	
	// Select messages based on game status
	switch input.GameStatus {
	case models.GameStatusWaiting:
		messages = []string{
			"Gather 'round, brave souls! The dice await your courage (and your liver).",
			"A new drinking game is forming. Join now or forever hold your sobriety!",
			"Looking for players who can roll dice better than they can hold their liquor.",
			fmt.Sprintf("We've got %d player(s) so far. The more the merrier (and drunker)!", input.ParticipantCount),
			"Game night is loading... Please wait while we prepare the regrets for tomorrow morning.",
		}
	case models.GameStatusActive:
		messages = []string{
			"The game is afoot! Roll those dice and pray to the drinking gods.",
			"May the odds be ever in your favor (but the drinks against you).",
			"It's rolling time! Remember: a 6 means you're lucky, a 1 means you're thirsty.",
			"Game in progress! Roll well or prepare to drink well.",
			"The dice are hot, and soon your throat will be too! Roll wisely.",
		}
	case models.GameStatusRollOff:
		messages = []string{
			"ROLL-OFF! When regular drinking games aren't intense enough.",
			"It's tie-breaker time! May the luckiest drinker win.",
			"The tension mounts as our tied players face the ultimate test of luck.",
			"Roll-off in progress! This is where legends (and hangovers) are made.",
			"The dice gods demand a sacrifice... of sobriety! Roll to determine who drinks.",
		}
	case models.GameStatusCompleted:
		messages = []string{
			"Game over! Time to pay your liquid debts.",
			"The dice have spoken, and they said 'drink up!'",
			"Another game for the books (and another round for the losers).",
			"Game complete! Remember: it's not about winning, it's about making friends drink.",
			"The final tally is in. Bottoms up to the unlucky ones!",
		}
	default:
		// Fallback message
		return &GetGameStatusMessageOutput{
			Message: "Ronnied drinking game in progress. May the odds be in your favor!",
		}, nil
	}
	
	// Select a random message from the appropriate list
	if len(messages) > 0 {
		randomIndex := rand.Intn(len(messages))
		return &GetGameStatusMessageOutput{
			Message: messages[randomIndex],
		}, nil
	}
	
	// Fallback message if no messages available
	return &GetGameStatusMessageOutput{
		Message: "Ronnied drinking game in progress. May the odds be in your favor!",
	}, nil
}

// GetRollResultMessage returns a dynamic message for a dice roll result
func (s *service) GetRollResultMessage(ctx context.Context, input *GetRollResultMessageInput) (*GetRollResultMessageOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	var title, message string
	isPersonal := input.IsPersonalMessage

	// Generate dynamic messages based on roll value
	switch {
	case input.IsCriticalHit:
		// Critical hit (6)
		if isPersonal {
			titles := []string{
				"CRITICAL HIT!",
				"BOOM! Critical Hit!",
				"Natural 6!",
				"Perfect Roll!",
				"MAXIMUM DAMAGE!",
			}
			
			messages := []string{
				"You rolled a 6! Time to make someone drink!",
				"Incredible! You just rolled a 6 and get to assign a drink!",
				"You're on fire! A perfect 6 means someone's about to get thirsty!",
				"The dice gods favor you today! That's a 6! Choose your victim!",
				"CRITICAL HIT! You rolled a 6 and now have the power to make someone drink!",
			}
			
			title = titles[rand.Intn(len(titles))]
			message = messages[rand.Intn(len(messages))]
		} else {
			titles := []string{
				"CRITICAL HIT!",
				"BOOM! Critical Hit!",
				"Natural 6!",
				"Perfect Roll!",
				"MAXIMUM DAMAGE!",
			}
			
			messages := []string{
				fmt.Sprintf("%s rolled a 6! Time to make someone drink!", input.PlayerName),
				fmt.Sprintf("Incredible! %s just rolled a 6 and gets to assign a drink!", input.PlayerName),
				fmt.Sprintf("%s is on fire! A perfect 6 means someone's about to get thirsty!", input.PlayerName),
				fmt.Sprintf("The dice gods favor %s today! That's a 6! Choose your victim!", input.PlayerName),
				fmt.Sprintf("CRITICAL HIT! %s rolled a 6 and now has the power to make someone drink!", input.PlayerName),
			}
			
			title = titles[rand.Intn(len(titles))]
			message = messages[rand.Intn(len(messages))]
		}
		
	case input.IsCriticalFail:
		// Critical fail (1)
		if isPersonal {
			titles := []string{
				"CRITICAL FAIL!",
				"Ouch! Snake Eyes!",
				"Natural 1!",
				"MINIMUM DAMAGE!",
				"Better luck next time!",
			}
			
			messages := []string{
				"You rolled a 1! Time to drink up!",
				"Oof! You just rolled a 1. Bottoms up!",
				"You angered the dice gods with that 1! Drink up, friend!",
				"The dice have spoken! You rolled a 1 and must take a drink!",
				"CRITICAL FAIL! You rolled a 1 and have to drink!",
			}
			
			title = titles[rand.Intn(len(titles))]
			message = messages[rand.Intn(len(messages))]
		} else {
			titles := []string{
				"CRITICAL FAIL!",
				"Ouch! Snake Eyes!",
				"Natural 1!",
				"MINIMUM DAMAGE!",
				"Better luck next time!",
			}
			
			messages := []string{
				fmt.Sprintf("%s rolled a 1! Time to drink up!", input.PlayerName),
				fmt.Sprintf("Oof! %s just rolled a 1. Bottoms up!", input.PlayerName),
				fmt.Sprintf("%s angered the dice gods with that 1! Drink up, friend!", input.PlayerName),
				fmt.Sprintf("The dice have spoken! %s rolled a 1 and must take a drink!", input.PlayerName),
				fmt.Sprintf("CRITICAL FAIL! %s rolled a 1 and has to drink!", input.PlayerName),
			}
			
			title = titles[rand.Intn(len(titles))]
			message = messages[rand.Intn(len(messages))]
		}
		
	default:
		// Normal roll (2-5)
		if isPersonal {
			titles := []string{
				fmt.Sprintf("You rolled a %d!", input.RollValue),
				fmt.Sprintf("It's a %d!", input.RollValue),
				fmt.Sprintf("%d Points!", input.RollValue),
				fmt.Sprintf("Roll: %d", input.RollValue),
				fmt.Sprintf("The dice shows %d", input.RollValue),
			}
			
			messages := []string{
				fmt.Sprintf("You rolled a %d. Not bad!", input.RollValue),
				fmt.Sprintf("The dice landed on %d.", input.RollValue),
				fmt.Sprintf("Your roll: %d - Keep trying!", input.RollValue),
				fmt.Sprintf("A solid %d!", input.RollValue),
				fmt.Sprintf("You rolled %d. The game continues!", input.RollValue),
			}
			
			title = titles[rand.Intn(len(titles))]
			message = messages[rand.Intn(len(messages))]
		} else {
			titles := []string{
				fmt.Sprintf("%d!", input.RollValue),
				fmt.Sprintf("It's a %d!", input.RollValue),
				fmt.Sprintf("%d Points!", input.RollValue),
				fmt.Sprintf("Roll: %d", input.RollValue),
				fmt.Sprintf("The dice shows %d", input.RollValue),
			}
			
			messages := []string{
				fmt.Sprintf("%s rolled a %d. Not bad!", input.PlayerName, input.RollValue),
				fmt.Sprintf("The dice landed on %d for %s.", input.RollValue, input.PlayerName),
				fmt.Sprintf("%s's roll: %d - Keep trying!", input.PlayerName, input.RollValue),
				fmt.Sprintf("A solid %d from %s!", input.RollValue, input.PlayerName),
				fmt.Sprintf("%s rolled %d. The game continues!", input.PlayerName, input.RollValue),
			}
			
			title = titles[rand.Intn(len(titles))]
			message = messages[rand.Intn(len(messages))]
		}
	}

	return &GetRollResultMessageOutput{
		Title:   title,
		Message: message,
	}, nil
}

// GetGameStartedMessage returns a dynamic message for when a game is started
func (s *service) GetGameStartedMessage(ctx context.Context, input *GetGameStartedMessageInput) (*GetGameStartedMessageOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	// Create a variety of fun messages for when a game is started
	messages := []string{
		"Game Started! Click the button below to roll your dice.",
		fmt.Sprintf("The game is ON! %d players are ready to roll. Your turn now!", input.PlayerCount),
		"Let the dice decide your fate! Roll now!",
		"Time to test your luck! Click to roll the dice!",
		"The game has begun! Roll the dice and see what destiny has in store!",
		"Ready, set, ROLL! Click the button to throw your dice!",
		fmt.Sprintf("Game's on! You and %d other brave souls are about to tempt fate!", input.PlayerCount-1),
		"May the odds be ever in your favor! Roll your dice!",
		"It's dice time! Click to roll and see if luck is on your side today!",
		"Game started! Will you roll a critical hit or a critical fail? Find out now!",
	}

	// Select a random message
	selectedMessage := messages[s.rand.Intn(len(messages))]

	return &GetGameStartedMessageOutput{
		Message: selectedMessage,
	}, nil
}

// GetErrorMessage returns a user-friendly error message
func (s *service) GetErrorMessage(ctx context.Context, input *GetErrorMessageInput) (*GetErrorMessageOutput, error) {
	var messages []string
	var tone MessageTone
	
	// Set default tone if not specified
	if input.PreferredTone == "" {
		tone = ToneFunny
	} else {
		tone = input.PreferredTone
	}
	
	// Select messages based on error type
	switch input.ErrorType {
	case "game_active":
		messages = []string{
			"This game is already rolling! Catch the next one.",
			"Too late, hotshot! The dice are already in motion.",
			"The game has started without you. Next time be quicker!",
			"Sorry, this train has left the station. Wait for the next game.",
		}
	case "game_roll_off":
		messages = []string{
			"There's an epic roll-off happening! Wait for the next round.",
			"Roll-off in progress! No new players allowed in this tense moment.",
			"The fate of drinks is being decided in a roll-off. Join the next game!",
			"Can't join during a roll-off! The tension is too high for newcomers.",
		}
	case "game_completed":
		messages = []string{
			"This game is already over! Check out who's buying drinks.",
			"You missed this one completely. The game is already finished!",
			"Too late! The tab has already been settled for this game.",
			"Game over! But you can start a new one if you're thirsty.",
		}
	case "invalid_game_state":
		messages = []string{
			"The game is in a weird state. Try again later or start a new one.",
			"Something's off with this game. Maybe it's had too many drinks?",
			"Can't join right now. The game is... confused.",
			"This game has gone rogue! Best to start a fresh one.",
		}
	case "game_full":
		messages = []string{
			"This game is packed! Try again when someone leaves.",
			"No room at the inn! This game is full.",
			"Too many players already! Wait for the next game.",
			"This party's at capacity! Try again later.",
		}
	case "already_rolled":
		messages = []string{
			"You've already rolled! Give someone else a turn.",
			"Eager, aren't we? You've already had your turn!",
			"One roll per turn! You'll have to wait.",
			"The dice need a break from you! You've already rolled.",
		}
	case "not_your_turn":
		messages = []string{
			"Patience! It's not your turn yet.",
			"Hold your horses! Someone else is rolling now.",
			"Wait your turn! The dice will come to you soon.",
			"The dice aren't ready for you yet! Wait your turn.",
		}
	default:
		messages = []string{
			"Something went wrong! Try again later.",
			"Oops! The dice got confused. Try again.",
			"Error! The dice gods are displeased. Try again later.",
			"Technical difficulties! The dice are being recalibrated.",
		}
	}
	
	// Select a random message
	selectedMessage := messages[s.rand.Intn(len(messages))]
	
	return &GetErrorMessageOutput{
		Message: selectedMessage,
		Tone:    tone,
	}, nil
}
