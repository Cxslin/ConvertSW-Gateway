package services

type CompletionResult struct {
	AnswerText    string
	ReasoningText string
	Events        []UpstreamEvent
}

func MergeCompletionEvent(result CompletionResult, event UpstreamEvent) CompletionResult {
	if event.Content != "" {
		result.AnswerText += event.Content
	}
	if event.ReasoningText != "" {
		result.ReasoningText += event.ReasoningText
	}
	result.Events = append(result.Events, event)
	return result
}
