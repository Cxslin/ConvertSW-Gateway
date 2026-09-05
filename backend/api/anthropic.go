package api

import "qwen2api-go/toolcall"

func AnthropicRoutes() []RouteSpec {
	return []RouteSpec{
		{Method: "POST", Path: "/v1/messages", Auth: "api_key", Description: "Anthropic messages compatibility"},
		{Method: "POST", Path: "/v1/messages/count_tokens", Auth: "api_key", Description: "Anthropic token counting compatibility"},
	}
}

func AnthropicMessagePayload(id, model, text string) map[string]any {
	return map[string]any{
		"id": id, "type": "message", "role": "assistant", "model": model,
		"content":       []map[string]any{{"type": "text", "text": text}},
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
	}
}

func AnthropicToolUsePayload(id, model string, calls []toolcall.ParsedToolCall) map[string]any {
	blocks := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		input := call.Input
		if input == nil {
			input = map[string]any{}
		}
		blocks = append(blocks, map[string]any{
			"type":  "tool_use",
			"id":    call.ID,
			"name":  call.Name,
			"input": input,
		})
	}
	return map[string]any{
		"id": id, "type": "message", "role": "assistant", "model": model,
		"content":       blocks,
		"stop_reason":   "tool_use",
		"stop_sequence": nil,
	}
}

func AnthropicUsage(inputTokens, outputTokens int) map[string]int {
	return map[string]int{"input_tokens": inputTokens, "output_tokens": outputTokens}
}
