package dice

import (
	"math/rand"
	"time"
)

// Service implements dice rolling functionality
type Service struct {
	random *rand.Rand
}

// NewService creates a new dice service
func NewService() *Service {
	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)
	
	return &Service{
		random: random,
	}
}

// TODO: Implement interface methods
