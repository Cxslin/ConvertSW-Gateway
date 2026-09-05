package account_pool

import (
	"strings"
	"sync"
	"time"
)

type Account struct {
	Email               string  `json:"email"`
	Password            string  `json:"password"`
	Token               string  `json:"token"`
	Cookies             string  `json:"cookies"`
	Username            string  `json:"username"`
	ActivationPending   bool    `json:"activation_pending"`
	StatusCode          string  `json:"status_code"`
	LastError           string  `json:"last_error"`
	LastRequestStarted  float64 `json:"last_request_started"`
	LastRequestFinished float64 `json:"last_request_finished"`
	ConsecutiveFailures int     `json:"consecutive_failures"`
	RateLimitStrikes    int     `json:"rate_limit_strikes"`
	Valid               bool    `json:"valid,omitempty"`
	Inflight            int     `json:"inflight,omitempty"`
	RateLimitedUntil    float64 `json:"rate_limited_until,omitempty"`
}

type Pool struct {
	mu                    sync.Mutex
	accounts              []*Account
	maxInflightPerAccount int
	globalMaxInflight     int
	globalInUse           int
}

func New(accounts []Account, maxInflightPerAccount, globalMaxInflight int) *Pool {
	p := &Pool{
		maxInflightPerAccount: max(1, maxInflightPerAccount),
		globalMaxInflight:     globalMaxInflight,
	}
	for i := range accounts {
		acc := accounts[i]
		acc.Normalize()
		p.accounts = append(p.accounts, &acc)
	}
	return p
}

func (p *Pool) Snapshot() []Account {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Account, 0, len(p.accounts))
	for _, acc := range p.accounts {
		out = append(out, *acc)
	}
	return out
}

func (p *Pool) Add(acc Account) {
	p.mu.Lock()
	defer p.mu.Unlock()
	acc.Normalize()
	p.accounts = append(p.accounts, &acc)
}

func (p *Pool) Remove(email string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	email = strings.ToLower(strings.TrimSpace(email))
	for i, acc := range p.accounts {
		if strings.ToLower(acc.Email) == email {
			p.accounts = append(p.accounts[:i], p.accounts[i+1:]...)
			return true
		}
	}
	return false
}

func (a *Account) Normalize() {
	a.Email = strings.TrimSpace(a.Email)
	a.Token = strings.TrimSpace(a.Token)
	if a.StatusCode == "" && a.Token != "" {
		a.StatusCode = "active"
	}
	if a.Token != "" && !a.ActivationPending && a.StatusCode != "invalid" {
		a.Valid = true
	}
}

func (a *Account) Available(maxInflight int) bool {
	if a == nil || !a.Valid || a.ActivationPending || strings.TrimSpace(a.Token) == "" {
		return false
	}
	if a.RateLimitedUntil > float64(time.Now().Unix()) {
		return false
	}
	return a.Inflight < max(1, maxInflight)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
