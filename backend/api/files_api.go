package api

import (
	"path/filepath"
	"strings"
)

func FileRoutes() []RouteSpec {
	return []RouteSpec{
		{Method: "POST", Path: "/v1/files", Auth: "api_key", Description: "upload file for context"},
		{Method: "DELETE", Path: "/v1/files/{file_id}", Auth: "api_key", Description: "delete uploaded file"},
	}
}

func SafeUploadName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "upload.bin"
	}
	return name
}
