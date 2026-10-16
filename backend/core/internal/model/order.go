package model

import "time"

// Order is a completed survey ("заявка"). Each answer keeps a copy of the
// question text, so old orders stay readable after the survey is replaced.
type Order struct {
	ID             int64     `json:"id"`
	ReceivedAt     time.Time `json:"received_at"` // UTC
	TelegramUserID int64     `json:"tg_user_id"`
	Username       string    `json:"username"` // without "@"; may be empty
	Answers        []Answer  `json:"answers"`
}

type Answer struct {
	QuestionID   string `json:"question_id"`
	QuestionText string `json:"question_text"`
	Answer       string `json:"answer"`
}
