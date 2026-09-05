package core

import "strings"

func FilterSensitiveFields(fields map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range fields {
		lowered := strings.ToLower(k)
		if strings.Contains(lowered, "token") || strings.Contains(lowered, "cookie") || strings.Contains(lowered, "password") || strings.Contains(lowered, "authorization") {
			if s, ok := v.(string); ok {
				out[k] = RedactToken(s)
			} else {
				out[k] = "***"
			}
			continue
		}
		out[k] = v
	}
	return out
}
