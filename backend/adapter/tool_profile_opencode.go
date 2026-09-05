package adapter

func openCodeToolProfile() cliToolProfile {
	return cliToolProfile{
		ID:          toolProfileOpenCode,
		DisplayName: "OpenCode CLI",
		Match: func(names toolNameSet) bool {
			return names.hasAny("bash", "read", "write", "edit") &&
				names.hasAny("todowrite", "todo_write", "glob", "grep") &&
				!names.hasAny("skills_list", "skill_view")
		},
		Priority: map[string]int{
			"read":      0,
			"bash":      1,
			"glob":      2,
			"grep":      3,
			"write":     4,
			"edit":      5,
			"todowrite": 80,
		},
		Rules: []string{
			"Use exact OpenCode action names as listed. Do not borrow Claude Code capitalized names or Hermes underscore names unless those exact names are listed.",
			"Treat todo/planning tools as planning state only; use file/shell/search tools for executable work and verification.",
		},
	}
}
