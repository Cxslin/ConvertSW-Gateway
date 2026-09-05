package services

type ClientProfile struct {
	Name      string
	UserAgent string
	Headers   map[string]string
}

func DefaultClientProfile() ClientProfile {
	return ClientProfile{
		Name:      "qwen-web",
		UserAgent: "Mozilla/5.0 qwen2api-go",
		Headers: map[string]string{
			"Origin":  "https://chat.qwen.ai",
			"Referer": "https://chat.qwen.ai/",
		},
	}
}
