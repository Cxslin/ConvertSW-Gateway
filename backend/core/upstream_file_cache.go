package core

import (
	"sync"
	"time"
)

type UpstreamFile struct {
	LocalID    string
	UpstreamID string
	URL        string
	ExpiresAt  time.Time
}

type UpstreamFileCache struct {
	mu    sync.Mutex
	items map[string]UpstreamFile
}

func NewUpstreamFileCache() *UpstreamFileCache {
	return &UpstreamFileCache{items: map[string]UpstreamFile{}}
}

func (c *UpstreamFileCache) Put(item UpstreamFile) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[item.LocalID] = item
}

func (c *UpstreamFileCache) Get(localID string) (UpstreamFile, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[localID]
	if !ok {
		return UpstreamFile{}, false
	}
	if !item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt) {
		delete(c.items, localID)
		return UpstreamFile{}, false
	}
	return item, true
}

func (c *UpstreamFileCache) Cleanup(now time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	removed := 0
	for key, item := range c.items {
		if !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
			delete(c.items, key)
			removed++
		}
	}
	return removed
}
