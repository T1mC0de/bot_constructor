package jsonfile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/T1mC0de/bot_constructor/backend/core/internal/model"
	"github.com/T1mC0de/bot_constructor/backend/core/internal/storage"
)

var ctx = context.Background()

func newStore(t *testing.T) (*Store, string) {
	dir := t.TempDir()
	return New(filepath.Join(dir, "survey.json"), filepath.Join(dir, "orders.json")), dir
}

func testSurvey(greeting string) model.Survey {
	return model.Survey{Greeting: greeting, Questions: []model.Question{{ID: "date", Text: "На какую дату?"}}}
}

func testOrder(user int64, answers ...string) model.Order {
	o := model.Order{TelegramUserID: user}
	for i, a := range answers {
		o.Answers = append(o.Answers, model.Answer{QuestionID: fmt.Sprintf("q%d", i), QuestionText: "Вопрос?", Answer: a})
	}
	return o
}

func TestNoSurvey(t *testing.T) {
	s, _ := newStore(t)
	if _, err := s.GetSurvey(ctx); !errors.Is(err, storage.ErrNoSurvey) {
		t.Errorf("err = %v, want storage.ErrNoSurvey", err)
	}
}

func TestSaveAndReplaceSurvey(t *testing.T) {
	s, _ := newStore(t)
	for _, greeting := range []string{"Первая", "Вторая"} {
		if err := s.SaveSurvey(ctx, testSurvey(greeting)); err != nil {
			t.Fatal(err)
		}
		got, err := s.GetSurvey(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got.Greeting != greeting {
			t.Errorf("greeting = %q, want %q", got.Greeting, greeting)
		}
	}
}

func TestBrokenSurveyFile(t *testing.T) {
	s, dir := newStore(t)
	if err := os.WriteFile(filepath.Join(dir, "survey.json"), []byte(`{"greeting":`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetSurvey(ctx); err == nil || errors.Is(err, storage.ErrNoSurvey) {
		t.Errorf("err = %v, want an error about the broken file", err)
	}
}

func TestOrderIDsSurviveRestart(t *testing.T) {
	s, dir := newStore(t)
	for want := int64(1); want <= 3; want++ {
		o, err := s.CreateOrder(ctx, testOrder(want, "ответ"))
		if err != nil {
			t.Fatal(err)
		}
		if o.ID != want {
			t.Errorf("ID = %d, want %d", o.ID, want)
		}
		if o.ReceivedAt.Location() != time.UTC || time.Since(o.ReceivedAt) > time.Minute {
			t.Errorf("received_at = %v, want now in UTC", o.ReceivedAt)
		}
	}

	// A new Store on the same files is what the service sees after a restart.
	restarted := New(filepath.Join(dir, "survey.json"), filepath.Join(dir, "orders.json"))
	o, err := restarted.CreateOrder(ctx, testOrder(9, "ответ"))
	if err != nil {
		t.Fatal(err)
	}
	if o.ID != 4 {
		t.Errorf("ID after restart = %d, want 4", o.ID)
	}
}

func TestOrderSpecialCharacters(t *testing.T) {
	s, _ := newStore(t)
	want, err := s.CreateOrder(ctx, testOrder(1, `запятая, "кавычки"`, "перенос\nстроки", `\ / { } <b>`))
	if err != nil {
		t.Fatal(err)
	}
	orders, err := s.ListOrders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 1 || !reflect.DeepEqual(orders[0], want) {
		t.Errorf("orders = %+v, want [%+v]", orders, want)
	}
}

func TestNoOrders(t *testing.T) {
	s, _ := newStore(t)
	orders, err := s.ListOrders(ctx)
	if err != nil || orders == nil || len(orders) != 0 {
		t.Errorf("ListOrders() = %#v, %v; want an empty list", orders, err)
	}
}

func TestConcurrentOrders(t *testing.T) {
	s, _ := newStore(t)
	const buyers = 20
	var wg sync.WaitGroup
	for b := range buyers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.CreateOrder(ctx, testOrder(int64(b), "a", "b", "c")); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	orders, err := s.ListOrders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != buyers {
		t.Fatalf("%d orders, want %d", len(orders), buyers)
	}
	for i, o := range orders {
		if o.ID != int64(i+1) || len(o.Answers) != 3 {
			t.Errorf("order %d: ID %d, %d answers", i, o.ID, len(o.Answers))
		}
	}
}
