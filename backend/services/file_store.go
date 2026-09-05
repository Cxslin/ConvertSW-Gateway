package services

import (
	"strings"
	"sync"
	"time"
)

type StoredFile struct {
	ID        string
	Name      string
	Path      string
	MimeType  string
	Size      int64
	CreatedAt time.Time
}

type FileStore struct {
	mu    sync.Mutex
	files map[string]StoredFile
}

func NewFileStore() *FileStore {
	return &FileStore{files: map[string]StoredFile{}}
}

func (s *FileStore) Put(file StoredFile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file.ID = strings.TrimSpace(file.ID)
	if file.CreatedAt.IsZero() {
		file.CreatedAt = time.Now()
	}
	s.files[file.ID] = file
}

func (s *FileStore) Get(id string) (StoredFile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, ok := s.files[strings.TrimSpace(id)]
	return file, ok
}

func (s *FileStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.files[id]; !ok {
		return false
	}
	delete(s.files, id)
	return true
}
