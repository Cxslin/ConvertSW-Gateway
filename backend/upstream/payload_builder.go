package upstream

import (
	"crypto/rand"
	"encoding/hex"
)

// NormalizeChatType maps API-facing aliases onto Qwen upstream chat types.
func NormalizeChatType(chatType string) string {
	if chatType == "image_gen" || chatType == "t2i" {
		return "t2i"
	}
	return chatType
}

// BuildChatPayload builds the DeepSeek /api/v0/chat/completion request body.
func BuildChatPayload(chatID, model, content string, hasCustomTools bool, files []map[string]any, chatType string, imageOptions map[string]any, thinkingEnabled *bool, enableSearch bool) map[string]any {
	thinking := false
	if model == "deepseek-reasoner" || model == "deepseek-r1" || (thinkingEnabled != nil && *thinkingEnabled) {
		thinking = true
	}
	modelType := "default"
	if model == "deepseek-vision" || len(files) > 0 {
		modelType = "vision"
	}
	refFileIDs := []string{}
	for _, f := range files {
		if id, ok := f["id"].(string); ok && id != "" {
			refFileIDs = append(refFileIDs, id)
		} else if id, ok := f["file_id"].(string); ok && id != "" {
			refFileIDs = append(refFileIDs, id)
		}
	}

	payload := map[string]any{
		"chat_session_id":   chatID,
		"parent_message_id": nil,
		"prompt":            content,
		"ref_file_ids":      refFileIDs,
		"thinking_enabled":  thinking,
		"search_enabled":    enableSearch,
		"audio_id":          nil,
		"preempt":           false,
		"model_type":        modelType,
		"action":            nil,
		"model":             model,
		"chat_id":           chatID,
	}
	return payload
}

func imageRatio(options map[string]any) string {
	for _, key := range []string{"ratio", "aspect_ratio", "aspectRatio"} {
		if v, ok := options[key].(string); ok && v != "" {
			return v
		}
	}
	return "1:1"
}

func randomID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(buf)
}
