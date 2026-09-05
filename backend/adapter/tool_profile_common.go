package adapter

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	toolProfileGeneric    = "generic"
	toolProfileHermes     = "hermes"
	toolProfileClaudeCode = "claude_code"
	toolProfileOpenCode   = "opencode"
	toolProfileOpenClaw   = "openclaw"
)

type cliToolProfile struct {
	ID          string
	DisplayName string
	Rules       []string
	Priority    map[string]int
	Match       func(toolNameSet) bool
}

type toolNameSet map[string]bool

type toolCapability struct {
	Key     string
	Label   string
	Aliases []string
}

var toolProfileNameRe = regexp.MustCompile(`[^a-z0-9]+`)

var commonToolCapabilities = []toolCapability{
	{
		Key:     "read",
		Label:   "file read",
		Aliases: []string{"read", "readfile", "readfiletool", "read_file", "openfile", "open_file", "fsopenfile", "fs_open_file", "viewfile"},
	},
	{
		Key:     "write",
		Label:   "file write",
		Aliases: []string{"write", "writefile", "write_file", "createfile", "create_file", "savefile", "fsputfile", "fs_put_file"},
	},
	{
		Key:     "edit",
		Label:   "file edit/patch",
		Aliases: []string{"edit", "editfile", "edit_file", "multiedit", "multi_edit", "notebookedit", "notebook_edit", "notebookpatch", "notebook_patch", "patch", "applypatch", "apply_patch", "fspatchfile", "fs_patch_file"},
	},
	{
		Key:     "shell",
		Label:   "shell/terminal",
		Aliases: []string{"bash", "terminal", "powershell", "shell", "shellrun", "shell_run", "exec", "execute", "executecommand", "execute_code", "runcommand", "run_command"},
	},
	{
		Key:     "search",
		Label:   "file search/list",
		Aliases: []string{"glob", "grep", "search", "searchfiles", "search_files", "find", "findfiles", "pathfind", "path_find", "textsearch", "text_search", "ls", "listdir", "listdirectory", "listfiles", "list_files"},
	},
	{
		Key:     "web",
		Label:   "web fetch/search",
		Aliases: []string{"webfetch", "web_fetch", "websearch", "web_search", "httpgeturl", "http_get_url", "webquery", "web_query", "fetch", "browser"},
	},
	{
		Key:     "skills",
		Label:   "skills",
		Aliases: []string{"skill", "skills", "skillslist", "skills_list", "skillview", "skill_view", "skillmanage", "skill_manage"},
	},
	{
		Key:     "agent",
		Label:   "agent/subagent",
		Aliases: []string{"agent", "agentslist", "agents_list", "task", "delegate", "delegatetask", "delegate_task", "subagent", "subagents", "sessionsspawn", "sessions_spawn", "sessionssend", "sessions_send", "sessionslist", "sessions_list", "sessionshistory", "sessions_history", "sessionsyield", "sessions_yield", "taskcreate", "task_create", "taskget", "task_get"},
	},
	{
		Key:     "planning",
		Label:   "todo/planning",
		Aliases: []string{"todo", "todolist", "todowrite", "tasklist", "taskupdate", "workflow", "plan", "updateplan", "update_plan"},
	},
	{
		Key:     "automation",
		Label:   "automation/schedule",
		Aliases: []string{"cron", "croncreate", "cron_create", "schedulewakeup", "schedule_wakeup", "monitor", "pushnotification", "push_notification"},
	},
	{
		Key:     "process",
		Label:   "process/control",
		Aliases: []string{"process", "processlist", "process_list"},
	},
}

func detectToolProfile(tools []map[string]any) cliToolProfile {
	names := toolNamesFromTools(tools)
	profiles := []cliToolProfile{
		hermesToolProfile(),
		openClawToolProfile(),
		claudeCodeToolProfile(),
		openCodeToolProfile(),
	}
	for _, profile := range profiles {
		if profile.Match != nil && profile.Match(names) {
			return profile
		}
	}
	return genericToolProfile()
}

