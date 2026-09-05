package services

import "strings"

func IsNewTopic(prompt, previousPrompt string) bool {
	prompt = strings.TrimSpace(strings.ToLower(prompt))
	previousPrompt = strings.TrimSpace(strings.ToLower(previousPrompt))
	if prompt == "" || previousPrompt == "" {
		return true
	}
	if strings.Contains(prompt, previousPrompt) || strings.Contains(previousPrompt, prompt) {
		return false
	}
	return true
}
