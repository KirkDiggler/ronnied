package clock

import "time"

//go:generate mockgen -package=mocks -destination=mocks/mock_clock.go github.com/KirkDiggler/ronnied/internal/common/clock Clock
type Clock interface {
	Now() time.Time
}

// DefaultClock implements the Clock interface using the system clock
type DefaultClock struct{}

// New returns a new DefaultClock
func New() *DefaultClock {
	return &DefaultClock{}
}

// Now returns the current time
func (c *DefaultClock) Now() time.Time {
	return time.Now()
}
