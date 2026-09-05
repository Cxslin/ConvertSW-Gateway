package services

import "unicode/utf8"

func EstimateTokens(text string) int {
	runes := utf8.RuneCountInString(text)
	if runes == 0 {
		return 0
	}
	return (runes + 3) / 4
}

func UsageFromText(prompt, completion string) map[string]any {
	promptTokens := EstimateTokens(prompt)
	completionTokens := EstimateTokens(completion)
	return map[string]any{
		"prompt_tokens":     promptTokens,
		"completion_tokens": completionTokens,
		"total_tokens":      promptTokens + completionTokens,
	}
}
