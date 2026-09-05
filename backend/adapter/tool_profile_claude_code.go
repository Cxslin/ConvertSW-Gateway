package adapter

func claudeCodeToolProfile() cliToolProfile {
	return cliToolProfile{
		ID:          toolProfileClaudeCode,
		DisplayName: "Claude Code CLI",
		Match: func(names toolNameSet) bool {
			return names.hasAny("Bash", "Read", "Write", "Edit", "MultiEdit", "Glob", "Grep") &&
				names.hasAny("Task", "TaskCreate", "TaskGet", "TaskOutput", "TaskStop", "TaskUpdate", "WebFetch", "WebSearch", "Agent", "AskUserQuestion")
		},
		Priority: map[string]int{
			"read":         0,
			"bash":         1,
			"powershell":   1,
			"glob":         2,
			"grep":         3,
			"write":        4,
			"edit":         5,
			"multiedit":    5,
			"webfetch":     6,
			"websearch":    7,
			"task":         40,
			"agent":        40,
			"taskcreate":   40,
			"todowrite":    80,
			"tasklist":     80,
			"taskupdate":   80,
			"croncreate":   90,
			"schedulewake": 90,
		},
		Rules: []string{
			"Use exact Claude Code action names such as Read, Bash, Write, Edit, MultiEdit, Glob, Grep, WebFetch, WebSearch, and Task/Agent when they are listed.",
			"Do not translate Claude Code tool names to Hermes names such as read_file or terminal unless those exact names are listed in Available action names.",
			"Use Task/Agent only for explicit or clearly beneficial delegation; direct file and shell tools should handle ordinary project work.",
		},
	}
}
