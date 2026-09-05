package services

import "time"

type ExpiringItem interface {
	Expired(now time.Time) bool
}

func FilterExpired[T ExpiringItem](items []T, now time.Time) []T {
	out := items[:0]
	for _, item := range items {
		if !item.Expired(now) {
			out = append(out, item)
		}
	}
	return out
}

type ExpiringFile struct {
	StoredFile
	ExpiresAt time.Time
}

func (f ExpiringFile) Expired(now time.Time) bool {
	return !f.ExpiresAt.IsZero() && now.After(f.ExpiresAt)
}
