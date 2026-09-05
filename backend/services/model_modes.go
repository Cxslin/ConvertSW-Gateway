package services

import "strings"

type ModelMode struct {
	RequestedModel string
	BaseModel      string
	ChatType       string
	ForceThinking  bool
	Mode           string
}

func DefaultModelAliases() map[string]string {
	return map[string]string{
		"gpt-4o": "qwen3.6-plus", "gpt-4o-mini": "qwen3.5-flash", "gpt-4": "qwen3.6-plus",
		"gpt-3.5-turbo": "qwen3.5-flash", "gpt-5": "qwen3.6-plus",
		"claude-sonnet-4-5": "qwen3.6-plus", "claude-3-haiku": "qwen3.5-flash",
		"gemini-2.5-pro": "qwen3.6-plus", "gemini-2.5-flash": "qwen3.5-flash",
		"qwen": "qwen3.6-plus", "qwen-plus": "qwen3.6-plus", "qwen-turbo": "qwen3.5-flash",
	}
}

func ResolveModel(name string, aliases map[string]string) string {
	trimmed := strings.TrimSpace(name)
	if v, ok := aliases[trimmed]; ok {
		return v
	}
	if v, ok := aliases[strings.ToLower(trimmed)]; ok {
		return v
	}
	for _, suffix := range modelModeSuffixes() {
		lowered := strings.ToLower(trimmed)
		if strings.HasSuffix(lowered, suffix) && len(trimmed) > len(suffix) {
			base := strings.TrimSpace(trimmed[:len(trimmed)-len(suffix)])
			if mapped := ResolveModel(base, aliases); mapped != base && mapped != "" {
				return mapped + trimmed[len(trimmed)-len(suffix):]
			}
		}
	}
	return trimmed
}

func ParseModelMode(modelID, defaultModel string) ModelMode {
	requested := strings.TrimSpace(modelID)
	if requested == "" {
		requested = defaultModel
	}
	lowered := strings.ToLower(requested)
	suffixes := []struct {
		suffix        string
		chatType      string
		forceThinking bool
		mode          string
	}{
		{"-deep-research", "deep_research", false, "deep_research"},
		{"-deep_research", "deep_research", false, "deep_research"},
		{"-web-dev", "web_dev", false, "webdev"},
		{"-thinking", "t2t", true, "thinking"},
		{"-search", "t2t", false, "search"},
		{"-webdev", "web_dev", false, "webdev"},
		{"-image", "t2i", false, "image"},
		{"-video", "t2v", false, "video"},
		{"-slides", "slides", false, "slides"},
		{"-t2i", "t2i", false, "image"},
		{"-t2v", "t2v", false, "video"},
	}
	for _, s := range suffixes {
		if strings.HasSuffix(lowered, s.suffix) {
			return ModelMode{
				RequestedModel: requested,
				BaseModel:      strings.TrimSpace(requested[:len(requested)-len(s.suffix)]),
				ChatType:       s.chatType,
				ForceThinking:  s.forceThinking,
				Mode:           s.mode,
			}
		}
	}
	return ModelMode{RequestedModel: requested, BaseModel: requested, ChatType: "t2t", Mode: "chat"}
}

func modelModeSuffixes() []string {
	return []string{"-deep-research", "-deep_research", "-web-dev", "-thinking", "-search", "-webdev", "-image", "-video", "-slides", "-t2i", "-t2v"}
}
