package toolcall

import (
	"regexp"
	"strings"
)

// ParseTextKVInput accepts simple key=value / key: value argument blocks.
func ParseTextKVInput(text string) map[string]any {
	out := map[string]any{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		sep := strings.Index(line, "=")
		if colon := strings.Index(line, ":"); sep < 0 || colon >= 0 && colon < sep {
			sep = colon
		}
		if sep <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:sep])
		value := strings.Trim(strings.TrimSpace(line[sep+1:]), `"'`)
		if key != "" {
			out[key] = value
		}
	}
	return out
}

func parseTextKVToolCalls(text string, allowed map[string]string, tools []map[string]any) []ParsedToolCall {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	values := map[string][]string{"name": {}, "arguments": {}}
	current := ""
	keyAliases := map[string]string{
		"function.name":      "name",
		"name":               "name",
		"tool":               "name",
		"tool.name":          "name",
		"tool_name":          "name",
		"function.arguments": "arguments",
		"arguments":          "arguments",
		"args":               "arguments",
		"input":              "arguments",
		"tool_input":         "arguments",
		"parameters":         "arguments",
	}
	keyRe := regexp.MustCompile(`(?is)^\s*([A-Za-z_.-][A-Za-z0-9_.-]*)\s*:\s*(.*)$`)
	for _, rawLine := range strings.Split(text, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if match := keyRe.FindStringSubmatch(line); len(match) == 3 {
			key := strings.ToLower(strings.TrimSpace(match[1]))
			if mapped := keyAliases[key]; mapped != "" {
				current = mapped
				values[mapped] = append(values[mapped], strings.TrimSpace(match[2]))
				continue
			}
		}
		if current != "" {
			values[current] = append(values[current], rawLine)
		}
	}
	if len(values["name"]) == 0 {
		return nil
	}
	rawName := strings.Trim(strings.TrimSpace(strings.Split(strings.Join(values["name"], "\n"), "\n")[0]), `"'`)
	name := canonicalToolName(rawName, allowed)
	if name == "" {
		return nil
	}
	args := strings.TrimSpace(strings.Join(values["arguments"], "\n"))
	input := NormalizeToolInput(args)
	call := ParsedToolCall{ID: "call_" + randomID()[:12], Name: name, Input: input}
	call.Input = CoerceToolInput(call.Name, call.Input, tools)
	if missingRequiredArgs(call.Name, call.Input, tools) {
		return nil
	}
	return []ParsedToolCall{call}
}
