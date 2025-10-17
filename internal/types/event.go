package types

import (
	"context"
	"fmt"

	"github.com/zondax/fil-trace-check/api"
)

const (
	EventProviderBeryx = "beryx"
)

type EventProvider interface {
	GetAddressEventHeights(ctx context.Context, address string) ([]int64, error)
}

func NewEventProvider(eventProvider string, eventProviderToken string) (EventProvider, error) {
	switch eventProvider {
	case EventProviderBeryx:
		return api.NewBeryx(eventProviderToken), nil
	default:
		return nil, fmt.Errorf("unknown event provider: %s", eventProvider)
	}
}
