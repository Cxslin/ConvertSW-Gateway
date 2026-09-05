package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"
)

type LogFields map[string]any

func LogInfo(logger *slog.Logger, ctx context.Context, msg string, fields LogFields) {
	if logger == nil {
		return
	}
	logger.InfoContext(ctx, msg, attrs(ctx, fields)...)
}

func LogWarn(logger *slog.Logger, ctx context.Context, msg string, fields LogFields) {
	if logger == nil {
		return
	}
	logger.WarnContext(ctx, msg, attrs(ctx, fields)...)
}

func LogError(logger *slog.Logger, ctx context.Context, msg string, fields LogFields) {
	if logger == nil {
		return
	}
	logger.ErrorContext(ctx, msg, attrs(ctx, fields)...)
}

func attrs(ctx context.Context, fields LogFields) []any {
	out := []any{}
	if id := TraceID(ctx); id != "" {
		out = append(out, "trace_id", id)
	}
	for k, v := range FilterSensitiveFields(fields) {
		out = append(out, k, v)
	}
	return out
}

func RedactToken(token string) string {
	token = strings.TrimSpace(token)
	if len(token) <= 10 {
		return "***"
	}
	return token[:6] + "..." + token[len(token)-4:]
}

func PromptSHA256(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
