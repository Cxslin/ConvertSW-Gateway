package api

func ModelRoutes() []RouteSpec {
	return []RouteSpec{
		{Method: "GET", Path: "/v1/models", Auth: "api_key", Description: "OpenAI compatible model list"},
		{Method: "GET", Path: "/v1/models/{model}", Auth: "api_key", Description: "OpenAI compatible model detail"},
	}
}

func ModelDetail(id, baseModel, ownedBy string) map[string]any {
	if ownedBy == "" {
		ownedBy = "qwen"
	}
	return map[string]any{"id": id, "object": "model", "owned_by": ownedBy, "base_model": baseModel}
}
