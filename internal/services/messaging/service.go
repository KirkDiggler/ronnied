package messaging

import (
	"context"
	"math/rand"
	"time"
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
				"Roll-off in progress! Your moment of glory (or shame) is coming soon.",
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

// GetGameStatusMessage returns a message describing the current game status
func (s *service) GetGameStatusMessage(ctx context.Context, input *GetGameStatusMessageInput) (*GetGameStatusMessageOutput, error) {
	var messages []string
	var tone MessageTone
	
	// Set default tone if not specified
	if input.PreferredTone == "" {
		tone = ToneNeutral
	} else {
		tone = input.PreferredTone
	}
	
	// Select messages based on game status
	if input.GameStatus.IsWaiting() {
		if input.ParticipantCount == 0 {
			messages = []string{
				"A new game is forming! Who will be the first to join?",
				"Fresh game, no players yet. Don't be shy!",
				"The dice are lonely. Won't someone join them?",
				"Game night is starting! Who's in?",
			}
		} else {
			messages = []string{
				"Waiting for more players or for the game to begin!",
				"The dice are warming up! Waiting for the game to start.",
				"Players are gathering... who else wants in?",
				"Almost ready to roll! Just waiting for the signal.",
			}
		}
	} else if input.GameStatus.IsActive() {
		messages = []string{
			"Game on! Time to roll those dice!",
			"The game is afoot! Roll for your dignity!",
			"Dice are hot! Who will be lucky today?",
			"Roll or forever hold your peace!",
		}
	} else if input.GameStatus.IsRollOff() {
		messages = []string{
			"Roll-off in progress! Who will emerge victorious?",
			"The tension builds during this epic roll-off!",
			"It's a roll-off! May the odds be ever in your favor.",
			"Roll-off time! This is where legends are made.",
		}
	} else {
		messages = []string{
			"Game complete! Check out who owes drinks!",
			"That's a wrap! Time to pay your debts.",
			"Game over! The tab awaits the unfortunate.",
			"The dice have spoken! Time to settle up.",
		}
	}
	
	// Select a random message
	selectedMessage := messages[s.rand.Intn(len(messages))]
	
	return &GetGameStatusMessageOutput{
		Message: selectedMessage,
		Tone:    tone,
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
