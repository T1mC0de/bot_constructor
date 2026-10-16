package dialog

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/T1mC0de/bot_constructor/backend/bot/internal/coreapi"
	"github.com/T1mC0de/bot_constructor/backend/bot/internal/session"
)

// fakeBackend keeps the survey and the orders in memory.
type fakeBackend struct {
	survey    coreapi.Survey
	surveyErr error
	orders    []coreapi.Order
	orderErr  error
}

func (f *fakeBackend) GetSurvey(context.Context) (coreapi.Survey, error) {
	return f.survey, f.surveyErr
}

func (f *fakeBackend) CreateOrder(_ context.Context, o coreapi.Order) (coreapi.Order, error) {
	if f.orderErr != nil {
		return coreapi.Order{}, f.orderErr
	}
	o.ID = int64(len(f.orders) + 1)
	o.Answers = append([]coreapi.Answer(nil), o.Answers...)
	f.orders = append(f.orders, o)
	return o, nil
}

var cakeSurvey = coreapi.Survey{
	Greeting: "Привет! Помогу оформить заказ торта.",
	Questions: []coreapi.Question{
		{ID: "date", Text: "На какую дату?"},
		{ID: "weight", Text: "Какой вес?"},
		{ID: "design", Text: "Пожелания к оформлению?"},
	},
}

type chat struct {
	t       *testing.T
	dialog  *Dialog
	backend *fakeBackend
}

func newChat(t *testing.T) *chat {
	backend := &fakeBackend{survey: cakeSurvey}
	return &chat{t: t, dialog: New(backend, session.NewMemory()), backend: backend}
}

// send sends text from buyer 7 (@anna) and checks the bot's replies.
func (c *chat) send(text string, want ...string) {
	c.t.Helper()
	got := c.dialog.Handle(context.Background(), Message{ChatID: 7, UserID: 7, Username: "anna", Text: text})
	if !reflect.DeepEqual(got, want) {
		c.t.Errorf("on %.20q got %q, want %q", text, got, want)
	}
}

func TestFullRun(t *testing.T) {
	c := newChat(t)
	c.send("/start", cakeSurvey.Greeting, "На какую дату?")
	c.send("  20 октября ", "Какой вес?")
	c.send("2 кг", "Пожелания к оформлению?")
	c.send("Белый, с розами", "Заявка №1 отправлена")

	if len(c.backend.orders) != 1 {
		t.Fatalf("%d orders, want 1", len(c.backend.orders))
	}
	o := c.backend.orders[0]
	want := []coreapi.Answer{
		{QuestionID: "date", QuestionText: "На какую дату?", Answer: "20 октября"},
		{QuestionID: "weight", QuestionText: "Какой вес?", Answer: "2 кг"},
		{QuestionID: "design", QuestionText: "Пожелания к оформлению?", Answer: "Белый, с розами"},
	}
	if o.TelegramUserID != 7 || o.Username != "anna" || !reflect.DeepEqual(o.Answers, want) {
		t.Errorf("order = %+v", o)
	}

	c.send("ещё", TextPressStart) // the run is over
}

func TestBadAnswersRepeatQuestion(t *testing.T) {
	tests := []struct{ name, text, reply string }{
		{"not text", "", TextNotText},
		{"spaces", " \n\t ", TextEmptyAnswer},
		{"invisible characters", "\u2800", TextEmptyAnswer},
		{"1001 characters", strings.Repeat("я", 1001), TextTooLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newChat(t)
			c.send("/start", cakeSurvey.Greeting, "На какую дату?")
			c.send("20 октября", "Какой вес?")
			c.send(tt.text, tt.reply, "Какой вес?")
			c.send(tt.text, tt.reply, "Какой вес?") // still the same question
			c.send("2 кг", "Пожелания к оформлению?")
			c.send("нет", "Заявка №1 отправлена")
			if got := c.backend.orders[0].Answers[1].Answer; got != "2 кг" {
				t.Errorf("answer to Q2 = %q", got)
			}
		})
	}
}

func TestAnswerOf1000Characters(t *testing.T) {
	c := newChat(t)
	c.send("/start", cakeSurvey.Greeting, "На какую дату?")
	c.send(strings.Repeat("я", 1000), "Какой вес?")
}

func TestMessageWithoutStart(t *testing.T) {
	c := newChat(t)
	c.send("Здравствуйте", TextPressStart)
	c.send("", TextPressStart)
}

func TestStartAgainMidRun(t *testing.T) {
	c := newChat(t)
	c.send("/start", cakeSurvey.Greeting, "На какую дату?")
	c.send("20 октября", "Какой вес?")
	c.send("/start", cakeSurvey.Greeting, "На какую дату?")
	c.send("21 октября", "Какой вес?")
	c.send("1 кг", "Пожелания к оформлению?")
	c.send("нет", "Заявка №1 отправлена")
	if got := c.backend.orders[0].Answers[0].Answer; got != "21 октября" {
		t.Errorf("answer to Q1 = %q, want the one from the new run", got)
	}
}

func TestNoSurvey(t *testing.T) {
	c := newChat(t)
	c.backend.surveyErr = errors.New("survey is not uploaded yet")
	c.send("/start", TextNoSurvey)
	c.send("ответ", TextPressStart)
}

func TestSurveyReplacedMidRun(t *testing.T) {
	c := newChat(t)
	c.send("/start", cakeSurvey.Greeting, "На какую дату?")
	c.send("20 октября", "Какой вес?")

	c.backend.survey = coreapi.Survey{Greeting: "Новая анкета", Questions: []coreapi.Question{{ID: "q", Text: "Новый вопрос?"}}}
	c.send("2 кг", "Пожелания к оформлению?") // the running session keeps the old survey
	c.send("нет", "Заявка №1 отправлена")

	c.send("/start", "Новая анкета", "Новый вопрос?") // the next run uses the new one
	c.send("да", "Заявка №2 отправлена")
}

func TestSaveFailure(t *testing.T) {
	c := newChat(t)
	c.send("/start", cakeSurvey.Greeting, "На какую дату?")
	c.send("20 октября", "Какой вес?")
	c.send("2 кг", "Пожелания к оформлению?")

	c.backend.orderErr = errors.New("disk full")
	c.send("нет", TextSaveFailed)

	c.backend.orderErr = nil
	c.send("нет", "Заявка №1 отправлена")
	if n := len(c.backend.orders[0].Answers); n != 3 {
		t.Errorf("%d answers, want 3", n)
	}
}

// brokenSessions fails every call, like a database that is down.
type brokenSessions struct{}

func (brokenSessions) Get(context.Context, int64) (session.Session, bool, error) {
	return session.Session{}, false, errors.New("db is down")
}
func (brokenSessions) Save(context.Context, int64, session.Session) error {
	return errors.New("db is down")
}
func (brokenSessions) Delete(context.Context, int64) error { return errors.New("db is down") }

func TestSessionStoreFailure(t *testing.T) {
	c := newChat(t)
	c.dialog = New(c.backend, brokenSessions{})
	c.send("/start", TextError)
	c.send("ответ", TextError)
}
