package adapter

func openClawToolProfile() cliToolProfile {
	return cliToolProfile{
		ID:          toolProfileOpenClaw,
		DisplayName: "OpenClaw CLI",
		Match: func(names toolNameSet) bool {
			if names.hasAny("skills_list", "skill_view", "delegate_task") {
				return false
			}
			if names.hasAny("openclaw", "agents_list", "sessions_spawn", "sessions_send", "sessions_history", "sessions_list", "subagents") {
				return true
			}
			return names.hasAny("exec", "process") &&
				names.hasAny("read", "write", "edit", "ls", "grep", "find", "web_fetch", "web_search") &&
				!names.hasAny("bash", "Bash", "terminal", "todo_write", "todowrite")
		},
		Priority: map[string]int{
			"read":            0,
			"exec":            1,
			"ls":              2,
			"grep":            3,
			"find":            3,
			"write":           4,
			"edit":            5,
			"applypatch":      5,
			"webfetch":        6,
			"websearch":       7,
			"agentslist":      35,
			"sessionsspawn":   40,
			"sessionssend":    41,
			"subagents":       42,
			"sessionslist":    45,
			"sessionshistory": 45,
			"process":         70,
			"updateplan":      80,
		},
		Rules: []string{
			"OpenClaw tool names differ from Hermes, Claude Code, and OpenCode. Use exact OpenClaw names from Available action names only.",
			"Treat OpenClaw read/write/edit/exec as available client-executable QNML actions whenever they are listed. Do not answer that any listed OpenClaw action does not exist.",
			"Cross-CLI capability mapping: Read/read_file -> read; Bash/PowerShell/terminal/shell/run_command -> exec; Write/write_file -> write; Edit/MultiEdit/edit_file -> edit or apply_patch when listed; Glob/Grep/Search -> grep/find/ls only when those exact names are listed, otherwise use exec for shell list/search commands; WebFetch/WebSearch -> web_fetch/web_search when listed; Agent/Task/subagent -> sessions_spawn/subagents only when listed.",
			"Use process only to inspect or control background exec sessions that already exist; use exec for new shell commands.",
			"OpenClaw exposes skills primarily through its CLI/config/runtime, not necessarily as model tools. If no exact skills tool is listed, use read/exec/write to record skills as unavailable or externally inspected instead of calling Hermes skills_list/skill_view or Claude Skill.",
			"Never substitute Hermes names such as terminal/read_file/write_file/delegate_task or Claude Code names such as Bash/Read/Write/Task unless those exact names are listed in Available action names.",
		},
	}
}
