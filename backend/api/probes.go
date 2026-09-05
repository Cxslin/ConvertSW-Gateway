package api

type RouteSpec struct {
	Method      string
	Path        string
	Auth        string
	Description string
}

func ProbeRoutes() []RouteSpec {
	return []RouteSpec{
		{Method: "GET", Path: "/healthz", Auth: "none", Description: "liveness probe"},
		{Method: "GET", Path: "/readyz", Auth: "none", Description: "account readiness probe"},
	}
}

func HealthPayload(version string) map[string]any {
	return map[string]any{"status": "qwen2API Enterprise Gateway is running", "docs": "/docs", "version": version}
}

func ReadyPayload(ready bool, detail map[string]any) map[string]any {
	return map[string]any{"ready": ready, "accounts": detail}
}
