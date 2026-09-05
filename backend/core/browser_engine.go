package core

import "time"

type BrowserOptions struct {
	Headless bool
	Timeout  time.Duration
	PoolSize int
}

func DefaultBrowserOptions(settings Settings) BrowserOptions {
	timeout := time.Duration(settings.BrowserStreamTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 1800 * time.Second
	}
	poolSize := settings.BrowserPoolSize
	if poolSize <= 0 {
		poolSize = 1
	}
	return BrowserOptions{Headless: true, Timeout: timeout, PoolSize: poolSize}
}
