package api

import "strings"

func VideoRoutes() []RouteSpec {
	return []RouteSpec{
		{Method: "POST", Path: "/v1/videos/generations", Auth: "api_key", Description: "Qwen video generation compatibility"},
	}
}

func NormalizeVideoSize(size string) string {
	switch strings.TrimSpace(size) {
	case "16:9", "9:16", "1:1", "4:3", "3:4":
		return size
	default:
		return "16:9"
	}
}
