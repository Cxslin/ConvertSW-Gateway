package services

func ChatCompletionPayload(id string, created int64, model, answer string, usage map[string]any) map[string]any {
	return map[string]any{
		"id":      id,
		"object":  "chat.completion",
		"created": created,
		"model":   model,
		"choices": []map[string]any{{
			"index":         0,
			"message":       map[string]any{"role": "assistant", "content": answer},
			"finish_reason": "stop",
		}},
		"usage": usage,
	}
}

func MediaGenerationPayload(created int64, urls []string, kind string) map[string]any {
	data := []map[string]any{}
	for _, url := range urls {
		item := map[string]any{"url": url}
		if kind != "" {
			item["type"] = kind
		}
		data = append(data, item)
	}
	return map[string]any{"created": created, "data": data}
}
