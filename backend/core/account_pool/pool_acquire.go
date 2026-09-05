package account_pool

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrNoAccount = errors.New("no available qwen account")

func (p *Pool) Acquire(ctx context.Context, preferredEmail string) (*Account, error) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if acc := p.pick(preferredEmail); acc != nil {
			return acc, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (p *Pool) pick(preferredEmail string) *Account {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.globalMaxInflight > 0 && p.globalInUse >= p.globalMaxInflight {
		return nil
	}
	preferredEmail = strings.ToLower(strings.TrimSpace(preferredEmail))
	var best *Account
	for _, acc := range p.accounts {
		if preferredEmail != "" && strings.ToLower(acc.Email) != preferredEmail {
			continue
		}
		if !acc.Available(p.maxInflightPerAccount) {
			continue
		}
		if best == nil || acc.Inflight < best.Inflight {
			best = acc
		}
	}
	if best == nil && preferredEmail != "" {
		for _, acc := range p.accounts {
			if acc.Available(p.maxInflightPerAccount) && (best == nil || acc.Inflight < best.Inflight) {
				best = acc
			}
		}
	}
	if best == nil {
		return nil
	}
	best.Inflight++
	best.LastRequestStarted = float64(time.Now().UnixMilli()) / 1000
	p.globalInUse++
	return best
}

func (p *Pool) Release(acc *Account) {
	if acc == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if acc.Inflight > 0 {
		acc.Inflight--
	}
	if p.globalInUse > 0 {
		p.globalInUse--
	}
	acc.LastRequestFinished = float64(time.Now().UnixMilli()) / 1000
}

func (p *Pool) MarkSuccess(acc *Account) {
	if acc == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	acc.Valid = true
	acc.StatusCode = "active"
	acc.LastError = ""
	acc.ConsecutiveFailures = 0
	acc.RateLimitStrikes = 0
}

func (p *Pool) MarkInvalid(acc *Account, status, message string) {
	if acc == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	acc.Valid = false
	acc.StatusCode = status
	acc.LastError = message
	acc.ConsecutiveFailures++
}

func (p *Pool) MarkRateLimited(acc *Account, cooldownSeconds int, message string) {
	if acc == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	acc.RateLimitStrikes++
	acc.LastError = message
	acc.RateLimitedUntil = float64(time.Now().Add(time.Duration(cooldownSeconds) * time.Second).Unix())
}
