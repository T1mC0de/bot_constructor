// Package api is the HTTP API of the core service. The seller uploads the survey
// and downloads orders; the bot reads the survey and creates orders.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/T1mC0de/bot_constructor/backend/core/internal/model"
	"github.com/T1mC0de/bot_constructor/backend/core/internal/storage"
)

type server struct {
	store storage.Store
}

// NewHandler returns the API handler. The API has no authorization: in
// iteration 1 it runs locally only.
func NewHandler(store storage.Store) http.Handler {
	s := &server{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/v1/survey", s.getSurvey)
	mux.HandleFunc("PUT /api/v1/survey", s.putSurvey)
	mux.HandleFunc("GET /api/v1/orders", s.listOrders)
	mux.HandleFunc("POST /api/v1/orders", s.createOrder)
	return mux
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) getSurvey(w http.ResponseWriter, r *http.Request) {
	survey, err := s.store.GetSurvey(r.Context())
	if errors.Is(err, storage.ErrNoSurvey) {
		writeError(w, http.StatusNotFound, "анкета ещё не загружена", nil)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, survey)
}

func (s *server) putSurvey(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "не удалось прочитать тело запроса (максимум 1 МБ)", nil)
		return
	}
	survey, errs := model.ParseSurvey(body)
	if len(errs) > 0 {
		writeError(w, http.StatusBadRequest, "анкета содержит ошибки", errs)
		return
	}
	if err := s.store.SaveSurvey(r.Context(), survey); err != nil {
		internalError(w, err)
		return
	}
	slog.Info("survey replaced", "questions", len(survey.Questions))
	writeJSON(w, http.StatusOK, survey)
}

func (s *server) listOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := s.store.ListOrders(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

// createOrder is called by the bot when a buyer has answered every question.
// The core assigns the order number and the time.
func (s *server) createOrder(w http.ResponseWriter, r *http.Request) {
	var order model.Order
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&order); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON: "+err.Error(), nil)
		return
	}
	order, err := s.store.CreateOrder(r.Context(), order)
	if err != nil {
		internalError(w, err)
		return
	}
	slog.Info("order created", "order_id", order.ID)
	writeJSON(w, http.StatusCreated, order)
}

type errorResponse struct {
	Error   string             `json:"error"`
	Details []model.FieldError `json:"details,omitempty"`
}

func writeError(w http.ResponseWriter, status int, msg string, details []model.FieldError) {
	writeJSON(w, status, errorResponse{Error: msg, Details: details})
}

func internalError(w http.ResponseWriter, err error) {
	slog.Error("api error", "err", err)
	writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервера", nil)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
