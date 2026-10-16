package coreapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeCore answers like the core API does.
func fakeCore(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/survey", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"greeting":"Привет","questions":[{"id":"date","text":"На какую дату?"}]}`)
	})
	mux.HandleFunc("POST /api/v1/orders", func(w http.ResponseWriter, r *http.Request) {
		var o Order
		if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
			t.Errorf("bad request body: %v", err)
		}
		o.ID = 42
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(o)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestGetSurvey(t *testing.T) {
	core := fakeCore(t)
	survey, err := New(core.URL).GetSurvey(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if survey.Greeting != "Привет" || len(survey.Questions) != 1 || survey.Questions[0].ID != "date" {
		t.Errorf("survey = %+v", survey)
	}
}

func TestGetSurveyError(t *testing.T) {
	core := fakeCore(t)
	if _, err := New(core.URL + "/nowhere").GetSurvey(context.Background()); err == nil {
		t.Error("no error for 404")
	}
	if _, err := New("http://127.0.0.1:1").GetSurvey(context.Background()); err == nil {
		t.Error("no error when core is down")
	}
}

func TestCreateOrder(t *testing.T) {
	core := fakeCore(t)
	order := Order{TelegramUserID: 7, Username: "anna", Answers: []Answer{{QuestionID: "date", QuestionText: "На какую дату?", Answer: "20 октября"}}}
	created, err := New(core.URL).CreateOrder(context.Background(), order)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != 42 || created.Username != "anna" || created.Answers[0].Answer != "20 октября" {
		t.Errorf("created = %+v", created)
	}
}
