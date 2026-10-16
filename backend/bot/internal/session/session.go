// Package session keeps the buyers' unfinished surveys. The dialog works only
// with the Store interface: today the sessions live in memory (Memory), later
// they can move to a database or Redis to survive a restart.
package session

import (
	"context"

	"github.com/T1mC0de/bot_constructor/backend/bot/internal/coreapi"
)

// Session is one buyer's way through the survey.
type Session struct {
	Survey  coreapi.Survey // read on /start: replacing the survey does not affect a running session
	Answers []coreapi.Answer
}

type Store interface {
	// Get returns the session of the chat; ok is false if there is none.
	Get(ctx context.Context, chatID int64) (s Session, ok bool, err error)
	Save(ctx context.Context, chatID int64, s Session) error
	Delete(ctx context.Context, chatID int64) error
}

// Memory keeps sessions in a map; they are lost on restart.
// It is not safe for concurrent use, like the dialog that uses it.
type Memory struct {
	sessions map[int64]Session
}

var _ Store = (*Memory)(nil)

func NewMemory() *Memory {
	return &Memory{sessions: map[int64]Session{}}
}

func (m *Memory) Get(_ context.Context, chatID int64) (Session, bool, error) {
	s, ok := m.sessions[chatID]
	return s, ok, nil
}

func (m *Memory) Save(_ context.Context, chatID int64, s Session) error {
	m.sessions[chatID] = s
	return nil
}

func (m *Memory) Delete(_ context.Context, chatID int64) error {
	delete(m.sessions, chatID)
	return nil
}
