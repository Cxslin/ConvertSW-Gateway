package services

import "strings"

func ToolFewShotPrompt(toolNames []string) string {
	if len(toolNames) == 0 {
		return ""
	}
	return "When a tool is needed, emit a single tool_call with one of these names: " + strings.Join(toolNames, ", ")
}
