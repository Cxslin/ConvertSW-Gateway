package services

import "strings"

func CleanRefusalPrefix(text string) string {
	prefixes := []string{
		"I can't help with that.",
		"I’m sorry, but I can’t help with that.",
		"抱歉，我不能帮助完成该请求。",
	}
	trimmed := strings.TrimSpace(text)
	for _, prefix := range prefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		}
	}
	return text
}
