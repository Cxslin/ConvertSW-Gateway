package services

import (
	"regexp"
	"strings"

	"qwen2api-go/toolcall"
)

type ParsedToolCall = toolcall.ParsedToolCall
type ToolSieve = toolcall.ToolSieve
type SieveEvent = toolcall.SieveEvent

func ParseToolCalls(text string, tools []map[string]any) []ParsedToolCall {
	return toolcall.ParseToolCalls(text, tools)
}

func ParseToolCallsSilent(text string, tools []map[string]any) ([]map[string]any, string) {
	calls := toolcall.ParseToolCalls(text, tools)
	if len(calls) == 0 {
		return []map[string]any{{"type": "text", "text": text}}, "end_turn"
	}
	blocks := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		blocks = append(blocks, map[string]any{
			"type":  "tool_use",
			"id":    call.ID,
			"name":  call.Name,
			"input": call.Input,
		})
	}
	return blocks, "tool_use"
}

func OpenAIToolCalls(calls []ParsedToolCall) []map[string]any {
	return toolcall.OpenAIToolCalls(calls)
}

func NewToolSieve(tools []map[string]any) *ToolSieve {
	return toolcall.NewToolSieve(tools)
}

func CoerceToolInput(name string, input any, tools []map[string]any) any {
	return toolcall.CoerceToolInput(name, input, tools)
}

func HasTextualToolMarker(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	lowered := strings.ToLower(text)
	for _, marker := range []string{
		"<|qnml|tool_calls",
		"</|qnml|tool_calls",
		"<|qnml|invoke",
		"</|qnml|invoke",
		"<|qnml|parameter",
		"</|qnml|parameter",
		"<tool_calls",
		"</tool_calls",
		"<invoke",
		"</invoke",
		"<tool_call",
		"</tool_call",
		"##tool_call##",
		"##end_call##",
		"function.name:",
		"function.arguments:",
		"qnml|tool_calls",
		"qnml|invoke",
		"qnml|parameter",
	} {
		if strings.Contains(lowered, marker) {
			return true
		}
	}
	return false
}

func ExtractAttemptedToolName(text string, toolNames []string) string {
	if strings.TrimSpace(text) == "" || len(toolNames) == 0 {
		return ""
	}
	registry := map[string]string{}
	for _, name := range toolNames {
		if strings.TrimSpace(name) == "" {
			continue
		}
		registry[toolAliasKey(name)] = name
		registry[toolAliasKey(strings.ToLower(name))] = name
		if alias := qwenSafeToolAlias(name); alias != "" {
			registry[toolAliasKey(alias)] = name
		}
	}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?is)<\s*(?:\|\s*)?QNML(?:\s*\|\s*|\s+)?invoke\b[^>]*?name\s*=\s*(?:"([^"]+)"|'([^']+)'|([^\s>|/]+))`),
		regexp.MustCompile(`(?is)<\s*invoke\b[^>]*?name\s*=\s*(?:"([^"]+)"|'([^']+)'|([^\s>/]+))`),
		regexp.MustCompile(`(?is)"name"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`(?is)function\.name\s*:\s*([^\s\n]+)`),
	}
	for _, re := range patterns {
		for _, match := range re.FindAllStringSubmatch(text, -1) {
			for _, group := range match[1:] {
				candidate := strings.Trim(strings.TrimSpace(group), `"'`)
				if candidate == "" {
					continue
				}
				if canonical, ok := registry[toolAliasKey(candidate)]; ok {
					return canonical
				}
			}
		}
	}
	return ""
}

func toolAliasKey(value string) string {
	return regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "")
}

func qwenSafeToolAlias(name string) string {
	switch name {
	case "Read":
		return "fs_open_file"
	case "Write":
		return "fs_put_file"
	case "Edit":
		return "fs_patch_file"
	case "Bash":
		return "shell_run"
	case "Grep":
		return "text_search"
	case "Glob":
		return "path_find"
	case "NotebookEdit":
		return "notebook_patch"
	case "WebFetch":
		return "http_get_url"
	case "WebSearch":
		return "web_query"
	default:
		if strings.HasPrefix(name, "u_") {
			return name
		}
		return "u_" + name
	}
}
