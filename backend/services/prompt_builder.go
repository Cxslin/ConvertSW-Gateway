package services

import (
	"strings"

	"qwen2api-go/adapter"
)

const (
	ClaudeCodeOpenAIProfile = "claude_code_openai"
	OpenClawOpenAIProfile   = "openclaw_openai"
)

type PromptMessage struct {
	Role    string
	Content string
}

func BuildPrompt(messages []PromptMessage, attachments string, toolInstructions string) string {
	parts := []string{}
	for _, msg := range messages {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			role = "user"
		}
		parts = append(parts, "["+strings.Title(role)+"]\n"+content)
	}
	if strings.TrimSpace(attachments) != "" {
		parts = append(parts, "[Attachments]\n"+attachments)
	}
	if strings.TrimSpace(toolInstructions) != "" {
		parts = append(parts, toolInstructions)
	}
	return strings.Join(parts, "\n\n")
}

func MessagesToPrompt(body map[string]any) (string, []map[string]any) {
	return adapter.MessagesToPrompt(body)
}

func BuildToolInstructions(tools []map[string]any) string {
	return adapter.BuildToolInstructions(tools)
}

func NormalizeTools(raw any) []map[string]any {
	return adapter.NormalizeTools(raw)
}

func ExtractContentText(content any) string {
	return adapter.ExtractContentText(content)
}

func BuildClaudeCodePrompt(messages []any, tools []map[string]any) string {
	body := map[string]any{
		"messages": messages,
		"tools":    tools,
	}
	prompt, _ := adapter.MessagesToPrompt(body)
	return prompt
}

func BuildWorkspaceFinalReminder(workspaceRoot string) string {
	return strings.Join([]string{
		"[WORKSPACE REMINDER]",
		"When the user says current directory, use the client tool runtime's current working directory.",
		"Do not invent or force an absolute workspace path unless the user explicitly provided it or a recent tool result proved it.",
	}, "\n")
}
