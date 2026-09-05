package services

import (
	"strings"
	"time"
)

const OpenAIModelCreatedEpoch = int64(1700000000)

func BuildModelEntry(modelID, baseModel string, capabilities map[string]bool, mode, displayName, family string, created int64, ownedBy string) map[string]any {
	if baseModel == "" {
		baseModel = modelID
	}
	if displayName == "" {
		displayName = modelID
	}
	if family == "" {
		family = baseModel
	}
	if created == 0 {
		created = OpenAIModelCreatedEpoch
	}
	if ownedBy == "" {
		ownedBy = "qwen"
	}
	return map[string]any{
		"id":           modelID,
		"object":       "model",
		"created":      created,
		"owned_by":     ownedBy,
		"capabilities": capabilities,
		"base_model":   baseModel,
		"mode":         mode,
		"display_name": displayName,
		"family":       family,
	}
}

func BuildFallbackModelList(modelAliases map[string]string) map[string]any {
	now := time.Now().Unix()
	data := []map[string]any{}
	for modelID, resolved := range modelAliases {
		family := resolved
		if idx := strings.Index(family, "-"); idx >= 0 {
			family = family[:idx]
		}
		data = append(data, BuildModelEntry(modelID, resolved, map[string]bool{}, "chat", modelID, family, now, "qwen2api"))
	}
	return map[string]any{"object": "list", "data": data}
}

func BuildOpenAIModelList(upstream []map[string]any) map[string]any {
	seen := map[string]struct{}{}
	data := []map[string]any{}
	add := func(entry map[string]any) {
		id := anyString(entry["id"], "")
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		data = append(data, entry)
	}
	variants := []struct {
		capability string
		suffix     string
		mode       string
		caps       map[string]bool
	}{
		{"thinking", "-thinking", "thinking", map[string]bool{"thinking": true}},
		{"search", "-search", "search", map[string]bool{"search": true}},
		{"deep_research", "-deep-research", "deep_research", map[string]bool{"deep_research": true, "search": true}},
		{"image_gen", "-image", "image", map[string]bool{"image_gen": true}},
		{"video_gen", "-video", "video", map[string]bool{"video_gen": true}},
		{"web_dev", "-webdev", "webdev", map[string]bool{"web_dev": true}},
		{"slides", "-slides", "slides", map[string]bool{"slides": true}},
	}
	for _, raw := range upstream {
		modelID := firstString(raw["id"], raw["model"], raw["name"])
		if modelID == "" {
			continue
		}
		display := firstString(raw["display_name"], raw["displayName"], raw["name"])
		if display == "" {
			display = modelID
		}
		family := DeriveFamily(modelID, raw)
		caps := ExtractModelCapabilities(raw)
		entryMode := "chat"
		entryBaseModel := modelID
		if baseModel, mode, detectedCaps, ok := detectVariantModel(modelID); ok {
			entryMode = mode
			entryBaseModel = baseModel
			for key, value := range detectedCaps {
				if value {
					caps[key] = true
				}
			}
		}
		add(BuildModelEntry(modelID, entryBaseModel, caps, entryMode, display, family, OpenAIModelCreatedEpoch, "qwen"))
		if entryMode != "chat" {
			continue
		}
		for _, variant := range variants {
			if caps[variant.capability] || variant.capability == "search" {
				add(BuildModelEntry(modelID+variant.suffix, modelID, variant.caps, variant.mode, display+" "+variant.mode, family, OpenAIModelCreatedEpoch, "qwen"))
			}
		}
	}
	return map[string]any{"object": "list", "data": data}
}

func detectVariantModel(modelID string) (string, string, map[string]bool, bool) {
	trimmed := strings.TrimSpace(modelID)
	lowered := strings.ToLower(trimmed)
	variants := []struct {
		suffix string
		mode   string
		caps   map[string]bool
	}{
		{"-thinking", "thinking", map[string]bool{"thinking": true}},
		{"-search", "search", map[string]bool{"search": true}},
		{"-deep-research", "deep_research", map[string]bool{"deep_research": true, "search": true}},
		{"-deep_research", "deep_research", map[string]bool{"deep_research": true, "search": true}},
		{"-image", "image", map[string]bool{"image_gen": true}},
		{"-video", "video", map[string]bool{"video_gen": true}},
		{"-web-dev", "webdev", map[string]bool{"web_dev": true}},
		{"-webdev", "webdev", map[string]bool{"web_dev": true}},
		{"-slides", "slides", map[string]bool{"slides": true}},
		{"-t2i", "image", map[string]bool{"image_gen": true}},
		{"-t2v", "video", map[string]bool{"video_gen": true}},
	}
	for _, variant := range variants {
		if strings.HasSuffix(lowered, variant.suffix) && len(trimmed) > len(variant.suffix) {
			return strings.TrimSpace(trimmed[:len(trimmed)-len(variant.suffix)]), variant.mode, copyCaps(variant.caps), true
		}
	}
	return "", "", nil, false
}

func copyCaps(src map[string]bool) map[string]bool {
	dst := make(map[string]bool, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func ExtractModelCapabilities(item map[string]any) map[string]bool {
	caps := map[string]bool{
		"thinking":      false,
		"search":        false,
		"vision":        false,
		"deep_research": false,
		"image_gen":     false,
		"video_gen":     false,
		"web_dev":       false,
		"slides":        false,
	}
	meta := map[string]any{}
	if info, ok := item["info"].(map[string]any); ok {
		if m, ok := info["meta"].(map[string]any); ok {
			meta = m
		}
	}
	if m, ok := item["meta"].(map[string]any); ok {
		for k, v := range m {
			meta[k] = v
		}
	}
	if rawCaps, ok := meta["capabilities"].(map[string]any); ok {
		for k, v := range rawCaps {
			if b, ok := v.(bool); ok {
				caps[k] = b
			}
		}
	}
	applyChatType := func(chatType string) {
		switch chatType {
		case "deep_research":
			caps["deep_research"] = true
		case "t2i", "image_gen":
			caps["image_gen"] = true
		case "t2v":
			caps["video_gen"] = true
		case "web_dev":
			caps["web_dev"] = true
		case "slides":
			caps["slides"] = true
		}
	}
	switch v := meta["chat_type"].(type) {
	case string:
		applyChatType(v)
	case []any:
		for _, item := range v {
			applyChatType(anyString(item, ""))
		}
	}
	return caps
}

func DeriveFamily(modelID string, item map[string]any) string {
	if family := anyString(item["family"], ""); family != "" {
		return family
	}
	if strings.HasPrefix(modelID, "qwen3.") {
		parts := strings.Split(strings.SplitN(modelID, "-", 2)[0], ".")
		if len(parts) >= 2 {
			return parts[0] + "." + parts[1]
		}
	}
	if idx := strings.Index(modelID, "-"); idx > 0 {
		return modelID[:idx]
	}
	return modelID
}

func anyString(v any, fallback string) string {
	if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
		return s
	}
	return fallback
}

func firstString(values ...any) string {
	for _, value := range values {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
