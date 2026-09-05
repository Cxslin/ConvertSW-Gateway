package services

import "encoding/json"

func OpenAIChatChunk(id string, created int64, model string, delta map[string]any, finish any) string {
	payload := map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]any{{"index": 0, "delta": delta, "finish_reason": finish}},
	}
	raw, _ := json.Marshal(payload)
	return "data: " + string(raw) + "\n\n"
}

func OpenAIDoneChunk() string {
	return "data: [DONE]\n\n"
}
