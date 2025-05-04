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
				"Look man, I would say I would pour you a drink, but I'm already busy.",
			}
		} else if input.GameStatus.IsActive() {
			messages = []string{
				"Ready to roll? Here's your dice button again. Don't lose it this time! 😉",
				"Found your dice button again! Try not to drop it this time.",
				"Look who came back for their dice! Roll away, my friend.",
				"The dice await your command... again. Let's hope they're luckier this time!",
				"Did you lose your dice... again? here you go bud",
			}
		} else if input.GameStatus.IsRollOff() {
			messages = []string{
				"Patience, young padawan! The roll-off is in progress. Your turn will come.",
				"Hold your horses! There's a roll-off happening.",
				"Roll-off in progress! This is where legends (and hangovers) are made.",
				"The tension builds during this roll-off! Stay tuned for your turn.",
			}
		} else if input.GameStatus.IsCompleted() {
			messages = []string{
				"You're still here? it's over, go home, you're drunk",
				"I know, I miss that game too, maybe start another one?",
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
			"Do you want drunk people? Because that's how you get drunk people!",
			"Welcome to the DANGER ZONE! Grab a drink and prepare to roll.",
			"Holy shitsnacks! A new player has joined the game!",
			"Phrasing! But seriously, welcome to the game.",
			"Just the tip... of the iceberg of fun you're about to have!",
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
			"Roll-off in progress! This is where legends (and hangovers) are made.",
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
				"CRIT!",
				"BOOM! Critical Hit!",
				"Nat 6!",
				"Perfect Roll!",
				"MAXIMUM DAMAGE!",
				"DANGER ZONE!",
				"PHRASING!",
			}

			messages := []string{
				"You rolled a 6! Time to make someone drink!",
				"Incredible! You just rolled a 6 and get to assign a drink!",
				"You're on fire! This means someone's about to get thirsty!",
				"The dice gods favor you today! That's a 6! Choose your victim!",
				"CRIT! You have the power to make someone drink!",
				"WOOOO! That's how you get ants! And by ants, I mean drinks for someone else!",
				"Holy shitsnacks! You rolled a 6! Time to make someone drink!",
				"Sploosh! That's a 6! You get to choose who drinks!",
				"Do you want drunk people? Because that's how you get drunk people!",
				"Lana. Lana. LANA! LANAAAA! You rolled a 6!",
			}

			title = titles[rand.Intn(len(titles))]
			message = messages[rand.Intn(len(messages))]
		} else {
			titles := []string{
				"CRIT!",
				"BOOM! Critical Hit!",
				"Nat 6!",
				"Perfect Roll!",
				"MAXIMUM DAMAGE!",
				"DANGER ZONE!",
				"PHRASING!",
			}

			messages := []string{
				fmt.Sprintf("%s rolled a 6! Time to make someone drink!", input.PlayerName),
				fmt.Sprintf("Incredible! %s just rolled a 6 and gets to assign a drink!", input.PlayerName),
				fmt.Sprintf("%s is on fire! A perfect 6 means someone's about to get thirsty!", input.PlayerName),
				fmt.Sprintf("The dice gods favor %s today! That's a 6! Choose your victim!", input.PlayerName),
				fmt.Sprintf("CRIT! %s has the power to make someone drink!", input.PlayerName),
				fmt.Sprintf("%s rolled a 6! Someone's getting a drink whether they like it or not!", input.PlayerName),
				fmt.Sprintf("Look at %s showing off with that 6! Now they get to choose who drinks!", input.PlayerName),
				fmt.Sprintf("A wild 6 appears for %s! Time to inflict some liquid damage!", input.PlayerName),
				fmt.Sprintf("%s just rolled a 6! Finally achieving something in life!", input.PlayerName),
				fmt.Sprintf("The chosen one! %s now wields the power of drink assignment!", input.PlayerName),
				fmt.Sprintf("Against all odds, %s somehow managed to roll a 6! Must be their birthday!", input.PlayerName),
				fmt.Sprintf("%s's 6 is the universe's way of saying someone needs to drink more!", input.PlayerName),
				fmt.Sprintf("%s just entered the DANGER ZONE with that 6! Time to assign a drink!", input.PlayerName),
				fmt.Sprintf("Holy shitsnacks! %s rolled a 6! Someone's about to get wasted!", input.PlayerName),
				fmt.Sprintf("Do you want drunk people? Because that's how %s gets drunk people! With a 6!", input.PlayerName),
				fmt.Sprintf("%s rolled a 6! Just the tip... of greatness!", input.PlayerName),
				fmt.Sprintf("Sploosh! %s rolled a 6 and gets to make someone drink!", input.PlayerName),
				fmt.Sprintf("%s: 'I swear I had something for this...' *rolls a 6*", input.PlayerName),
			}

			title = titles[rand.Intn(len(titles))]
			message = messages[rand.Intn(len(messages))]
		}

	case input.IsCriticalFail:
		// Critical fail (1)
		if isPersonal {
			titles := []string{
				"CRITICAL FAIL!",
				"Ouch! Snake Eye!",
				"Nat 1!",
				"MINIMUM DAMAGE!",
				"Better luck next time!",
				"I didn't see it, was it a good roll?",
				"DANGER ZONE!",
				"PHRASING!",
			}

			messages := []string{
				"You rolled a 1! Drink up!",
				"Oof! You just rolled a 1. Bottoms up!",
				"You angered the dice gods with that 1! Drink up, friend!",
				"Ronnie has spoken! You rolled a 1 and must take a drink!",
				"CRITICAL FAIL! You rolled a 1 and have to drink!",
				"Wow, that's... just... wow. You rolled a 1. Drink until you forget that happened.",
				"That's what I call a 'Pam-level' roll. A 1! Drink up!",
				"You rolled a 1! Do you want to get ants? Because that's how you get ants.",
				"That's a 1! I had something for this... something about drinking?",
				"Nooope! You rolled a 1. Time to drink away the shame.",
			}

			title = titles[rand.Intn(len(titles))]
			message = messages[rand.Intn(len(messages))]
		} else {
			titles := []string{
				"CRITICAL FAIL!",
				"Ouch! Snake Eye!",
				"Nat 1!",
				"MINIMUM DAMAGE!",
				"Better luck next time!",
				":trumpet: Sad Trumpet :(",
				"DANGER ZONE!",
				"PHRASING!",
			}

			messages := []string{
				fmt.Sprintf("%s rolled a 1! Time to drink up!", input.PlayerName),
				fmt.Sprintf("Oof! %s just rolled a 1. Bottoms up!", input.PlayerName),
				fmt.Sprintf("%s angered the dice gods with that 1! Drink up, friend!", input.PlayerName),
				fmt.Sprintf("Ronnie has spoken! %s rolled a 1 and must take a drink!", input.PlayerName),
				fmt.Sprintf("CRITICAL FAIL! %s rolled a 1 and has to drink!", input.PlayerName),
				fmt.Sprintf("%s rolled a 1! Exactly what we all expected from them!", input.PlayerName),
				fmt.Sprintf("The dice gods have forsaken %s with that 1! Drink to drown your sorrows!", input.PlayerName),
				fmt.Sprintf("A spectacular fail from %s with that 1! At least they're consistent!", input.PlayerName),
				fmt.Sprintf("%s's 1 is a perfect representation of their life choices so far!", input.PlayerName),
				fmt.Sprintf("The universe has spoken: %s rolled a 1 and needs alcohol to cope with it!", input.PlayerName),
				fmt.Sprintf("Even the dice are laughing at %s's pathetic 1! Drink up, buddy!", input.PlayerName),
				fmt.Sprintf("%s just proved they can't even roll dice properly with that 1! Drink to forget!", input.PlayerName),
				fmt.Sprintf("Wow, that's... just... wow. %s rolled a 1. Drink until you forget that happened.", input.PlayerName),
				fmt.Sprintf("%s rolled a 1! Guess they'll be entering the DANGER ZONE of drunkenness soon!", input.PlayerName),
				fmt.Sprintf("Holy shitsnacks! %s rolled a 1! That's like, the worst possible outcome!", input.PlayerName),
				fmt.Sprintf("%s: 'Wait, I had something for this...' *rolls a 1* 'Dammit!'", input.PlayerName),
				fmt.Sprintf("That's what I call a 'Pam-level' roll from %s. A 1! Drink up!", input.PlayerName),
				fmt.Sprintf("%s rolled a 1! Do you want to get drunk? Because that's how you get drunk.", input.PlayerName),
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
				"Meh.",
				"Just the tip... of the dice.",
			}

			messages := []string{
				fmt.Sprintf("You rolled a %d. Not bad!, not good either but not bad?", input.RollValue),
				fmt.Sprintf("The dice landed on %d. I guess that could work", input.RollValue),
				fmt.Sprintf("Your roll: %d - Keep trying!", input.RollValue),
				fmt.Sprintf("A solid %d! That works for you in your income bracket", input.RollValue),
				fmt.Sprintf("You rolled %d. The game continues!", input.RollValue),
				fmt.Sprintf("You rolled a %d. Are we not doing 'phrasing' anymore?", input.RollValue),
				fmt.Sprintf("A %d? I swear I had something for this...", input.RollValue),
				fmt.Sprintf("You rolled a %d. That's what we call a 'Cyril-level' performance.", input.RollValue),
				fmt.Sprintf("A %d. Not great, not terrible. Like a sad handjob.", input.RollValue),
				fmt.Sprintf("You rolled a %d. Meh, I've seen better. I've seen worse, but I've definitely seen better.", input.RollValue),
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
				"Meh.",
				"Just the tip... of the dice.",
			}

			messages := []string{
				fmt.Sprintf("%s rolled a %d. Not bad!", input.PlayerName, input.RollValue),
				fmt.Sprintf("The dice landed on %d for %s.", input.RollValue, input.PlayerName),
				fmt.Sprintf("%s's roll: %d - Keep trying!", input.PlayerName, input.RollValue),
				fmt.Sprintf("A solid %d from %s!", input.RollValue, input.PlayerName),
				fmt.Sprintf("%s rolled %d. The game continues!", input.PlayerName, input.RollValue),
				fmt.Sprintf("%s rolled a %d. That's what happens when you roll with those sweaty hands.", input.PlayerName, input.RollValue),
				fmt.Sprintf("%s got a %d. Exactly what you'd expect from someone with their gaming history.", input.PlayerName, input.RollValue),
				fmt.Sprintf("A %d from %s! That works for someone in their income bracket.", input.RollValue, input.PlayerName),
				fmt.Sprintf("%s rolled a %d. I've seen better, I've seen worse, but mostly I've seen better.", input.PlayerName, input.RollValue),
				fmt.Sprintf("The dice gods granted %s a %d. Must be all that clean living.", input.PlayerName, input.RollValue),
				fmt.Sprintf("%s's dice showed %d. Probably the highlight of their week.", input.PlayerName, input.RollValue),
				fmt.Sprintf("A %d appears for %s! Just like their dating life - mediocre but functional.", input.RollValue, input.PlayerName),
				fmt.Sprintf("%s rolled a %d. Let's all pretend to be impressed.", input.PlayerName, input.RollValue),
				fmt.Sprintf("The dice gave %s a %d. The dice never lie about a person's true worth.", input.PlayerName, input.RollValue),
				fmt.Sprintf("%s's %d is about as impressive as their choice of clothes tonight.", input.PlayerName, input.RollValue),
				fmt.Sprintf("%s rolled a %d. Are we not doing 'phrasing' anymore?", input.PlayerName, input.RollValue),
				fmt.Sprintf("%s: 'I swear I had something for this...' *rolls a %d*", input.PlayerName, input.RollValue),
				fmt.Sprintf("A %d from %s. Not great, not terrible. Like a sad handjob.", input.RollValue, input.PlayerName),
				fmt.Sprintf("%s rolled a %d. That's what we call a 'Cyril-level' performance.", input.PlayerName, input.RollValue),
				fmt.Sprintf("%s got a %d. Meh, I've seen better. I've seen worse, but I've definitely seen better.", input.PlayerName, input.RollValue),
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
		"Let the dice decide your fate! Roll now!",
		"Time to test your luck! Click to roll the dice!",
		"The game has begun! Roll the dice and see what destiny has in store!",
		"Ready, set, ROLL! Click the button to throw your dice!",
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
		tone = MessageToneFunny
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
			"Roll-off in progress! This is where legends (and hangovers) are made.",
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

// GetRollWhisperMessage returns a supportive whisper message after a roll
func (s *service) GetRollWhisperMessage(ctx context.Context, input *GetRollWhisperMessageInput) (*GetRollWhisperMessageOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	var messages []string
	var tone MessageTone

	// Set default tone if not specified
	if input.PreferredTone == "" {
		tone = MessageToneEncouraging // Default to encouraging tone for whispers
	} else {
		tone = input.PreferredTone
	}

	// Select messages based on roll type
	switch {
	case input.IsCriticalHit:
		// Critical hit (6)
		messages = []string{
			fmt.Sprintf("Hey %s, between you and me, that was a killer roll! Make 'em drink good!", input.PlayerName),
			fmt.Sprintf("*whispers* %s, I knew you had it in you! Now make someone suffer!", input.PlayerName),
			fmt.Sprintf("Don't tell anyone, but that's the best roll I've seen all day, %s!", input.PlayerName),
			fmt.Sprintf("*quietly* %s, you're my favorite player. That 6 was destiny!", input.PlayerName),
			fmt.Sprintf("Just between us, %s, I might have nudged that dice a little for you. Our secret!", input.PlayerName),
			fmt.Sprintf("*whispers* Welcome to the DANGER ZONE, %s! That 6 is your ticket to power!", input.PlayerName),
			fmt.Sprintf("Hey %s, do you want drunk people? Because that's how you get drunk people! Nice 6!", input.PlayerName),
			fmt.Sprintf("*quietly* Sploosh! That 6 was amazing, %s. Now make someone pay!", input.PlayerName),
			fmt.Sprintf("Between us, %s, I'm pretty sure that roll just got you a seat on the highway to the DANGER ZONE!", input.PlayerName),
			fmt.Sprintf("*whispers* That's how you roll dice, %s! Other Barry would be proud.", input.PlayerName),
		}

	case input.IsCriticalFail:
		// Critical fail (1)
		messages = []string{
			fmt.Sprintf("*whispers* Hey %s, don't worry about that 1. We all have bad days!", input.PlayerName),
			fmt.Sprintf("Between you and me, %s, I've seen worse rolls... well, actually I haven't, but chin up!", input.PlayerName),
			fmt.Sprintf("*quietly* %s, that drink might help you forget that terrible roll!", input.PlayerName),
			fmt.Sprintf("Don't let the others see you sweat, %s. Act like you meant to roll that 1!", input.PlayerName),
			fmt.Sprintf("Hey %s, at least you're consistent! Consistently unlucky, but still...", input.PlayerName),
			fmt.Sprintf("*whispers* Holy shitsnacks, %s! That was... not great. Drink to forget!", input.PlayerName),
			fmt.Sprintf("Between us, %s, that roll was what I call a classic Cyril. Total failure.", input.PlayerName),
			fmt.Sprintf("*quietly* %s, you just entered a whole new DANGER ZONE of failure with that 1!", input.PlayerName),
			fmt.Sprintf("Phrasing! But seriously %s, that 1 was pretty bad. Have a drink, you need it.", input.PlayerName),
			fmt.Sprintf("*whispers* %s, I'd say that roll was disappointing, but that would imply I expected more from you.", input.PlayerName),
		}

	default:
		// Normal roll (2-5)
		if input.RollValue <= 3 {
			// Lower normal rolls
			messages = []string{
				fmt.Sprintf("*whispers* Not great, not terrible, %s. You'll get 'em next time!", input.PlayerName),
				fmt.Sprintf("Between us, %s, a %d isn't so bad considering your usual luck!", input.PlayerName, input.RollValue),
				fmt.Sprintf("*quietly* %s, I've seen you roll worse. This is practically a win!", input.PlayerName),
				fmt.Sprintf("Hey %s, at least it wasn't a 1! Small victories, right?", input.PlayerName),
				fmt.Sprintf("Don't tell anyone, but I think the dice are warming up to you, %s!", input.PlayerName),
				fmt.Sprintf("*whispers* %s, that roll was like a sad handjob. Technically does the job, but nobody's excited about it.", input.PlayerName),
				fmt.Sprintf("Between us, %s, that %d is what we call a 'Pam after five drinks' level of performance.", input.PlayerName, input.RollValue),
				fmt.Sprintf("*quietly* %s, I had something for this... something about mediocrity?", input.PlayerName),
				fmt.Sprintf("Do you want to be average? Because that's how you get average, %s.", input.PlayerName),
				fmt.Sprintf("*whispers* Just the tip... of mediocrity with that %d, %s.", input.RollValue, input.PlayerName),
			}
		} else {
			// Higher normal rolls
			messages = []string{
				fmt.Sprintf("*whispers* Nice roll, %s! Just one away from greatness!", input.PlayerName),
				fmt.Sprintf("Between us, %s, that %d is pretty solid. Not a 6, but respectable!", input.PlayerName, input.RollValue),
				fmt.Sprintf("*quietly* %s, you're getting better at this! Keep it up!", input.PlayerName),
				fmt.Sprintf("Hey %s, with rolls like that, you might just survive this game!", input.PlayerName),
				fmt.Sprintf("Don't tell the others, but I'm rooting for you, %s! That was a good one!", input.PlayerName),
				fmt.Sprintf("*whispers* %s, that's what I call a solid roll. Not quite the DANGER ZONE, but close!", input.PlayerName),
				fmt.Sprintf("Between us, %s, that %d is like Woodhouse - reliable but not exciting.", input.PlayerName, input.RollValue),
				fmt.Sprintf("*quietly* %s, you're entering the... zone of... adequate rolling? I had something for this.", input.PlayerName),
				fmt.Sprintf("Sploosh! Well, more like a splash. Nice %d, %s.", input.RollValue, input.PlayerName),
				fmt.Sprintf("*whispers* %s, that roll was like a turtle neck - tactically solid but not flashy.", input.PlayerName),
			}
		}
	}

	// Select a random message
	selectedMessage := messages[s.rand.Intn(len(messages))]

	return &GetRollWhisperMessageOutput{
		Message: selectedMessage,
		Tone:    tone,
	}, nil
}

// GetLeaderboardMessage returns a funny message for a player in the leaderboard
func (s *service) GetLeaderboardMessage(ctx context.Context, input *GetLeaderboardMessageInput) (*GetLeaderboardMessageOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	var message string

	// Different messages based on rank
	if input.Rank == 0 { // First place
		messages := []string{
			fmt.Sprintf("%s is the champion of drinking delegation! %d drinks assigned - their liver thanks them for the outsourcing.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("First place: %s with %d drinks! The puppetmaster of alcohol consumption!", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("All hail %s, who managed to make others drink %d times! Truly a master of the dice.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("%s takes the gold with %d drinks assigned! Their strategy? Pure luck and zero remorse.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("The crown goes to %s with %d drinks! Remember this moment, it's all downhill from here.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("%s dominated with %d drinks assigned! Exactly what we'd expect from someone in their income bracket.", input.PlayerName, input.DrinkCount),
		}
		message = messages[s.rand.Intn(len(messages))]
	} else if input.Rank == 1 { // Second place
		messages := []string{
			fmt.Sprintf("Silver medalist %s assigned %d drinks. So close to greatness, yet so far.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("%s takes second place with %d drinks assigned. First loser is still a loser!", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("Almost impressive: %s with %d drinks. Maybe try harder next time?", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("%s got %d drinks assigned - the silver medal of making others suffer!", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("Second place: %s with %d drinks. Nobody remembers second place, but we'll try.", input.PlayerName, input.DrinkCount),
		}
		message = messages[s.rand.Intn(len(messages))]
	} else if input.Rank == 2 { // Third place
		messages := []string{
			fmt.Sprintf("Bronze tier: %s with %d drinks assigned. At least you made the podium!", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("%s managed %d drinks. Bronze is just a fancy word for third place.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("Third place goes to %s with %d drinks. Not great, not terrible, just mediocre.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("%s: %d drinks assigned and a bronze medal. It's like winning, but worse!", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("The bronze medal of drink delegation goes to %s with %d drinks. You're technically on the podium!", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("%s takes the bronze with %d drinks assigned. Better luck next time, maybe?", input.PlayerName, input.DrinkCount),
		}
		message = messages[s.rand.Intn(len(messages))]
	} else if input.Rank == input.TotalPlayers-1 { // Last place
		messages := []string{
			fmt.Sprintf("Dead last: %s with %d drinks. Even the dice feel sorry for you.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("%s: %d drinks assigned. You're so bad at this it's almost impressive.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("Last place: %s with %d drinks. There's always next game to disappoint everyone again!", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("%s assigned %d drinks. If 'participation trophy' was a person.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("And then there's %s with %d drinks. Thanks for showing up, I guess?", input.PlayerName, input.DrinkCount),
		}
		message = messages[s.rand.Intn(len(messages))]
	} else { // Middle of the pack
		messages := []string{
			fmt.Sprintf("%s: %d drinks assigned. Perfectly average, just like their choice of clothing.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("Middle of the pack: %s with %d drinks. Not good, not bad, just... there.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("%s managed to assign %d drinks. The embodiment of 'meh'.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("%s: %d drinks. Aggressively mediocre performance as usual.", input.PlayerName, input.DrinkCount),
			fmt.Sprintf("With %d drinks assigned, %s continues their lifelong streak of being unremarkable.", input.DrinkCount, input.PlayerName),
		}
		message = messages[s.rand.Intn(len(messages))]
	}

	return &GetLeaderboardMessageOutput{
		Message: message,
	}, nil
}

// GetPayDrinkMessage returns a fun message when a player pays a drink
func (s *service) GetPayDrinkMessage(ctx context.Context, input *GetPayDrinkMessageInput) (*GetPayDrinkMessageOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	// Fun titles for paying drinks
	titles := []string{
		"Debt Cleared!",
		"Tab Paid!",
		"Drink Settled!",
		"Cheers to That!",
		"Bottoms Up!",
		"DANGER ZONE!",
		"Phrasing!",
		"Sploosh!",
	}

	// Fun messages for paying drinks
	messages := []string{
		fmt.Sprintf("%s chugs with dignity and honor! 🍻", input.PlayerName),
		fmt.Sprintf("%s takes their medicine like a champ! 💪", input.PlayerName),
		fmt.Sprintf("%s settles their tab with the drinking gods! 🙏", input.PlayerName),
		fmt.Sprintf("%s drinks up and lives to roll another day! 🎲", input.PlayerName),
		fmt.Sprintf("The bartender nods approvingly as %s pays their dues! 🍸", input.PlayerName),
		fmt.Sprintf("%s raises their glass in a toast to bad luck and good friends! 🥂", input.PlayerName),
		fmt.Sprintf("%s accepts defeat gracefully and drinks up! 🏳️", input.PlayerName),
		fmt.Sprintf("A moment of silence as %s downs their drink... and it's gone! 💨", input.PlayerName),
		fmt.Sprintf("%s proves they're a player of honor by paying their drink debt! 🎖️", input.PlayerName),
		fmt.Sprintf("The drinking gods are pleased with %s's sacrifice! 🔱", input.PlayerName),
		fmt.Sprintf("%s enters the DANGER ZONE of drunkenness! 🚨", input.PlayerName),
		fmt.Sprintf("%s: \"I swear I had something for this...\" *chugs drink* 🍺", input.PlayerName),
		fmt.Sprintf("Holy shitsnacks! %s actually paid their drink! 🎉", input.PlayerName),
		fmt.Sprintf("%s drinks like a champion! Just the tip... of the glass! 🥃", input.PlayerName),
		fmt.Sprintf("Sploosh! %s downs that drink like a pro! 💦", input.PlayerName),
		fmt.Sprintf("%s drinks like Pam on a Tuesday morning! Impressive! 🏆", input.PlayerName),
		fmt.Sprintf("Do you want hangovers? Because that's how %s gets hangovers! 🤢", input.PlayerName),
		fmt.Sprintf("%s: \"Lana. Lana. LANA! LANAAAA!\" *drinks* \"...Danger Zone.\" 🥴", input.PlayerName),
	}

	// Select random title and message
	selectedTitle := titles[s.rand.Intn(len(titles))]
	selectedMessage := messages[s.rand.Intn(len(messages))]

	return &GetPayDrinkMessageOutput{
		Title:   selectedTitle,
		Message: selectedMessage,
	}, nil
}
