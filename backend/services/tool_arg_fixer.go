package services

import "encoding/json"

func CoerceToolArguments(value any) any {
	switch v := value.(type) {
	case nil:
		return map[string]any{}
	case string:
		var decoded any
		if err := json.Unmarshal([]byte(v), &decoded); err == nil {
			return CoerceToolArguments(decoded)
		}
		return map[string]any{"input": v}
	default:
		return v
	}
}
