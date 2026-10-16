package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/T1mC0de/bot_constructor/backend/core/internal/model"
	"github.com/T1mC0de/bot_constructor/backend/core/internal/storage/jsonfile"
)

const validSurvey = `{
  "greeting": "Привет! Помогу оформить заказ торта.",
  "questions": [
    {"id": "date", "text": "На какую дату нужен торт?"},
    {"id": "weight", "text": "Какой вес торта?"}
  ]
}`

type testAPI struct {
	handler    http.Handler
	store      *jsonfile.Store
	surveyFile string
}

func newTestAPI(t *testing.T) *testAPI {
	dir := t.TempDir()
	surveyFile := filepath.Join(dir, "survey.json")
	store := jsonfile.New(surveyFile, filepath.Join(dir, "orders.json"))
	return &testAPI{handler: NewHandler(store), store: store, surveyFile: surveyFile}
}

func (a *testAPI) request(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("response is not JSON: %s", rec.Body)
	}
	return v
}

func TestHealth(t *testing.T) {
	a := newTestAPI(t)
	if rec := a.request("GET", "/healthz", ""); rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestGetSurveyNotUploaded(t *testing.T) {
	a := newTestAPI(t)
	if rec := a.request("GET", "/api/v1/survey", ""); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestUploadAndGetSurvey(t *testing.T) {
	a := newTestAPI(t)
	if rec := a.request("PUT", "/api/v1/survey", validSurvey); rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body %s", rec.Code, rec.Body)
	}

	rec := a.request("GET", "/api/v1/survey", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d", rec.Code)
	}
	got := decode[model.Survey](t, rec)
	if got.Greeting != "Привет! Помогу оформить заказ торта." || len(got.Questions) != 2 {
		t.Errorf("GET returned %+v", got)
	}
}

func TestUploadInvalidSurvey(t *testing.T) {
	a := newTestAPI(t)
	a.request("PUT", "/api/v1/survey", validSurvey)
	before, _ := os.ReadFile(a.surveyFile)

	invalid := `{"greeting": "", "questions": [{"id": "a", "text": "x"}, {"id": "a", "text": ""}], "title": "x"}`
	rec := a.request("PUT", "/api/v1/survey", invalid)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decode[errorResponse](t, rec); len(got.Details) != 4 {
		t.Errorf("details = %+v, want 4 errors", got.Details)
	}

	after, _ := os.ReadFile(a.surveyFile)
	if string(before) != string(after) {
		t.Error("survey file changed after an invalid upload")
	}
}

func TestUploadBrokenJSON(t *testing.T) {
	a := newTestAPI(t)
	if rec := a.request("PUT", "/api/v1/survey", `{"greeting":`); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestListOrders(t *testing.T) {
	a := newTestAPI(t)
	rec := a.request("GET", "/api/v1/orders", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("no orders: status %d, body %s; want 200 []", rec.Code, rec.Body)
	}

	a.store.CreateOrder(context.Background(), model.Order{
		TelegramUserID: 100,
		Username:       "anna",
		Answers:        []model.Answer{{QuestionID: "date", QuestionText: "Дата?", Answer: "20 октября"}},
	})
	orders := decode[[]model.Order](t, a.request("GET", "/api/v1/orders", ""))
	if len(orders) != 1 || orders[0].ID != 1 || orders[0].Answers[0].Answer != "20 октября" {
		t.Errorf("orders = %+v", orders)
	}
}

func TestCreateOrder(t *testing.T) {
	a := newTestAPI(t)
	order := `{"tg_user_id": 7, "username": "anna", "answers": [{"question_id": "date", "question_text": "Дата?", "answer": "20 октября"}]}`
	for want := int64(1); want <= 2; want++ {
		rec := a.request("POST", "/api/v1/orders", order)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
		}
		got := decode[model.Order](t, rec)
		if got.ID != want || got.Username != "anna" || got.ReceivedAt.Location() != time.UTC {
			t.Errorf("created %+v, want ID %d, received_at in UTC", got, want)
		}
	}
}

func TestCreateOrderBadJSON(t *testing.T) {
	a := newTestAPI(t)
	if rec := a.request("POST", "/api/v1/orders", `{"tg_user_id":`); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
