package toolcall

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
)

// ParsedToolCall is the normalized OpenAI-compatible function call extracted
// from Qwen text output.
type ParsedToolCall struct {
	ID    string
	Name  string
	Input any
}

// ParseToolCalls extracts tool calls from XML/QNML-style and JSON fragments.
func ParseToolCalls(text string, tools []map[string]any) []ParsedToolCall {
	if strings.TrimSpace(text) == "" || len(tools) == 0 {
		return nil
	}
	allowed := map[string]string{}
	for _, tool := range tools {
		name := stringValue(tool, "name", "")
		if name == "" {
			continue
		}
		allowed[strings.ToLower(name)] = name
	}
	if len(allowed) == 0 {
		return nil
	}
	calls := []ParsedToolCall{}
	calls = append(calls, parseQNMLToolCalls(text, allowed, tools)...)
	calls = append(calls, parseXMLToolCalls(text, allowed)...)
	ForEachJSONFragment(text, func(value any) {
		calls = append(calls, parseJSONToolCalls(value, allowed)...)
	})
	calls = append(calls, parseTextKVToolCalls(text, allowed, tools)...)
	return DedupeToolCalls(coerceToolCalls(calls, tools))
}

// DedupeToolCalls removes repeated calls with the same name and arguments.
func DedupeToolCalls(calls []ParsedToolCall) []ParsedToolCall {
	seen := map[string]bool{}
	out := []ParsedToolCall{}
	for _, call := range calls {
		keyBytes, _ := json.Marshal(call.Input)
		key := strings.ToLower(call.Name) + "\x00" + string(keyBytes)
		if call.Name == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, call)
	}
	return out
}

// OpenAIToolCalls renders normalized calls as OpenAI chat.completions tool_calls.
func OpenAIToolCalls(calls []ParsedToolCall) []map[string]any {
	out := []map[string]any{}
	for _, call := range calls {
		args := ""
		switch v := call.Input.(type) {
		case string:
			args = v
		default:
			args = mustJSON(firstNonNil(v, map[string]any{}))
		}
		out = append(out, map[string]any{
			"id":       call.ID,
			"type":     "function",
			"function": map[string]any{"name": call.Name, "arguments": args},
		})
	}
	return out
}

// ResponsesToolItems renders normalized calls as OpenAI Responses API items.
func ResponsesToolItems(calls []ParsedToolCall) []map[string]any {
	out := []map[string]any{}
	for _, call := range calls {
		args := call.Input
		if _, ok := args.(string); !ok {
			args = mustJSON(firstNonNil(args, map[string]any{}))
		}
		out = append(out, map[string]any{
			"id":        "fc_" + randomID()[:12],
			"type":      "function_call",
			"status":    "completed",
			"call_id":   call.ID,
			"name":      call.Name,
			"arguments": args,
		})
	}
	return out
}

func randomID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(buf)
}

