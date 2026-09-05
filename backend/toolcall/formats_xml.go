package toolcall

import (
	"encoding/json"
	"regexp"
	"strings"
)

func parseXMLToolCalls(text string, allowed map[string]string) []ParsedToolCall {
	calls := []ParsedToolCall{}
	for _, match := range regexp.MustCompile(`(?is)<tool_call\b[^>]*>\s*(.*?)\s*</tool_call\s*>`).FindAllStringSubmatch(text, -1) {
		if len(match) < 2 {
			continue
		}
		body := strings.TrimSpace(match[1])
		var payload any
		if err := json.Unmarshal([]byte(body), &payload); err != nil {
			payload = parseToolInput(body)
		}
		for _, call := range parseJSONToolCalls(payload, allowed) {
			calls = append(calls, call)
		}
	}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?is)<tool_use\b[^>]*\bname=["']([^"']+)["'][^>]*>(.*?)</tool_use>`),
		regexp.MustCompile(`(?is)<tool_call\b[^>]*\bname=["']([^"']+)["'][^>]*>(.*?)</tool_call>`),
		regexp.MustCompile(`(?is)<function\b[^>]*\bname=["']([^"']+)["'][^>]*>(.*?)</function>`),
		regexp.MustCompile(`(?is)<invoke\b[^>]*\bname=["']([^"']+)["'][^>]*>(.*?)</invoke>`),
	}
	for _, re := range patterns {
		for _, match := range re.FindAllStringSubmatch(text, -1) {
			if len(match) < 3 {
				continue
			}
			name := canonicalToolName(match[1], allowed)
			if name == "" {
				continue
			}
			calls = append(calls, ParsedToolCall{
				ID:    "call_" + randomID()[:12],
				Name:  name,
				Input: parseToolInput(strings.TrimSpace(match[2])),
			})
		}
	}
	return calls
}
