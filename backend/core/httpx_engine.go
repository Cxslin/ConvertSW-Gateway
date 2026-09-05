package core

import (
	cryptorand "crypto/rand"
	"fmt"
	"net/http"
	"time"
)

func NewHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 1800 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

func QwenHeaders(token string) http.Header {
	headers := http.Header{}
	headers.Set("Accept", "application/json, text/event-stream")
	headers.Set("Content-Type", "application/json")
	headers.Set("User-Agent", "Mozilla/5.0 qwen2api-go")
	headers.Set("x-request-id", QwenRequestID())
	if token != "" {
		headers.Set("Authorization", "Bearer "+token)
	}
	return headers
}

func QwenRequestID() string {
	var b [16]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		now := time.Now().UnixNano()
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", uint32(now), uint16(now>>32), uint16(now>>48), uint16(now>>16), uint64(now))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