func stringValue(m map[string]any, key, fallback string) string {
	if m == nil {
		return fallback
	}
	if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func firstString(values ...any) string {
	for _, value := range values {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func mustJSON(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func coerceToolCalls(calls []ParsedToolCall, tools []map[string]any) []ParsedToolCall {
	if len(calls) == 0 {
		return calls
	}
	out := make([]ParsedToolCall, 0, len(calls))
	for _, call := range calls {
		call.Input = CoerceToolInput(call.Name, call.Input, tools)
		if missingRequiredArgs(call.Name, call.Input, tools) {
			continue
		}
		if invalidToolArgs(call.Input) {
			continue
		}
		out = append(out, call)
	}
	return out
}

func invalidToolArgs(input any) bool {
	m, ok := input.(map[string]any)
	if !ok {
		return false
	}
	for key, value := range m {
		if !isPathLikeArgName(key) {
			continue
		}
		if pathLikeArgLooksPolluted(toString(value)) {
			return true
		}
	}
	return false
}

func isPathLikeArgName(name string) bool {
	normalized := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "")
	switch normalized {
	case "path", "filepath", "filename", "targetfile", "file", "dir", "directory", "cwd", "workdir", "workingdirectory":
		return true
	default:
		return false
	}
}

func pathLikeArgLooksPolluted(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.ContainsRune(trimmed, '\x00') {
		return true
	}
	if strings.ContainsAny(trimmed, "\r\n<>") {
		return true
	}
	lowered := strings.ToLower(trimmed)
	for _, marker := range []string{
		"<![cdata[",
		"]]>",
		"qnml|",
		"|qnml",
		"tool_calls",
		"toolcalls",
		"invoke name=",
		"parameter name=",
		"</parameter",
		"</invoke",
		"function.name:",
		"function.arguments:",
	} {
		if strings.Contains(lowered, marker) {
			return true
		}
	}
	for _, marker := range []string{
		"我将", "我会", "我先", "首先", "现在", "接下来", "继续执行", "开始执行", "目录已创建",
		"i will", "i'll", "i am going to", "now i", "next i", "first i",
	} {
		if strings.Contains(lowered, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

// CoerceToolInput mirrors the Python bridge's practical argument repair layer.
// It keeps Qwen's small naming mistakes from reaching Claude Code as invalid
// calls, while preserving unknown tools and unknown fields.
func CoerceToolInput(name string, input any, tools []map[string]any) any {
	input = coerceToolInputBySchema(name, input, tools)
	m, ok := input.(map[string]any)
	if !ok {
		return input
	}
	fixed := cloneMap(m)
	switch name {
	case "AskUserQuestion":
		if question, ok := fixed["question"]; ok && fixed["questions"] == nil {
			delete(fixed, "question")
			fixed["questions"] = []any{map[string]any{
				"question":    question,
				"header":      "Question",
				"multiSelect": false,
				"options": []any{
					map[string]any{"label": "Yes", "description": "Confirm"},
					map[string]any{"label": "No", "description": "Decline"},
				},
			}}
		}
		if q, ok := fixed["questions"]; ok {
			if _, isList := q.([]any); !isList {
				fixed["questions"] = []any{q}
			}
		}
	case "Agent":
		if fixed["description"] == nil {
			fixed["description"] = "Execute sub-task"
		}
		if fixed["prompt"] == nil {
			fixed["prompt"] = fixed["description"]
		}
	case "Read":
		renameFirstPresent(fixed, "file_path", "path", "filename", "file")
	case "Write":
		renameFirstPresent(fixed, "file_path", "path", "target_file", "filename", "file")
		renameFirstPresent(fixed, "content", "text", "body", "data", "file_content", "contents", "value")
	case "Edit":
		renameFirstPresent(fixed, "file_path", "path", "target_file", "filename", "file")
	case "Bash", "PowerShell":
		renameFirstPresent(fixed, "command", "cmd", "script")
	default:
		if fixed["query"] == nil {
			if queries, ok := fixed["queries"]; ok && toolAcceptsField(name, tools, "query") {
				switch v := queries.(type) {
				case []any:
					parts := []string{}
					for _, item := range v {
						if s := strings.TrimSpace(toString(item)); s != "" {
							parts = append(parts, s)
						}
					}
					if len(parts) > 0 {
						delete(fixed, "queries")
						fixed["query"] = strings.Join(parts, "\n")
					}
				case string:
					if strings.TrimSpace(v) != "" {
						delete(fixed, "queries")
						fixed["query"] = strings.TrimSpace(v)
					}
				}
			}
		}
	}
	return fixed
}

func coerceToolInputBySchema(name string, input any, tools []map[string]any) any {
	m, ok := input.(map[string]any)
	if !ok {
		return input
	}
	props := schemaProperties(toolSchema(name, tools))
	if len(props) == 0 {
		return input
	}
	fixed := cloneMap(m)
	for key, value := range fixed {
		if schema, ok := props[key].(map[string]any); ok {
			fixed[key] = coerceValueBySchema(value, schema)
		}
	}
	return fixed
}

func coerceValueBySchema(value any, schema map[string]any) any {
	types := schemaTypes(schema)
	wantArray := types["array"]
	wantObject := types["object"]
	if s, ok := value.(string); ok && (wantArray || wantObject) {
		if parsed, changed := parseJSONStringForSchema(s, wantArray, wantObject); changed {
			value = parsed
		}
	}
	if wantArray {
		if m, ok := value.(map[string]any); ok {
			value = []any{m}
		}
		if list, ok := value.([]any); ok {
			if itemSchema, ok := schema["items"].(map[string]any); ok {
				out := make([]any, 0, len(list))
				for _, item := range list {
					out = append(out, coerceValueBySchema(item, itemSchema))
				}
				return out
			}
		}
		return value
	}
	if wantObject {
		if m, ok := value.(map[string]any); ok {
			props := schemaProperties(schema)
			if len(props) == 0 {
				return value
			}
			fixed := cloneMap(m)
			for key, child := range props {
				if childSchema, ok := child.(map[string]any); ok {
					if childValue, exists := fixed[key]; exists {
						fixed[key] = coerceValueBySchema(childValue, childSchema)
					}
				}
			}
			return fixed
		}
	}
	return value
}

func parseJSONStringForSchema(value string, wantArray, wantObject bool) (any, bool) {
	stripped := strings.TrimSpace(value)
	if stripped == "" {
		return value, false
	}
	candidates := []string{stripped}
	if wantArray && !strings.HasPrefix(stripped, "[") {
		candidates = append(candidates, "["+stripped+"]")
	}
	for _, candidate := range candidates {
		var parsed any
		if json.Unmarshal([]byte(candidate), &parsed) != nil {
			continue
		}
		if wantArray {
			if _, ok := parsed.([]any); ok {
				return parsed, true
			}
			if m, ok := parsed.(map[string]any); ok {
				return []any{m}, true
			}
		}
		if wantObject {
			if _, ok := parsed.(map[string]any); ok {
				return parsed, true
			}
		}
	}
	return value, false
}

func toolSchema(name string, tools []map[string]any) map[string]any {
	for _, tool := range tools {
		toolName := stringValue(tool, "name", "")
		var schema any = firstNonNil(tool["parameters"], tool["input_schema"])
		if fn, ok := tool["function"].(map[string]any); ok {
			if toolName == "" {
				toolName = firstString(fn["name"])
			}
			if schema == nil {
				schema = firstNonNil(fn["parameters"], fn["input_schema"])
			}
		}
		if toolName == name {
			if m, ok := schema.(map[string]any); ok {
				return m
			}
		}
	}
	return nil
}

func schemaProperties(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	if props, ok := schema["properties"].(map[string]any); ok {
		return props
	}
	return nil
}

func schemaTypes(schema map[string]any) map[string]bool {
	out := map[string]bool{}
	switch raw := schema["type"].(type) {
	case string:
		out[raw] = true
	case []any:
		for _, item := range raw {
			if s, ok := item.(string); ok {
				out[s] = true
			}
		}
	}
	if schema["properties"] != nil {
		out["object"] = true
	}
	if schema["items"] != nil {
		out["array"] = true
	}
	for _, key := range []string{"anyOf", "oneOf", "allOf"} {
		if variants, ok := schema[key].([]any); ok {
			for _, variant := range variants {
				if m, ok := variant.(map[string]any); ok {
					for t := range schemaTypes(m) {
						out[t] = true
					}
				}
			}
		}
	}
	return out
}

func missingRequiredArgs(name string, input any, tools []map[string]any) bool {
	m, ok := input.(map[string]any)
	if !ok {
		return false
	}
	required := requiredToolArgs(name, tools)
	for _, key := range required {
		value, ok := m[key]
		if !ok || value == nil {
			return true
		}
		if s, ok := value.(string); ok && strings.TrimSpace(s) == "" && !requiredArgAllowsEmptyString(name, key) {
			return true
		}
	}
	return false
}

func requiredArgAllowsEmptyString(toolName, argName string) bool {
	tool := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(strings.TrimSpace(toolName)), "")
	arg := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(strings.TrimSpace(argName)), "")
	switch tool {
	case "write", "writefile", "createfile":
		switch arg {
		case "content", "text", "body", "data", "value", "contents", "filecontent":
			return true
		}
	}
	return false
}

func requiredToolArgs(name string, tools []map[string]any) []string {
	seen := map[string]bool{}
	out := []string{}
	add := func(keys ...string) {
		for _, key := range keys {
			if key != "" && !seen[key] {
				seen[key] = true
				out = append(out, key)
			}
		}
	}
	schema := toolSchema(name, tools)
	if list, ok := schema["required"].([]any); ok {
		for _, item := range list {
			if s, ok := item.(string); ok {
				add(s)
			}
		}
	}
	switch name {
	case "Read":
		add("file_path")
	case "Write":
		add("file_path", "content")
	case "Edit":
		add("file_path")
	case "Bash", "PowerShell":
		add("command")
	}
	return out
}

func toolAcceptsField(name string, tools []map[string]any, field string) bool {
	props := schemaProperties(toolSchema(name, tools))
	_, ok := props[field]
	return ok
}

func renameFirstPresent(m map[string]any, canonical string, aliases ...string) {
	if m[canonical] != nil {
		return
	}
	for _, alias := range aliases {
		if value, ok := m[alias]; ok {
			delete(m, alias)
			m[canonical] = value
			return
		}
	}
}

func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for key, value := range m {
		out[key] = value
	}
	return out
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	default:
		return strings.TrimSpace(strings.Trim(strings.ReplaceAll(mustJSON(v), "\n", " "), `"`))
	}
}
