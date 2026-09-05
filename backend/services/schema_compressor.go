package services

func CompressSchema(schema map[string]any, maxProperties int) map[string]any {
	if maxProperties <= 0 {
		return schema
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok || len(props) <= maxProperties {
		return schema
	}
	nextProps := map[string]any{}
	count := 0
	for key, value := range props {
		if count >= maxProperties {
			break
		}
		nextProps[key] = value
		count++
	}
	out := map[string]any{}
	for key, value := range schema {
		out[key] = value
	}
	out["properties"] = nextProps
	return out
}
