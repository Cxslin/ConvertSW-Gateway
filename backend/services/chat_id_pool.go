package services

import (
	"fmt"
	"strings"
	"time"
)

type WarmChat struct {
	Email     string
	Token     string
	Model     string
	ChatType  string
	ChatID    string
	ExpiresAt time.Time
}

type ModelWarmKey struct {
	Model    string
	ChatType string
}

func WarmChatKey(email, model, chatType string) string {
	return fmt.Sprintf("%s|%s|%s", strings.ToLower(strings.TrimSpace(email)), strings.TrimSpace(model), NormalizeWarmChatType(chatType))
}

func NormalizeWarmChatType(chatType string) string {
	if chatType == "image_gen" || chatType == "t2i" {
		return "t2i"
	}
	if strings.TrimSpace(chatType) == "" {
		return "t2t"
	}
	return chatType
}
