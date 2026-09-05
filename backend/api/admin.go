package api

func AdminRoutes() []RouteSpec {
	return []RouteSpec{
		{Method: "GET", Path: "/admin/status", Auth: "admin", Description: "runtime status and account pool summary"},
		{Method: "GET", Path: "/admin/accounts", Auth: "admin", Description: "list qwen accounts"},
		{Method: "POST", Path: "/admin/accounts", Auth: "admin", Description: "import or create qwen account"},
		{Method: "DELETE", Path: "/admin/accounts", Auth: "admin", Description: "delete qwen account"},
		{Method: "GET", Path: "/admin/keys", Auth: "admin", Description: "list API keys"},
		{Method: "POST", Path: "/admin/keys", Auth: "admin", Description: "create API key"},
		{Method: "DELETE", Path: "/admin/keys", Auth: "admin", Description: "delete API key"},
	}
}

func AdminStatusPayload(version string, accounts map[string]any, settings map[string]any) map[string]any {
	return map[string]any{"version": version, "accounts": accounts, "settings": settings}
}
