package api

import "qwen2api-go/toolcall"

func ResponsesRoutes() []RouteSpec {
	return []RouteSpec{
		{Method: "POST", Path: "/v1/responses", Auth: "api_key", Description: "OpenAI Responses compatible completion"},
	}
}

func ResponseOutputText(id, text string) map[string]any {
	return map[string]any{
		"id":          id,
		"object":      "response",
		"status":      "completed",
		"output":      []map[string]any{{"type": "message", "role": "assistant", "content": []map[string]any{{"type": "output_text", "text": text}}}},
		"output_text": text,
	}
}

func ResponseToolItems(calls []toolcall.ParsedToolCall) []map[string]any {
	return toolcall.ResponsesToolItems(calls)
}

func ResponseCompletedPayload(id, model, outputText string, output []map[string]any, usage map[string]any) map[string]any {
	if output == nil {
		output = []map[string]any{{"type": "message", "status": "completed", "role": "assistant", "content": []map[string]any{{"type": "output_text", "text": outputText, "annotations": []any{}}}}}
	}
	return map[string]any{
		"id": id, "object": "response", "created_at": 0, "status": "completed", "model": model,
		"output": output, "parallel_tool_calls": true, "error": nil, "incomplete_details": nil,
		"output_text": outputText, "usage": usage,
	}
}
