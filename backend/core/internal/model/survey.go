// Package model holds the data types shared by the API, the storage and the bot.
package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Survey limits from the SRS (FR-2, FR-3). Lengths are counted in Unicode
// characters, not bytes: the texts are in Russian.
const (
	MaxGreetingLen     = 1000
	MaxQuestions       = 20
	MaxQuestionTextLen = 300
)

// Survey is what the seller uploads: a greeting and free-text questions asked in order.
// All questions are required.
type Survey struct {
	Greeting  string     `json:"greeting"`
	Questions []Question `json:"questions"`
}

type Question struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// FieldError is one problem in a survey. Field is empty when the problem is not
// tied to a single field (broken JSON, unknown field).
type FieldError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

var questionIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// ParseSurvey decodes a survey from JSON, trims spaces around the texts and
// validates it. It returns every problem found, not just the first one.
func ParseSurvey(data []byte) (Survey, []FieldError) {
	var s Survey
	if err := json.Unmarshal(data, &s); err != nil {
		return Survey{}, []FieldError{{Message: "некорректный JSON: " + err.Error()}}
	}
	s.Greeting = strings.TrimSpace(s.Greeting)
	for i := range s.Questions {
		s.Questions[i].Text = strings.TrimSpace(s.Questions[i].Text)
	}
	errs := s.Validate()

	// json.Unmarshal silently ignores unknown fields; a strict decoder reports
	// them (the first one only, which is enough to spot a typo).
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&Survey{}); err != nil {
		msg := strings.Replace(err.Error(), "json: unknown field", "неизвестное поле", 1)
		errs = append(errs, FieldError{Message: msg})
	}
	return s, errs
}

// Validate checks the rules of FR-2 and FR-3 and returns all violations.
func (s Survey) Validate() []FieldError {
	var errs []FieldError
	add := func(field, format string, args ...any) {
		errs = append(errs, FieldError{Field: field, Message: fmt.Sprintf(format, args...)})
	}

	switch n := utf8.RuneCountInString(strings.TrimSpace(s.Greeting)); {
	case n == 0:
		add("greeting", "обязательное поле")
	case n > MaxGreetingLen:
		add("greeting", "не длиннее %d символов, сейчас %d", MaxGreetingLen, n)
	}

	switch n := len(s.Questions); {
	case n == 0:
		add("questions", "нужен хотя бы один вопрос")
	case n > MaxQuestions:
		add("questions", "не больше %d вопросов, сейчас %d", MaxQuestions, n)
	}

	seen := map[string]bool{}
	for i, q := range s.Questions {
		switch {
		case q.ID == "":
			add(fmt.Sprintf("questions[%d].id", i), "обязательное поле")
		case !questionIDPattern.MatchString(q.ID):
			add(fmt.Sprintf("questions[%d].id", i), "только латиница, цифры, «_» и «-», не длиннее 32 символов")
		case seen[q.ID]:
			add(fmt.Sprintf("questions[%d].id", i), "id %q уже есть в анкете", q.ID)
		}
		seen[q.ID] = true

		switch n := utf8.RuneCountInString(strings.TrimSpace(q.Text)); {
		case n == 0:
			add(fmt.Sprintf("questions[%d].text", i), "обязательное поле")
		case n > MaxQuestionTextLen:
			add(fmt.Sprintf("questions[%d].text", i), "не длиннее %d символов, сейчас %d", MaxQuestionTextLen, n)
		}
	}
	return errs
}
