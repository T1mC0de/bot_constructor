package model

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func validSurvey() Survey {
	return Survey{
		Greeting: "Привет! Помогу оформить заказ торта.",
		Questions: []Question{
			{ID: "date", Text: "На какую дату нужен торт?"},
			{ID: "weight", Text: "Какой вес торта?"},
		},
	}
}

func manyQuestions(n int) []Question {
	qs := make([]Question, n)
	for i := range qs {
		qs[i] = Question{ID: fmt.Sprintf("q%d", i), Text: "Вопрос?"}
	}
	return qs
}

// fields returns the fields of the errors, "" for errors without a field.
func fields(errs []FieldError) string {
	var out []string
	for _, e := range errs {
		out = append(out, e.Field)
	}
	return strings.Join(out, ",")
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		change func(s *Survey)
		want   string // comma-separated fields with errors
	}{
		{"valid", func(s *Survey) {}, ""},
		{"no greeting", func(s *Survey) { s.Greeting = "" }, "greeting"},
		{"greeting of spaces", func(s *Survey) { s.Greeting = "  \n " }, "greeting"},
		{"greeting 1000 chars", func(s *Survey) { s.Greeting = strings.Repeat("я", 1000) }, ""},
		{"greeting 1001 chars", func(s *Survey) { s.Greeting = strings.Repeat("я", 1001) }, "greeting"},
		{"no questions", func(s *Survey) { s.Questions = nil }, "questions"},
		{"20 questions", func(s *Survey) { s.Questions = manyQuestions(20) }, ""},
		{"21 questions", func(s *Survey) { s.Questions = manyQuestions(21) }, "questions"},
		{"no id", func(s *Survey) { s.Questions[0].ID = "" }, "questions[0].id"},
		{"id allowed chars", func(s *Survey) { s.Questions[0].ID = "Ab_9-z" }, ""},
		{"id 32 chars", func(s *Survey) { s.Questions[0].ID = strings.Repeat("a", 32) }, ""},
		{"id 33 chars", func(s *Survey) { s.Questions[0].ID = strings.Repeat("a", 33) }, "questions[0].id"},
		{"id in cyrillic", func(s *Survey) { s.Questions[0].ID = "дата" }, "questions[0].id"},
		{"id with space", func(s *Survey) { s.Questions[0].ID = "my date" }, "questions[0].id"},
		{"duplicate id", func(s *Survey) { s.Questions[1].ID = "date" }, "questions[1].id"},
		{"no text", func(s *Survey) { s.Questions[1].Text = " " }, "questions[1].text"},
		{"text 300 chars", func(s *Survey) { s.Questions[0].Text = strings.Repeat("ж", 300) }, ""},
		{"text 301 chars", func(s *Survey) { s.Questions[0].Text = strings.Repeat("ж", 301) }, "questions[0].text"},
		{
			"several errors at once",
			func(s *Survey) {
				s.Greeting = ""
				s.Questions = append(manyQuestions(20), Question{ID: "q0", Text: ""})
			},
			"greeting,questions,questions[20].id,questions[20].text",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := validSurvey()
			tt.change(&s)
			if got := fields(s.Validate()); got != tt.want {
				t.Errorf("errors in %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSurvey(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string // comma-separated fields with errors; "" stands for an error without a field
		n    int    // number of errors
	}{
		{"valid", `{"greeting":"Привет","questions":[{"id":"a","text":"Вопрос?"}]}`, "", 0},
		{"unknown field", `{"greeting":"Привет","questions":[{"id":"a","text":"Вопрос?"}],"title":"x"}`, "", 1},
		{"unknown question field", `{"greeting":"Привет","questions":[{"id":"a","text":"Вопрос?","type":"x"}]}`, "", 1},
		{"unknown field and rule errors", `{"greeting":"","questions":[],"title":"x"}`, "greeting,questions,", 3},
		{"broken JSON", `{"greeting":`, "", 1},
		{"wrong type", `{"greeting":42,"questions":[]}`, "", 1},
		{"empty object", `{}`, "greeting,questions", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := ParseSurvey([]byte(tt.json))
			if len(errs) != tt.n || fields(errs) != tt.want {
				t.Errorf("errors = %v, want %d in fields %q", errs, tt.n, tt.want)
			}
		})
	}
}

func TestParseSurveyUnknownFieldMessage(t *testing.T) {
	_, errs := ParseSurvey([]byte(`{"greeting":"Привет","questions":[{"id":"a","text":"Вопрос?"}],"title":"x"}`))
	if len(errs) != 1 || errs[0].Message != `неизвестное поле "title"` {
		t.Errorf("errors = %v", errs)
	}
}

func TestParseSurveyTrims(t *testing.T) {
	s, errs := ParseSurvey([]byte(`{"greeting":"  Привет!\n","questions":[{"id":"a","text":" Вопрос? "}]}`))
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if s.Greeting != "Привет!" || s.Questions[0].Text != "Вопрос?" {
		t.Errorf("got %+v, want trimmed texts", s)
	}
}

// The files in examples/ are used in the README and the demo.
func TestExamples(t *testing.T) {
	for file, wantErrors := range map[string]int{
		"survey_v1.json":      0,
		"survey_v2.json":      0,
		"survey_invalid.json": 5, // empty greeting, duplicate id, cyrillic id, empty text, unknown field
	} {
		data, err := os.ReadFile("../../examples/" + file)
		if err != nil {
			t.Fatal(err)
		}
		if _, errs := ParseSurvey(data); len(errs) != wantErrors {
			t.Errorf("%s: %d errors, want %d: %v", file, len(errs), wantErrors, errs)
		}
	}
}
