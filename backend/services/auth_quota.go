package services

import "time"

type QuotaState struct {
	ConsecutiveFailures int
	RateLimitStrikes    int
	RateLimitedUntil    time.Time
}

func (q QuotaState) Available(now time.Time) bool {
	return q.RateLimitedUntil.IsZero() || now.After(q.RateLimitedUntil)
}

func NextRateLimitCooldown(baseSeconds, maxSeconds, strikes int) time.Duration {
	if baseSeconds <= 0 {
		baseSeconds = 600
	}
	if maxSeconds <= 0 {
		maxSeconds = 3600
	}
	if strikes < 0 {
		strikes = 0
	}
	cooldown := baseSeconds
	for i := 0; i < strikes; i++ {
		cooldown *= 2
		if cooldown >= maxSeconds {
			return time.Duration(maxSeconds) * time.Second
		}
	}
	return time.Duration(cooldown) * time.Second
}
