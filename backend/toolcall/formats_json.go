package toolcall

import (
	"encoding/json"
	"regexp"
	"strings"
)

func parseJSONToolCalls(value any, allowed map[string]string) []ParsedToolCall {
	calls := []ParsedToolCall{}
	switch v := value.(type) {
	case map[string]any:
		if rawList, ok := v["tool_calls"].([]any); ok {
			for _, raw := range rawList {
				calls = append(calls, parseJSONToolCalls(raw, allowed)...)
			}
		}
		if rawList, ok := v["tools"].([]any); ok {
			for _, raw := range rawList {
				calls = append(calls, parseJSONToolCalls(raw, allowed)...)
			}
		}
		name := firstString(v["name"], v["tool"], v["tool_name"], v["function_name"])
		input := firstNonNil(v["input"], v["arguments"], v["args"], v["parameters"])
		if fn, ok := v["function"].(map[string]any); ok {
			if name == "" {
				name = firstString(fn["name"])
			}
			if input == nil {
				input = firstNonNil(fn["arguments"], fn["input"], fn["parameters"])
			}
		}
		if name = canonicalToolName(name, allowed); name != "" {
			calls = append(calls, ParsedToolCall{
				ID:    firstNonEmpty(firstString(v["id"], v["call_id"]), "call_"+randomID()[:12]),
				Name:  name,
				Input: NormalizeToolInput(input),
			})
		}
	case []any:
		for _, item := range v {
			calls = append(calls, parseJSONToolCalls(item, allowed)...)
		}
	}
	return calls
}

// ForEachJSONFragment visits standalone JSON objects or arrays embedded in text.
func ForEachJSONFragment(text string, visit func(any)) {
	decoder := json.NewDecoder(strings.NewReader(stripJSONFence(text)))
	for {
		var value any
		if err := decoder.Decode(&value); err == nil {
			visit(value)
			continue
		}
		break
	}
	normalized := stripJSONFence(text)
	for _, candidate := range []string{normalized, repairLooseJSON(normalized), servicesRecoverJSONLike(normalized)} {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		var value any
		if err := json.Unmarshal([]byte(candidate), &value); err == nil {
			visit(value)
		}
	}
	for start := 0; start < len(normalized); start++ {
		if normalized[start] != '{' && normalized[start] != '[' {
			continue
		}
		for end := len(normalized); end > start; end-- {
			var value any
			fragment := normalized[start:end]
			if err := json.Unmarshal([]byte(fragment), &value); err == nil {
				visit(value)
				break
			}
			if repaired := repairLooseJSON(fragment); repaired != fragment {
				if err := json.Unmarshal([]byte(repaired), &value); err == nil {
					visit(value)
					break
				}
			}
			if recovered := servicesRecoverJSONLike(fragment); recovered != fragment {
				if err := json.Unmarshal([]byte(recovered), &value); err == nil {
					visit(value)
					break
				}
			}
		}
	}
}

func stripJSONFence(text string) string {
	stripped := strings.TrimSpace(text)
	if !strings.HasPrefix(stripped, "```") {
		return stripped
	}
	stripped = strings.TrimPrefix(stripped, "```json")
	stripped = strings.TrimPrefix(stripped, "```")
	stripped = strings.TrimSpace(stripped)
	if strings.HasSuffix(stripped, "```") {
		stripped = strings.TrimSpace(strings.TrimSuffix(stripped, "```"))
	}
	return stripped
}

func repairLooseJSON(text string) string {
	repaired := strings.TrimSpace(text)
	if repaired == "" {
		return repaired
	}
	replacements := []struct {
		re   *regexp.Regexp
		repl string
	}{
		{regexp.MustCompile(`(?is)"name="\s*`), `"name": "`},
		{regexp.MustCompile(`(?is)"name=([^",}\s]+)"`), `"name": "$1"`},
		{regexp.MustCompile(`(?is)"name=([^",}\s]+)`), `"name": "$1"`},
		{regexp.MustCompile(`(?is)"name\s*=\s*"`), `"name": "`},
		{regexp.MustCompile(`(?is)"(name|input|arguments|args|parameters|tool|tool_name|function_name)"\s*=\s*`), `"$1": `},
		{regexp.MustCompile(`(?is)([{,]\s*)(name|input|arguments|args|parameters|tool|tool_name|function_name)\s*:`), `$1"$2":`},
	}
	for _, replacement := range replacements {
		repaired = replacement.re.ReplaceAllString(repaired, replacement.repl)
	}
	return repaired
}

func servicesRecoverJSONLike(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}
	openBraces := strings.Count(text, "{") - strings.Count(text, "}")
	openBrackets := strings.Count(text, "[") - strings.Count(text, "]")
	for openBrackets > 0 {
		text += "]"
		openBrackets--
	}
	for openBraces > 0 {
		text += "}"
		openBraces--
	}
	return text
}