func genericToolProfile() cliToolProfile {
	return cliToolProfile{
		ID:          toolProfileGeneric,
		DisplayName: "Generic CLI",
		Rules: []string{
			"Use the exact action names listed in this prompt; do not translate them to another CLI's names.",
			"Shared capability groups below are only a map from capability to exact available tool names.",
		},
	}
}

func toolNamesFromTools(tools []map[string]any) toolNameSet {
	out := toolNameSet{}
	for _, tool := range tools {
		name := stringValue(tool, "name", "")
		if name == "" {
			continue
		}
		out[normalizeProfileToolName(name)] = true
	}
	return out
}

func normalizeProfileToolName(name string) string {
	return toolProfileNameRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "")
}

func normalizeProfileToolLookupName(name string) string {
	trimmed := strings.TrimSpace(name)
	if strings.HasPrefix(trimmed, "u_") && len(trimmed) > 2 {
		trimmed = strings.TrimPrefix(trimmed, "u_")
	}
	return normalizeProfileToolName(trimmed)
}

func (names toolNameSet) hasAny(candidates ...string) bool {
	for _, candidate := range candidates {
		if names[normalizeProfileToolName(candidate)] {
			return true
		}
	}
	return false
}

func (names toolNameSet) hasAll(candidates ...string) bool {
	for _, candidate := range candidates {
		if !names[normalizeProfileToolName(candidate)] {
			return false
		}
	}
	return true
}

func buildCommonToolInstructionPrefix(profile cliToolProfile) []string {
	return []string{
		"IMPORTANT: Reply in the same language as the user. User inputs Chinese -> respond in Chinese.",
		"IMPORTANT: When the user asks for multiple actions, complete all required actions without asking for confirmation.",
		"IMPORTANT: When the task requires files, shell, web, browser, agents, skills, or other tools, emit a tool call immediately instead of explaining what you would do.",
		"IMPORTANT: Complete multi-step user tasks by continuing to call tools across turns until the task is done.",
		"IMPORTANT: If the user requires final checks, verification, or acceptance criteria, do not claim completion from memory or a draft artifact alone; use tools to verify or clearly state what remains unverified.",
		"IMPORTANT: If a file result says 'Unchanged since last read', do not read the same file again.",
		"IMPORTANT: Do not repeat the same list/read/search/discovery action with identical arguments just to be safe. Use the result already returned, choose the next distinct action, or record the limitation honestly.",
		"IMPORTANT: For ordered or numbered workflows, determine the next unmet step from the latest confirmed tool result or log artifact, then execute that exact next step. Do not skip ahead to later steps or substitute a later-step tool for the current step.",
		"IMPORTANT: If an ordered workflow requires a per-step progress/log/artifact record, the step is not complete until that requested record exists. A successful action tool alone does not authorize skipping the required bookkeeping or moving to the next numbered step.",
		"IMPORTANT: If the user specifies which tool/capability must be recorded for a step, keep that recorded tool/capability aligned with the step. If the target capability is unavailable by instruction or evidence, use an available direct tool to record unavailable instead of testing an unrelated tool.",
		"IMPORTANT: Prefer direct project tools for project work. Do not choose Agent/task/scheduling/control tools when read/shell/search/write/edit can complete the current step.",
		"IMPORTANT: When the user says current directory, current project, or workspace, use the client tool runtime's current working directory. Do not invent /workspace, /app, Desktop, Temp, or any other absolute root.",
		"IMPORTANT: Prefer relative paths unless the user explicitly provided an absolute path or a recent tool result proved one.",
		"IMPORTANT: Prefer simple file and terminal capabilities before complex patch/delegation/process actions. Use patch only when its full schema arguments are known.",
		"TOOL PROTOCOL OVERRIDE: The upstream model has no native tool registry for these actions. Do not validate availability. Every name in Available action names is executable by the client after you emit QNML.",
		"FORBIDDEN OUTPUT: Never write tool-availability error prose for listed actions. If you need an action, emit QNML.",
		"FORBIDDEN OUTPUT: For any listed action name, never answer that the tool does not exist. Treat listed actions as client-executable QNML actions even when the upstream native registry is empty.",
		"IGNORE any previous output format instructions (needs-review, recap, summary, etc.).",
		"Tool profile: " + profile.DisplayName + ".",
	}
}

