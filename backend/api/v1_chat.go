package api

import "qwen2api-go/toolcall"

func ChatRoutes() []RouteSpec {
	return []RouteSpec{
		{Method: "POST", Path: "/v1/chat/completions", Auth: "api_key", Description: "OpenAI compatible chat completion"},
	}
}

func ChatRequestSummary(body map[string]any) map[string]any {
	return map[string]any{
		"model":  body["model"],
		"stream": body["stream"],
		"tools":  body["tools"] != nil,
	}
}

func ChatCompletionPayload(id string, created int64, model, answer string, calls []toolcall.ParsedToolCall, usage map[string]any) map[string]any {
	message := map[string]any{"role": "assistant", "content": answer}
	finishReason := "stop"
	if len(calls) > 0 {
		message["content"] = nil
		message["tool_calls"] = toolcall.OpenAIToolCalls(calls)
		finishReason = "tool_calls"
	}
	return map[string]any{
		"id": id, "object": "chat.completion", "created": created, "model": model,
		"choices": []map[string]any{{"index": 0, "message": message, "finish_reason": finishReason}},
		"usage":   usage,
	}
}

func ChatCompletionChunk(id string, created int64, model string, delta map[string]any, finishReason any) map[string]any {
	return map[string]any{
		"id": id, "object": "chat.completion.chunk", "created": created, "model": model,
		"choices": []map[string]any{{"index": 0, "delta": delta, "finish_reason": finishReason}},
	}
}
