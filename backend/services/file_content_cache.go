package services

import (
	"sync"
	"time"
)

type CachedContent struct {
	Text      string
	ExpiresAt time.Time
}

type FileContentCache struct {
	mu    sync.Mutex
	items map[string]CachedContent
}

func NewFileContentCache() *FileContentCache {
	return &FileContentCache{items: map[string]CachedContent{}}
}

func (c *FileContentCache) Put(id, text string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[id] = CachedContent{Text: text, ExpiresAt: time.Now().Add(ttl)}
}

func (c *FileContentCache) Get(id string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[id]
	if !ok || (!item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt)) {
		delete(c.items, id)
		return "", false
	}
	return item.Text, true
}
