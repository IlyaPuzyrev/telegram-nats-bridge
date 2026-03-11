package main

import (
	"context"
)

type BrokerInterface interface {
	Connect(ctx context.Context) error
	Publish(ctx context.Context, dest Destination, data interface{}) error
	Close() error
}
