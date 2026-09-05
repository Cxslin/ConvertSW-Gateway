package services

import (
	"net/http"
	"strings"
)

type AuthCredentials struct {
	Token   string
	Cookies string
	Email   string
}

func ResolveBearerToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	if token := strings.TrimSpace(r.URL.Query().Get("api_key")); token != "" {
		return token
	}
	return strings.TrimSpace(r.Header.Get("X-API-Key"))
}

func ResolveQwenCredentials(headers http.Header) AuthCredentials {
	return AuthCredentials{
		Token:   strings.TrimSpace(headers.Get("X-Qwen-Token")),
		Cookies: strings.TrimSpace(headers.Get("X-Qwen-Cookies")),
		Email:   strings.TrimSpace(headers.Get("X-Qwen-Account")),
	}
}
