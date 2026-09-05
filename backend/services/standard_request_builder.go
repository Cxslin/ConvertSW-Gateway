package services

import "qwen2api-go/adapter"

func BuildStandardRequest(body map[string]any, defaultModel, surface string, aliases map[string]string) adapter.StandardRequest {
	return adapter.BuildChatStandardRequest(
		body,
		defaultModel,
		surface,
		func(name string) string { return ResolveModel(name, aliases) },
		func(modelID, defaultModel string) adapter.ModelMode {
			mode := ParseModelMode(modelID, defaultModel)
			return adapter.ModelMode{
				RequestedModel: mode.RequestedModel,
				BaseModel:      mode.BaseModel,
				ChatType:       mode.ChatType,
				ForceThinking:  mode.ForceThinking,
				Mode:           mode.Mode,
			}
		},
	)
}