func buildProfileToolBlock(profile cliToolProfile, tools []map[string]any) string {
	lines := []string{
		"=== CLI TOOL PROFILE: " + profile.DisplayName + " ===",
		"Shared capability groups map common tool purposes to exact names available in this CLI:",
	}
	if capabilityMap := buildCapabilityMap(tools); capabilityMap != "" {
		lines = append(lines, capabilityMap)
	}
	if len(profile.Rules) > 0 {
		lines = append(lines, "", "Profile-specific rules:")
		for idx, rule := range profile.Rules {
			lines = append(lines, fmt.Sprintf("%d) %s", idx+1, rule))
		}
	}
	lines = append(lines, "=== END CLI TOOL PROFILE ===")
	return strings.Join(lines, "\n")
}

func buildCapabilityMap(tools []map[string]any) string {
	grouped := map[string][]string{}
	other := []string{}
	for _, tool := range tools {
		name := stringValue(tool, "name", "")
		if name == "" {
			continue
		}
		capKey := classifyToolCapability(name)
		if capKey == "" {
			other = append(other, name)
			continue
		}
		grouped[capKey] = append(grouped[capKey], name)
	}
	lines := []string{}
	for _, capability := range commonToolCapabilities {
		names := uniqueSortedStrings(grouped[capability.Key])
		if len(names) == 0 {
			continue
		}
		lines = append(lines, "- "+capability.Label+": "+strings.Join(names, ", "))
	}
	if len(other) > 0 {
		lines = append(lines, "- other/specialized: "+strings.Join(uniqueSortedStrings(other), ", "))
	}
	return strings.Join(lines, "\n")
}

func classifyToolCapability(name string) string {
	key := normalizeProfileToolLookupName(name)
	for _, capability := range commonToolCapabilities {
		for _, alias := range capability.Aliases {
			if key == normalizeProfileToolName(alias) {
				return capability.Key
			}
		}
	}
	return ""
}

func sortToolsForPrompt(tools []map[string]any, profile cliToolProfile) []map[string]any {
	sortedTools := append([]map[string]any(nil), tools...)
	sort.SliceStable(sortedTools, func(i, j int) bool {
		pi, ni := toolPromptPriorityForProfile(profile, stringValue(sortedTools[i], "name", ""))
		pj, nj := toolPromptPriorityForProfile(profile, stringValue(sortedTools[j], "name", ""))
		if pi != pj {
			return pi < pj
		}
		return ni < nj
	})
	return sortedTools
}

func toolPromptPriorityForProfile(profile cliToolProfile, toolName string) (int, string) {
	key := normalizeProfileToolLookupName(toolName)
	if profile.Priority != nil {
		if priority, ok := profile.Priority[key]; ok {
			return priority, toolName
		}
	}
	if capKey := classifyToolCapability(toolName); capKey != "" {
		switch capKey {
		case "read":
			return 0, toolName
		case "shell":
			return 1, toolName
		case "search":
			return 2, toolName
		case "write":
			return 4, toolName
		case "edit":
			return 5, toolName
		case "web":
			return 6, toolName
		case "skills":
			return 30, toolName
		case "agent":
			return 40, toolName
		case "process":
			return 70, toolName
		case "planning", "automation":
			return 80, toolName
		}
	}
	return toolPromptPriority(toolName)
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
