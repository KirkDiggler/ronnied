package dice

//go:generate mockgen -package=mocks -destination=mocks/mock_roller.go github.com/KirkDiggler/ronnied/internal/dice Roller

import (
	"math/rand"
	"time"
)

// Roller provides dice rolling functionality
type Roller interface {
	// Roll generates a random dice roll with the specified number of sides
	Roll(sides int) int
}

// roller implements the Roller interface
type roller struct {
	random *rand.Rand
}

// Config for dice roller
type Config struct {
	// Optional seed for testing
	Seed int64
}

// New creates a new dice roller
func New(cfg *Config) Roller {
	var seed int64
	if cfg != nil && cfg.Seed != 0 {
		seed = cfg.Seed
	} else {
		seed = time.Now().UnixNano()
	}

	source := rand.NewSource(seed)
	random := rand.New(source)

	return &roller{
		random: random,
	}
}

// Roll generates a random dice roll with the specified number of sides
func (r *roller) Roll(sides int) int {
	if sides < 1 {
		sides = 6 // Default to 6-sided die
	}

	return r.random.Intn(sides) + 1
}
