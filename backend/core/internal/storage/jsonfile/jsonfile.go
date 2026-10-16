// Package jsonfile is the storage.Store kept in two JSON files.
package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/T1mC0de/bot_constructor/backend/core/internal/model"
	"github.com/T1mC0de/bot_constructor/backend/core/internal/storage"
)

// Store reads and writes survey.json and orders.json. The context is not
// used: reading a local file cannot be cancelled anyway.
type Store struct {
	surveyPath string
	ordersPath string
	mu         sync.Mutex // one writer at a time
}

var _ storage.Store = (*Store)(nil)

func New(surveyPath, ordersPath string) *Store {
	return &Store{surveyPath: surveyPath, ordersPath: ordersPath}
}

func (s *Store) GetSurvey(_ context.Context) (model.Survey, error) {
	data, err := os.ReadFile(s.surveyPath)
	if errors.Is(err, fs.ErrNotExist) {
		return model.Survey{}, storage.ErrNoSurvey
	}
	if err != nil {
		return model.Survey{}, err
	}
	var survey model.Survey
	if err := json.Unmarshal(data, &survey); err != nil {
		return model.Survey{}, fmt.Errorf("%s: %w", s.surveyPath, err)
	}
	return survey, nil
}

func (s *Store) SaveSurvey(_ context.Context, survey model.Survey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeJSON(s.surveyPath, survey)
}

// CreateOrder numbers orders as last ID + 1.
func (s *Store) CreateOrder(_ context.Context, order model.Order) (model.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	orders, err := s.readOrders()
	if err != nil {
		return model.Order{}, err
	}
	order.ReceivedAt = time.Now().UTC().Truncate(time.Second)
	order.ID = 1
	if len(orders) > 0 {
		order.ID = orders[len(orders)-1].ID + 1
	}
	if err := writeJSON(s.ordersPath, append(orders, order)); err != nil {
		return model.Order{}, err
	}
	return order, nil
}

func (s *Store) ListOrders(_ context.Context) ([]model.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readOrders()
}

func (s *Store) readOrders() ([]model.Order, error) {
	orders := []model.Order{}
	data, err := os.ReadFile(s.ordersPath)
	if errors.Is(err, fs.ErrNotExist) {
		return orders, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &orders); err != nil {
		return nil, fmt.Errorf("%s: %w", s.ordersPath, err)
	}
	return orders, nil
}

// writeJSON replaces the file atomically: it writes a temporary file and renames
// it over the old one, so a reader never sees a half-written file.
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
