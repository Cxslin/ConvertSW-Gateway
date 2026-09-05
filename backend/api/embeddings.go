package api

func EmbeddingRoutes() []RouteSpec {
	return []RouteSpec{
		{Method: "POST", Path: "/v1/embeddings", Auth: "api_key", Description: "OpenAI embeddings compatibility"},
	}
}

func EmbeddingPayload(model string, vectors [][]float64) map[string]any {
	data := []map[string]any{}
	for i, vector := range vectors {
		data = append(data, map[string]any{"object": "embedding", "index": i, "embedding": vector})
	}
	return map[string]any{"object": "list", "model": model, "data": data}
}
