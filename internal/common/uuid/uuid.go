package uuid

import "github.com/google/uuid"

//go:generate mockgen -package=mocks -destination=mocks/mock_uuid.go github.com/KirkDiggler/ronnied/internal/common/uuid UUID

type UUID interface {
	NewUUID() string
}

// DefaultUUID implements the UUID interface using the uuid package

type DefaultUUID struct{}

func New() *DefaultUUID {
	return &DefaultUUID{}
}

// NewUUID returns a new UUID
func (d *DefaultUUID) NewUUID() string {
	return uuid.New().String()
}
