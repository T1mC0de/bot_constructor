// Bot is the Telegram bot for buyers. It reads the survey from core and sends
// completed orders there, both over the core HTTP API.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/T1mC0de/bot_constructor/backend/bot/internal/coreapi"
	"github.com/T1mC0de/bot_constructor/backend/bot/internal/dialog"
	"github.com/T1mC0de/bot_constructor/backend/bot/internal/session"
	"github.com/T1mC0de/bot_constructor/backend/bot/internal/telegram"
)

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		slog.Error("TELEGRAM_BOT_TOKEN is not set")
		os.Exit(1)
	}
	coreURL := os.Getenv("CORE_URL")
	if coreURL == "" {
		coreURL = "http://localhost:8080"
	}

	d := dialog.New(coreapi.New(coreURL), session.NewMemory())

	// Ctrl+C or SIGTERM stops the bot.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := telegram.Run(ctx, token, d)
	stop()
	if err != nil {
		slog.Error("bot stopped with error", "err", err)
		os.Exit(1)
	}
}
