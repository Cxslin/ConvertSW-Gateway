package services

import "time"

type GarbageCollector struct {
	lastRun time.Time
}

func (g *GarbageCollector) ShouldRun(now time.Time, interval time.Duration) bool {
	if interval <= 0 {
		interval = time.Minute
	}
	return g.lastRun.IsZero() || now.Sub(g.lastRun) >= interval
}

func (g *GarbageCollector) MarkRun(now time.Time) {
	g.lastRun = now
}
