package toolcall

import (
	"encoding/json"
	"regexp"
	"strings"
)

func canonicalToolName(name string, allowed map[string]string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if exact, ok := allowed[strings.ToLower(name)]; ok {
		return exact
	}
	if alias := qwenToolAlias(name); alias != "" {
		if exact, ok := allowed[strings.ToLower(alias)]; ok {
			return exact
		}
	}
	key := toolAliasKey(name)
	for allowedKey, canonical := range allowed {
		if toolAliasKey(allowedKey) == key || toolAliasKey(canonical) == key {
			return canonical
		}
		if alias := qwenToolAlias(canonical); alias != "" && toolAliasKey(alias) == key {
			return canonical
		}
	}
	return ""
}

func toolAliasKey(value string) string {
	return regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "")
}

func qwenToolAlias(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	explicit := map[string]string{
		"fs_open_file":   "Read",
		"fs_put_file":    "Write",
		"fs_patch_file":  "Edit",
		"shell_run":      "Bash",
		"text_search":    "Grep",
		"path_find":      "Glob",
		"notebook_patch": "NotebookEdit",
		"http_get_url":   "WebFetch",
		"web_query":      "WebSearch",
	}
	lowered := strings.ToLower(trimmed)
	if mapped, ok := explicit[lowered]; ok {
		return mapped
	}
	trimmedKey := toolAliasKey(trimmed)
	for alias, mapped := range explicit {
		if toolAliasKey(alias) == trimmedKey {
			return mapped
		}
	}
	if strings.HasPrefix(trimmed, "u_") && len(trimmed) > 2 {
		return strings.TrimPrefix(trimmed, "u_")
	}
	return ""
}

func parseToolInput(text string) any {
	if text == "" {
		return map[string]any{}
	}
	var value any
	if err := json.Unmarshal([]byte(text), &value); err == nil {
		return NormalizeToolInput(value)
	}
	re := regexp.MustCompile(`(?is)<([A-Za-z_][A-Za-z0-9_\-]*)>(.*?)</\1>`)
	params := map[string]any{}
	for _, match := range re.FindAllStringSubmatch(text, -1) {
		if len(match) == 3 {
			params[match[1]] = strings.TrimSpace(match[2])
		}
	}
	if len(params) > 0 {
		return params
	}
	if kv := ParseTextKVInput(text); len(kv) > 0 {
		return kv
	}
	return map[string]any{"input": text}
}

// NormalizeToolInput decodes nested JSON strings and supplies empty argument maps.
func NormalizeToolInput(value any) any {
	switch v := value.(type) {
	case nil:
		return map[string]any{}
	case string:
		if strings.TrimSpace(v) == "" {
			return map[string]any{}
		}
		var decoded any
		if err := json.Unmarshal([]byte(v), &decoded); err == nil {
			return NormalizeToolInput(decoded)
		}
		if kv := ParseTextKVInput(v); len(kv) > 0 {
			return kv
		}
		return v
	default:
		return v
	}
}
