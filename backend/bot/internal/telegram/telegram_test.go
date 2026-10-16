package telegram

import (
	"testing"

	"github.com/go-telegram/bot/models"

	"github.com/T1mC0de/bot_constructor/backend/bot/internal/dialog"
)

func TestToMessage(t *testing.T) {
	user := &models.User{ID: 7, Username: "anna"}
	private := models.Chat{ID: 7, Type: models.ChatTypePrivate}
	group := models.Chat{ID: -1, Type: models.ChatTypeGroup}

	tests := []struct {
		name   string
		update models.Update
		want   dialog.Message
		ok     bool
	}{
		{"text in private chat", models.Update{Message: &models.Message{Chat: private, From: user, Text: "привет"}},
			dialog.Message{ChatID: 7, UserID: 7, Username: "anna", Text: "привет"}, true},
		{"sticker in private chat", models.Update{Message: &models.Message{Chat: private, From: user, Sticker: &models.Sticker{}}},
			dialog.Message{ChatID: 7, UserID: 7, Username: "anna"}, true},
		{"group", models.Update{Message: &models.Message{Chat: group, From: user, Text: "/start"}}, dialog.Message{}, false},
		{"edited message", models.Update{EditedMessage: &models.Message{Chat: private, From: user, Text: "x"}}, dialog.Message{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toMessage(&tt.update)
			if got != tt.want || ok != tt.ok {
				t.Errorf("toMessage() = %+v, %v; want %+v, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}
