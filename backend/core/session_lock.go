package core

import "sync"

type SessionLocks struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewSessionLocks() *SessionLocks {
	return &SessionLocks{locks: map[string]*sync.Mutex{}}
}

func (s *SessionLocks) Lock(key string) func() {
	if key == "" {
		key = "default"
	}
	s.mu.Lock()
	lock := s.locks[key]
	if lock == nil {
		lock = &sync.Mutex{}
		s.locks[key] = lock
	}
	s.mu.Unlock()
	lock.Lock()
	return lock.Unlock
}
