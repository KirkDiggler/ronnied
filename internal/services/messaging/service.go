package messaging

import (
	"context"
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

// GetRollResultMessage returns a message for a dice roll result
func (s *service) GetRollResultMessage(ctx context.Context, input *GetRollResultMessageInput) (*GetRollResultMessageOutput, error) {
	var messages []string
	var tone MessageTone
	
	// Set default tone if not specified
	if input.PreferredTone == "" {
		if input.IsCriticalHit {
			tone = ToneCelebration
		} else if input.IsCriticalFail {
			tone = ToneSarcastic
		} else {
			tone = ToneNeutral
		}
	} else {
		tone = input.PreferredTone
	}
	
	// Select messages based on roll result
	if input.IsCriticalHit {
		messages = []string{
			"CRITICAL HIT! Time to make someone drink!",
			"A perfect 6! Someone's about to have a bad day!",
			"The dice gods smile upon you with a 6! Choose your victim!",
			"BOOM! A 6! Point your finger at the unlucky soul who drinks!",
		}
	} else if input.IsCriticalFail {
		messages = []string{
			"CRITICAL FAIL! Bottoms up!",
			"A pitiful 1! Drink up, buttercup!",
			"The dice gods laugh at your misfortune! Take a drink!",
			"Ouch, a 1! Hope that drink tastes good!",
		}
	} else {
		// Normal roll
		switch input.RollValue {
		case 2:
			messages = []string{
				"A 2! Not great, not terrible.",
				"Rolling a 2... barely better than a critical fail!",
				"A 2 appears! At least it's not a 1?",
				"The dice show 2. The dice are not your friend today.",
			}
		case 3:
			messages = []string{
				"A 3! Middle of the road.",
				"Rolling a 3... decidedly average.",
				"The dice land on 3. Could be worse!",
				"A 3 appears! Neither impressive nor embarrassing.",
			}
		case 4:
			messages = []string{
				"A 4! Getting better!",
				"Rolling a 4... not bad at all!",
				"The dice show 4. You're in good shape!",
				"A solid 4! You're doing alright.",
			}
		case 5:
			messages = []string{
				"A 5! So close to greatness!",
				"Rolling a 5... almost perfect!",
				"The dice show 5! Just shy of a critical hit!",
				"A 5! You can almost taste the victory.",
			}
		default:
			messages = []string{
				"The dice have spoken!",
				"The roll is complete!",
				"Let's see what fate has in store!",
				"The dice have decided your destiny!",
			}
		}
	}
	
	// Select a random message
	selectedMessage := messages[s.rand.Intn(len(messages))]
	
	return &GetRollResultMessageOutput{
		Message: selectedMessage,
		Tone:    tone,
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
