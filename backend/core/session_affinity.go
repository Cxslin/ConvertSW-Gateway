package core

import (
	"sync"
	"time"
)

type AffinityRecord struct {
	SessionID string
	Email     string
	ChatID    string
	Model     string
	ChatType  string
	ExpiresAt time.Time
}

type SessionAffinity struct {
	mu      sync.Mutex
	records map[string]AffinityRecord
}

func NewSessionAffinity() *SessionAffinity {
	return &SessionAffinity{records: map[string]AffinityRecord{}}
}

func (s *SessionAffinity) Put(record AffinityRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[record.SessionID] = record
}

func (s *SessionAffinity) Get(sessionID string) (AffinityRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[sessionID]
	if !ok {
		return AffinityRecord{}, false
	}
	if !record.ExpiresAt.IsZero() && time.Now().After(record.ExpiresAt) {
		delete(s.records, sessionID)
		return AffinityRecord{}, false
	}
	return record, true
}

func (s *SessionAffinity) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, sessionID)
}
