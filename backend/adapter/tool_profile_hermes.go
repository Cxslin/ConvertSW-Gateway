package adapter

func hermesToolProfile() cliToolProfile {
	return cliToolProfile{
		ID:          toolProfileHermes,
		DisplayName: "Hermes CLI",
		Match: func(names toolNameSet) bool {
			return names.hasAny("skills_list", "skill_view") ||
				names.hasAll("read_file", "terminal", "write_file") ||
				(names.hasAny("delegate_task") && names.hasAny("terminal", "read_file"))
		},
		Priority: map[string]int{
			"readfile":     0,
			"terminal":     1,
			"searchfiles":  2,
			"listfiles":    3,
			"writefile":    4,
			"patch":        5,
			"skillslist":   30,
			"skillview":    31,
			"delegatetask": 40,
			"process":      70,
		},
		Rules: []string{
			"Hermes tool names differ from Claude Code/OpenCode/OpenClaw. Use exact Hermes names from Available action names, not cross-CLI names.",
			"Cross-CLI capability mapping: Read/read -> read_file; Bash/PowerShell/shell -> terminal; Write/write -> write_file; Glob/Grep/Search -> search_files or the exact search/list tool that is available; Agent/Task/subagent -> delegate_task only if it is listed; Skill/skills -> skills_list then skill_view.",
			"If the user asks to test or cover a tool name from another CLI, call the closest Hermes capability tool when appropriate, or record unavailable from actual evidence. Never output fake tool-availability prose for listed tools.",
			"For skills, call skills_list before skill_view. Use skill_view only with a concrete skill identifier returned by skills_list or explicitly supplied by the user.",
			"skill_view is for inspecting one concrete skill only. For skills discovery use skills_list first. For delegation bookkeeping, final verification, or unavailable-tool records, use terminal or write_file to record the status.",
			"For delegate_task, use a bounded prompt with a clear deliverable and continue from its actual result. Prefer direct tools when direct read/shell/search/write/edit can complete the current step.",
		},
	}
}
