package api

import "strings"

func GeminiRoutes() []RouteSpec {
	return []RouteSpec{
		{Method: "POST", Path: "/v1beta/models/{model}:generateContent", Auth: "api_key", Description: "Gemini generateContent compatibility"},
		{Method: "POST", Path: "/v1beta/models/{model}:streamGenerateContent", Auth: "api_key", Description: "Gemini streamGenerateContent compatibility"},
	}
}

func GeminiModelFromPath(path string) string {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.Contains(part, ":") {
			return strings.SplitN(part, ":", 2)[0]
		}
	}
	return ""
}
