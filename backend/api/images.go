package api

import "strings"

func ImageRoutes() []RouteSpec {
	return []RouteSpec{
		{Method: "POST", Path: "/v1/images/generations", Auth: "api_key", Description: "OpenAI image generation compatibility"},
	}
}

func NormalizeImageSize(size string) (ratio string, width int, height int) {
	switch strings.TrimSpace(size) {
	case "1024x1792", "9:16":
		return "9:16", 1024, 1792
	case "1792x1024", "16:9":
		return "16:9", 1792, 1024
	default:
		return "1:1", 1024, 1024
	}
}
