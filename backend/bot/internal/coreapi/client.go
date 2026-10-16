// Package coreapi is the bot's client for the core HTTP API.
// The types below describe the JSON of that API; they are the only thing the
// bot knows about core.
package coreapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Survey struct {
	Greeting  string     `json:"greeting"`
	Questions []Question `json:"questions"`
}

type Question struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Order struct {
	ID             int64    `json:"id,omitempty"` // assigned by core
	TelegramUserID int64    `json:"tg_user_id"`
	Username       string   `json:"username"`
	Answers        []Answer `json:"answers"`
}

type Answer struct {
	QuestionID   string `json:"question_id"`
	QuestionText string `json:"question_text"`
	Answer       string `json:"answer"`
}

// Client calls the core API.
type Client struct {
	url  string // e.g. http://localhost:8080
	http *http.Client
}

func New(url string) *Client {
	return &Client{url: url, http: &http.Client{Timeout: 10 * time.Second}}
}

// GetSurvey returns the current survey.
func (c *Client) GetSurvey(ctx context.Context) (Survey, error) {
	var survey Survey
	err := c.call(ctx, http.MethodGet, "/api/v1/survey", nil, &survey)
	return survey, err
}

// CreateOrder saves the order; core returns it with the number assigned.
func (c *Client) CreateOrder(ctx context.Context, order Order) (Order, error) {
	var created Order
	err := c.call(ctx, http.MethodPost, "/api/v1/orders", order, &created)
	return created, err
}

// call sends in as JSON (if not nil) and decodes the JSON response into out.
func (c *Client) call(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.url+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&e)
		return fmt.Errorf("%s %s: %s %s", method, path, resp.Status, e.Error)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
