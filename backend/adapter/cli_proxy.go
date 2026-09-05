package adapter

import "strings"

type CLIRequest struct {
	Model  string
	Prompt string
	Stream bool
}

func ParseCLIArgs(args []string) CLIRequest {
	req := CLIRequest{Model: "gpt-3.5-turbo"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--model", "-m":
			if i+1 < len(args) {
				req.Model = args[i+1]
				i++
			}
		case "--stream":
			req.Stream = true
		default:
			if strings.TrimSpace(args[i]) != "" {
				if req.Prompt != "" {
					req.Prompt += " "
				}
				req.Prompt += args[i]
			}
		}
	}
	return req
}

func (r CLIRequest) ChatBody() map[string]any {
	return map[string]any{
		"model":  r.Model,
		"stream": r.Stream,
		"messages": []map[string]any{{
			"role":    "user",
			"content": r.Prompt,
		}},
	}
}
