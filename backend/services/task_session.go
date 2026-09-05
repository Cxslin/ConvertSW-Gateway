package services

import "time"

type TaskSession struct {
	ID        string
	Account   string
	Model     string
	ChatID    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewTaskSession(id, account, model, chatID string) TaskSession {
	now := time.Now()
	return TaskSession{ID: id, Account: account, Model: model, ChatID: chatID, CreatedAt: now, UpdatedAt: now}
}

func (s *TaskSession) Touch() {
	s.UpdatedAt = time.Now()
}
