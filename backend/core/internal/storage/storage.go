// Package storage describes how core keeps the survey and the orders.
// The rest of core works only with the Store interface, so the JSON files
// (package jsonfile) can be replaced with a database without touching the API.
package storage

import (
	"context"
	"errors"

	"github.com/T1mC0de/bot_constructor/backend/core/internal/model"
)

// ErrNoSurvey means the seller has not uploaded a survey yet.
var ErrNoSurvey = errors.New("survey is not uploaded yet")

type Store interface {
	// GetSurvey returns the current survey or ErrNoSurvey.
	GetSurvey(ctx context.Context) (model.Survey, error)
	// SaveSurvey replaces the current survey. The caller validates it first.
	SaveSurvey(ctx context.Context, survey model.Survey) error
	// CreateOrder assigns the order its number and the time it was received, and saves it.
	CreateOrder(ctx context.Context, order model.Order) (model.Order, error)
	// ListOrders returns all orders, oldest first.
	ListOrders(ctx context.Context) ([]model.Order, error)
}
