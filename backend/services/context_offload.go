package services

func ShouldOffloadContext(chars, inlineLimit, forceFileLimit int) bool {
	if forceFileLimit > 0 && chars >= forceFileLimit {
		return true
	}
	return inlineLimit > 0 && chars > inlineLimit
}

func ContextSnippet(text string, limit int) string {
	runes := []rune(text)
	if limit <= 0 || len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}
