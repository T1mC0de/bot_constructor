// Package dialog runs a buyer through the survey: what to ask next, which
// answers to accept, when to create the order. It knows nothing about Telegram,
// so it is tested without network access.
package dialog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/T1mC0de/bot_constructor/backend/bot/internal/coreapi"
	"github.com/T1mC0de/bot_constructor/backend/bot/internal/session"
)

// MaxAnswerLen is the answer length limit (FR-6), in Unicode characters.
const MaxAnswerLen = 1000

// Texts the bot sends to the buyer.
const (
	TextNoSurvey    = "Анкета пока не настроена. Попробуйте позже."
	TextPressStart  = "Нажмите /start, чтобы оформить заявку."
	TextEmptyAnswer = "Это обязательный вопрос, напишите ответ."
	TextTooLong     = "Ответ слишком длинный — максимум 1000 символов."
	TextNotText     = "Пожалуйста, ответьте текстом."
	TextSaveFailed  = "Не удалось сохранить заявку. Отправьте последний ответ ещё раз."
	TextError       = "Что-то пошло не так. Попробуйте ещё раз."
)

// Backend is what the dialog needs from core. coreapi.Client implements it.
type Backend interface {
	GetSurvey(ctx context.Context) (coreapi.Survey, error)
	CreateOrder(ctx context.Context, order coreapi.Order) (coreapi.Order, error)
}

// Message is a buyer's message with the Telegram details stripped off.
type Message struct {
	ChatID   int64
	UserID   int64
	Username string
	Text     string // empty for stickers, photos, voice and other non-text messages
}

// Dialog is not safe for concurrent use: the Telegram side passes it one
// message at a time.
type Dialog struct {
	backend  Backend
	sessions session.Store // by chat ID
}

func New(backend Backend, sessions session.Store) *Dialog {
	return &Dialog{backend: backend, sessions: sessions}
}

// Handle processes one message and returns the replies to send, in order.
func (d *Dialog) Handle(ctx context.Context, msg Message) []string {
	if msg.Text == "/start" {
		return d.start(ctx, msg.ChatID)
	}
	s, ok, err := d.sessions.Get(ctx, msg.ChatID)
	if err != nil {
		slog.Error("cannot read session", "chat_id", msg.ChatID, "err", err)
		return []string{TextError}
	}
	if !ok {
		return []string{TextPressStart}
	}

	question := s.Survey.Questions[len(s.Answers)]
	answer := trimAnswer(msg.Text)
	switch {
	case msg.Text == "":
		return []string{TextNotText, question.Text}
	case answer == "":
		return []string{TextEmptyAnswer, question.Text}
	case utf8.RuneCountInString(answer) > MaxAnswerLen:
		return []string{TextTooLong, question.Text}
	}

	s.Answers = append(s.Answers, coreapi.Answer{QuestionID: question.ID, QuestionText: question.Text, Answer: answer})
	if len(s.Answers) < len(s.Survey.Questions) {
		if err := d.sessions.Save(ctx, msg.ChatID, s); err != nil {
			slog.Error("cannot save session", "chat_id", msg.ChatID, "err", err)
			return []string{TextError}
		}
		return []string{s.Survey.Questions[len(s.Answers)].Text}
	}

	order, err := d.backend.CreateOrder(ctx, coreapi.Order{
		TelegramUserID: msg.UserID,
		Username:       msg.Username,
		Answers:        s.Answers,
	})
	if err != nil {
		// The session is not saved with the last answer, so the buyer can send it again.
		slog.Error("cannot save order", "chat_id", msg.ChatID, "err", err)
		return []string{TextSaveFailed}
	}
	if err := d.sessions.Delete(ctx, msg.ChatID); err != nil {
		slog.Error("cannot delete session", "chat_id", msg.ChatID, "err", err)
	}
	return []string{fmt.Sprintf("Заявка №%d отправлена", order.ID)}
}

// start begins a new run; a previous run in this chat is replaced.
func (d *Dialog) start(ctx context.Context, chatID int64) []string {
	survey, err := d.backend.GetSurvey(ctx)
	if err != nil {
		slog.Warn("cannot start survey", "chat_id", chatID, "err", err)
		if err := d.sessions.Delete(ctx, chatID); err != nil {
			slog.Error("cannot delete session", "chat_id", chatID, "err", err)
		}
		return []string{TextNoSurvey}
	}
	if err := d.sessions.Save(ctx, chatID, session.Session{Survey: survey}); err != nil {
		slog.Error("cannot save session", "chat_id", chatID, "err", err)
		return []string{TextError}
	}
	return []string{survey.Greeting, survey.Questions[0].Text}
}

// trimAnswer strips spaces and invisible characters. Telegram does not send a
// message made only of spaces, so people send "empty" messages made of
// characters like U+2800 "⠀"; those count as empty answers too.
func trimAnswer(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		switch r {
		case '\u200B', '\u200C', '\u200D', '\u2060', '\uFEFF', '\u2800', '\u115F', '\u1160', '\u3164', '\uFFA0':
			return true
		}
		return unicode.IsSpace(r)
	})
}
