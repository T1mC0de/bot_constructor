// Package telegram connects the dialog to Telegram: it receives buyers'
// messages (long polling) and sends the dialog's replies back.
package telegram

import (
	"context"
	"log/slog"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/T1mC0de/bot_constructor/backend/bot/internal/dialog"
)

// Run connects to Telegram and answers buyers until ctx is cancelled.
// It returns an error if the token is wrong or Telegram is unreachable.
func Run(ctx context.Context, token string, d *dialog.Dialog) error {
	// The library calls this for every update and passes itself as tg.
	onUpdate := func(ctx context.Context, tg *tgbot.Bot, update *models.Update) {
		msg, ok := toMessage(update)
		if !ok {
			return
		}
		for _, text := range d.Handle(ctx, msg) {
			if _, err := tg.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: msg.ChatID, Text: text}); err != nil {
				slog.Error("cannot send message", "chat_id", msg.ChatID, "err", err)
				return
			}
		}
	}

	tg, err := tgbot.New(token,
		tgbot.WithDefaultHandler(onUpdate),
		// One update at a time, in the order they arrive.
		tgbot.WithNotAsyncHandlers(),
		tgbot.WithErrorsHandler(func(err error) { slog.Warn("telegram", "err", err) }),
	)
	if err != nil {
		return err
	}

	slog.Info("telegram bot started")
	tg.Start(ctx) // blocks until ctx is cancelled
	slog.Info("telegram bot stopped")
	return nil
}

// toMessage keeps only new messages from people in private chats;
// groups, channels and edited messages are ignored.
func toMessage(update *models.Update) (dialog.Message, bool) {
	m := update.Message
	if m == nil || m.Chat.Type != models.ChatTypePrivate {
		return dialog.Message{}, false
	}
	return dialog.Message{ChatID: m.Chat.ID, UserID: m.From.ID, Username: m.From.Username, Text: m.Text}, true
}
