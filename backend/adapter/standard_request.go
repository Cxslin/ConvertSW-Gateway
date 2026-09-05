package adapter

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	workspaceNoticeBlockRe  = regexp.MustCompile(`(?is)\[WORKSPACE (?:ROOT - MUST OBEY|CONTEXT)\].*?\[/WORKSPACE (?:ROOT|CONTEXT)\]`)
	topicURLRe              = regexp.MustCompile(`https?://[^\s)'\"<>]+`)
	topicWinPathRe          = regexp.MustCompile(`[A-Za-z]:[\\/](?:[^\s<>'\"|:?*]+[\\/])*[^\s<>'\"|:?*\\/]+`)
	topicUnixPathRe         = regexp.MustCompile(`/(?:[\w.-]+/)+[\w.-]+`)
	topicCamelRe            = regexp.MustCompile(`[a-z][a-z0-9]*(?:[A-Z][a-z0-9]+)+`)
	topicFileRe             = regexp.MustCompile(`\b[\w-]+\.[a-zA-Z0-9]{1,8}\b`)
	qnmlHistoryInvokeRe     = regexp.MustCompile(`(?is)<\|QNML\|invoke\s+name="([^"]+)"[^>]*>(.*?)</\|QNML\|invoke>`)
	orderedStepLineRe       = regexp.MustCompile(`(?i)^\s*(?:[-*]\s*)?(?:R\d{2,4}\b|Round\s*\d+\b|Step\s*\d+\b|第\s*\d+\s*[步轮]|(?:\d{1,3})[.)、:：])`)
	orderedRoundLabelRe     = regexp.MustCompile(`(?i)\bR(\d{3,4})\b`)
	orderedRoundRecordRe    = regexp.MustCompile(`(?is)\{[^{}]*"round"\s*:\s*"R(\d{3,4})"[^{}]*"tool"\s*:[^{}]*\}`)
	orderedRoundRecordAltRe = regexp.MustCompile(`(?is)\{[^{}]*"tool"\s*:[^{}]*"round"\s*:\s*"R(\d{3,4})"[^{}]*\}`)
)

var topicStopwords = map[string]bool{
	"http": true, "https": true, "the": true, "and": true, "for": true, "with": true, "from": true,
	"into": true, "this": true, "that": true, "then": true, "have": true, "been": true, "will": true,
	"should": true, "would": true, "also": true, "read": true, "write": true, "readfile": true,
	"search": true, "grep": true, "glob": true, "tool": true, "tools": true,
	"给我": true, "这个": true, "那个": true, "就是": true, "然后": true, "一下": true, "请": true,
	"帮我": true, "需要": true, "进行": true, "操作": true,
}

type StandardRequest struct {
	Prompt          string
	ResponseModel   string
	ResolvedModel   string
	Surface         string
	Stream          bool
	Tools           []map[string]any
	ToolNames       []string
	ToolEnabled     bool
	ChatType        string
	ThinkingEnabled *bool
	ForceThinking   bool
	EnableSearch    bool
	ModelMode       string

	RepeatedToolName          string
	RepeatedToolSignature     string
	RepeatedToolCount         int
	LatestMessageIsToolResult bool
}

type ModelMode struct {
	RequestedModel string
	BaseModel      string
	ChatType       string
	ForceThinking  bool
	Mode           string
}

type ModelModeResolver func(modelID, defaultModel string) ModelMode
type ModelResolver func(name string) string

// BuildChatStandardRequest converts OpenAI-style chat bodies into the internal
// normalized request used by all compatibility surfaces.
func BuildChatStandardRequest(body map[string]any, defaultModel, surface string, resolveModel ModelResolver, parseModelMode ModelModeResolver) StandardRequest {
	requested := stringValue(body, "model", defaultModel)
	mode := parseModelMode(requested, defaultModel)
	thinking := ExtractThinkingEnabled(body)
	if mode.ForceThinking {
		v := true
		thinking = &v
	}
	enableSearch := false
	if b := coerceBool(body["enable_search"]); b != nil {
		enableSearch = *b
	}
	if mode.Mode == "search" || mode.ChatType == "deep_research" {
		enableSearch = true
	}
	messages := anyList(body["messages"])
	repeated, repeatedCount := latestRepeatedToolCallActivity(messages, 10)
	prompt, tools := MessagesToPrompt(body)
	toolEnabled := len(tools) > 0
	toolNames := []string{}
	for _, tool := range tools {
		if name := stringValue(tool, "name", ""); name != "" {
			toolNames = append(toolNames, name)
		}
	}
	return StandardRequest{
		Prompt: prompt, ResponseModel: requested, ResolvedModel: resolveTransportModel(mode, resolveModel),
		Surface: surface, Stream: boolValue(body["stream"]), Tools: tools, ToolNames: toolNames,
		ToolEnabled: toolEnabled, ChatType: mode.ChatType, ThinkingEnabled: thinking,
		ForceThinking: mode.ForceThinking, EnableSearch: enableSearch, ModelMode: mode.Mode,
		RepeatedToolName: repeated.Name, RepeatedToolSignature: repeated.Signature, RepeatedToolCount: repeatedCount,
		LatestMessageIsToolResult: latestMessageIsToolResult(messages),
	}
}

func resolveTransportModel(mode ModelMode, resolveModel ModelResolver) string {
	base := strings.TrimSpace(mode.BaseModel)
	if base == "" {
		base = strings.TrimSpace(mode.RequestedModel)
	}
	return resolveModel(base)
}

func ExtractThinkingEnabled(body map[string]any) *bool {
	if value, ok := body["enable_thinking"]; ok {
		return coerceBool(value)
	}
	if value, ok := body["thinking"]; ok {
		if m, ok := value.(map[string]any); ok {
			for _, key := range []string{"enabled", "enable", "enabled_thinking", "enable_thinking"} {
				if inner, ok := m[key]; ok {
					return coerceBool(inner)
				}
			}
		}
		return coerceBool(value)
	}
	if value, ok := body["thinking_mode"]; ok {
		return coerceBool(value)
	}
	return nil
}

func MessagesToPrompt(body map[string]any) (string, []map[string]any) {
	tools := NormalizeTools(body["tools"])
	messages := anyList(body["messages"])
	if len(tools) > 0 {
		return messagesToToolPrompt(messages, tools), tools
	}

	parts := []string{noClientToolsAuthenticityNotice()}
	for _, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		role := stringValue(msg, "role", "user")
		text := ExtractContentText(msg["content"])
		if text == "" {
			continue
		}
		switch role {
		case "system", "developer":
			parts = append(parts, "[System]\n"+text)
		case "assistant":
			parts = append(parts, "[Assistant]\n"+text)
		case "tool":
			parts = append(parts, "[Tool Result]\n"+text)
		default:
			parts = append(parts, "[User]\n"+text)
		}
	}
	return strings.Join(parts, "\n\n"), tools
}

func noClientToolsAuthenticityNotice() string {
	return strings.Join([]string{
		"[System]",
		"Client-side tools are not available in this request. If the task requires file access, command execution, web access, browser actions, agents, skills, or artifact verification, do not claim that you performed those actions. Answer only from the supplied conversation context and clearly state any execution or verification that remains unavailable.",
	}, "\n")
}

func messagesToToolPrompt(messages []any, tools []map[string]any) string {
	const maxHistoryMessages = 30
	const maxPromptChars = 40000

	originalMessages := append([]any(nil), messages...)
	messages = applyTopicIsolation(messages)
	workspaceNotice := extractWorkspaceNotice(messages)
	taskMemory := buildTaskMemoryBlock(messages, tools)
	trimmedMessages := trimToolHistoryMessages(messages, maxHistoryMessages)
	droppedNotice := buildDroppedHistoryNotice(originalMessages, trimmedMessages, tools)

	parts := []string{BuildToolInstructions(tools)}
	if workspaceNotice != "" {
		parts = append(parts, workspaceNotice)
	}
	if taskMemory != "" {
		parts = append(parts, taskMemory)
	}
	if droppedNotice != "" {
		parts = append(parts, droppedNotice)
	}
	if fewShot := buildFewShotBlock(qwenSafePromptTools(sortToolsForPrompt(tools, detectToolProfile(tools)))); fewShot != "" {
		parts = append(parts, fewShot)
	}
	history := renderToolHistory(trimmedMessages, maxHistoryMessages)
	used := len(strings.Join(parts, "\n\n"))
	for _, line := range history {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if used+len(line)+2 > maxPromptChars && len(parts) > 1 {
			break
		}
		parts = append(parts, line)
		used += len(line) + 2
	}
	if first := firstUserTaskLine(messages); first != "" && !historyContainsOriginal(history, first) {
		insertAt := 1
		parts = append(parts[:insertAt], append([]string{first}, parts[insertAt:]...)...)
	}
	if notice := buildToolResultFollowupNotice(messages); notice != "" {
		parts = append(parts, notice)
	}
	parts = append(parts, "Assistant:")
	return strings.Join(parts, "\n\n")
}

func renderAttachmentPlaceholder(block map[string]any, image bool) string {
	name := firstNonEmpty(
		stringValue(block, "filename", ""),
		stringValue(block, "name", ""),
		anyString(block["file_id"], ""),
	)
	if name == "" {
		if nested, ok := block["image_url"].(map[string]any); ok {
			name = firstNonEmpty(anyString(nested["url"], ""), anyString(nested["file_id"], "image"))
		}
	}
	if name == "" {
		if image {
			name = "image"
		} else {
			name = "file"
		}
	}
	if image {
		return "[Attached image: " + name + "]"
	}
	return "[Attached file: " + name + "]"
}

func applyTopicIsolation(messages []any) []any {
	if len(messages) < 3 {
		return messages
	}
	firstUserIdx := -1
	firstUserText := ""
	for idx, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok || stringValue(msg, "role", "") != "user" {
			continue
		}
		if text := strings.TrimSpace(extractUserTextOnly(msg["content"])); text != "" {
			firstUserIdx = idx
			firstUserText = text
			break
		}
	}
	if firstUserIdx < 0 {
		return messages
	}
	lastUserIdx := -1
	lastUserText := ""
	for idx := len(messages) - 1; idx >= 0; idx-- {
		msg, ok := messages[idx].(map[string]any)
		if !ok || stringValue(msg, "role", "") != "user" {
			continue
		}
		if text := strings.TrimSpace(extractUserTextOnly(msg["content"])); text != "" {
			lastUserIdx = idx
			lastUserText = text
			break
		}
	}
	if lastUserIdx < 0 || lastUserIdx == firstUserIdx {
		return messages
	}
	if !detectTopicChange(firstUserText, lastUserText) {
		return messages
	}
	out := make([]any, 0, len(messages))
	for idx, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		role := stringValue(msg, "role", "")
		if role == "system" || role == "developer" || idx >= lastUserIdx {
			out = append(out, raw)
		}
	}
	if len(out) == 0 {
		return messages
	}
	return out
}

func detectTopicChange(firstUserText, lastUserText string) bool {
	firstUserText = strings.TrimSpace(firstUserText)
	lastUserText = strings.TrimSpace(lastUserText)
	if firstUserText == "" || lastUserText == "" || firstUserText == lastUserText {
		return false
	}
	first := extractTopicEntities(firstUserText)
	last := extractTopicEntities(lastUserText)
	if len(first) == 0 || len(last) == 0 {
		return false
	}
	intersect := 0
	union := map[string]struct{}{}
	for key := range first {
		union[key] = struct{}{}
		if _, ok := last[key]; ok {
			intersect++
		}
	}
	for key := range last {
		union[key] = struct{}{}
	}
	if len(union) == 0 {
		return false
	}
	return float64(intersect)/float64(len(union)) < 0.1
}

func extractTopicEntities(text string) map[string]struct{} {
	text = strings.TrimSpace(text)
	out := map[string]struct{}{}
	if text == "" {
		return out
	}
	add := func(value string) {
		value = strings.TrimSpace(value)
		value = strings.TrimRight(value, ".,;:)]}>，。；：")
		if len([]rune(value)) < 4 {
			return
		}
		if topicStopwords[strings.ToLower(value)] {
			return
		}
		out[value] = struct{}{}
	}
	for _, match := range topicURLRe.FindAllString(text, -1) {
		add(match)
		if parts := regexp.MustCompile(`//([^/]+)`).FindStringSubmatch(match); len(parts) > 1 {
			add(strings.ToLower(parts[1]))
		}
	}
	for _, match := range topicWinPathRe.FindAllString(text, -1) {
		match = strings.ReplaceAll(match, "\\", "/")
		add(match)
		if idx := strings.LastIndex(match, "/"); idx >= 0 && idx+1 < len(match) {
			add(match[idx+1:])
		}
	}
	for _, match := range topicUnixPathRe.FindAllString(text, -1) {
		add(match)
		if idx := strings.LastIndex(match, "/"); idx >= 0 && idx+1 < len(match) {
			add(match[idx+1:])
		}
	}
	for _, match := range topicCamelRe.FindAllString(text, -1) {
		add(match)
	}
	for _, match := range topicFileRe.FindAllString(text, -1) {
		add(match)
	}
	return out
}

func extractWorkspaceNotice(messages []any) string {
	for _, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		role := stringValue(msg, "role", "")
		if role != "system" && role != "developer" {
			continue
		}
		text := ExtractContentText(msg["content"])
		if text == "" || (!strings.Contains(text, "[WORKSPACE ROOT - MUST OBEY]") && !strings.Contains(text, "[WORKSPACE CONTEXT]")) {
			continue
		}
		if match := workspaceNoticeBlockRe.FindString(text); strings.TrimSpace(match) != "" {
			return strings.TrimSpace(match)
		}
		return strings.TrimSpace(text)
	}
	return ""
}

func trimToolHistoryMessages(messages []any, maxHistoryMessages int) []any {
	if len(messages) <= maxHistoryMessages || maxHistoryMessages <= 0 {
		return messages
	}
	start := len(messages) - maxHistoryMessages
	firstUserIdx := -1
	for idx, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok || stringValue(msg, "role", "") != "user" {
			continue
		}
		if strings.TrimSpace(extractUserTextOnly(msg["content"])) != "" {
			firstUserIdx = idx
			break
		}
	}
	out := make([]any, 0, len(messages))
	for idx, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		role := stringValue(msg, "role", "")
		if role == "system" || role == "developer" || idx >= start || idx == firstUserIdx {
			out = append(out, raw)
		}
	}
	return out
}

func buildTaskMemoryBlock(messages []any, tools []map[string]any) string {
	if len(messages) == 0 || len(tools) == 0 {
		return ""
	}
	originalGoal := firstNonEmptyUserText(messages)
	currentGoal := latestNonEmptyUserText(messages)
	latestToolResult := latestToolResultSummary(messages)
	recentActivity := collectRecentToolActivity(messages, 6)
	toolCallCount, toolResultCount := countToolEvents(messages)

	lines := []string{
		"<TASK MEMORY - DO NOT DROP>",
		"This block is stable task memory for long tool chains.",
		"RAW HISTORY POLICY: The raw transcript may be windowed; this TASK MEMORY carries the task across unlimited tool turns.",
		fmt.Sprintf("TOOL PROGRESS: %d tool call(s), %d tool result(s) observed so far.", toolCallCount, toolResultCount),
	}
	if originalGoal != "" {
		lines = append(lines, "ORIGINAL GOAL: "+clipText(originalGoal, 1200))
		if steps := extractOrderedStepExcerpt(originalGoal, 36, 5000); steps != "" {
			lines = append(lines, "ORDERED STEP EXCERPT FROM ORIGINAL GOAL:")
			lines = append(lines, steps)
			lines = append(lines, "RULE: The ordered step excerpt is authoritative for later steps that may be clipped from raw history.")
		}
		if state := buildOrderedRoundRecordState(messages, originalGoal); state != "" {
			lines = append(lines, state)
		}
	}
	if currentGoal != "" && currentGoal != originalGoal {
		lines = append(lines, "CURRENT USER GOAL: "+clipText(currentGoal, 900))
	}
	if latestToolResult != "" {
		lines = append(lines, "LATEST TOOL RESULT: "+clipText(latestToolResult, 900))
	}
	if len(recentActivity) > 0 {
		lines = append(lines, "RECENT TOOL ACTIVITY:")
		for _, item := range recentActivity {
			lines = append(lines, "- "+clipText(item, 260))
		}
	}
	if notice := buildRepeatedToolCallNotice(messages); notice != "" {
		lines = append(lines, notice)
	}
	lines = append(lines, "RULE: Continue from the latest tool result and original goal. Do not restart, forget the task, or switch to review/summary unless the user asked for that.")
	lines = append(lines, "</TASK MEMORY>")
	return strings.Join(lines, "\n")
}

func buildOrderedRoundRecordState(messages []any, originalGoal string) string {
	planned := extractRoundNumbers(originalGoal, orderedRoundLabelRe)
	if len(planned) == 0 {
		return ""
	}
	recorded := map[int]bool{}
	for _, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for _, call := range messageToolCallActivities(msg) {
			for _, round := range extractRoundRecords(call.Args) {
				recorded[round] = true
			}
		}
		for _, round := range extractRoundRecords(contentToText(msg["content"])) {
			recorded[round] = true
		}
	}
	recordedPlanned := []int{}
	earliestMissing := 0
	for _, round := range planned {
		if recorded[round] {
			recordedPlanned = append(recordedPlanned, round)
			continue
		}
		if earliestMissing == 0 {
			earliestMissing = round
		}
	}
	if len(recordedPlanned) == 0 && earliestMissing == 0 {
		return ""
	}
	lines := []string{"ORDERED ROUND RECORD STATE:"}
	if len(recordedPlanned) > 0 {
		lines = append(lines, "RECORDED ROUND JSON ENTRIES OBSERVED: "+formatRoundList(recordedPlanned, 20))
	}
	if earliestMissing > 0 {
		lines = append(lines, fmt.Sprintf("EARLIEST PLANNED ROUND WITHOUT OBSERVED JSON RECORD: R%03d", earliestMissing))
		lines = append(lines, "RULE: For numbered workflows with per-round records, resume at the earliest missing record before advancing to later rounds.")
	}
	return strings.Join(lines, "\n")
}

func extractRoundNumbers(text string, re *regexp.Regexp) []int {
	if strings.TrimSpace(text) == "" || re == nil {
		return nil
	}
	seen := map[int]bool{}
	out := []int{}
	for _, match := range re.FindAllStringSubmatch(text, -1) {
		if len(match) < 2 {
			continue
		}
		n, err := strconv.Atoi(match[1])
		if err != nil || n <= 0 || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

func extractRoundRecords(text string) []int {
	seen := map[int]bool{}
	add := func(values []int) {
		for _, value := range values {
			seen[value] = true
		}
	}
	add(extractRoundNumbers(text, orderedRoundRecordRe))
	add(extractRoundNumbers(text, orderedRoundRecordAltRe))
	normalized := strings.ReplaceAll(text, "\\", "")
	if normalized != text {
		add(extractRoundNumbers(normalized, orderedRoundRecordRe))
		add(extractRoundNumbers(normalized, orderedRoundRecordAltRe))
	}
	var decoded any
	if json.Unmarshal([]byte(text), &decoded) == nil {
		collectRoundRecordsFromValue(decoded, seen)
	}
	out := make([]int, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Ints(out)
	return out
}

func collectRoundRecordsFromValue(value any, seen map[int]bool) {
	switch v := value.(type) {
	case string:
		for _, round := range extractRoundRecords(v) {
			seen[round] = true
		}
	case []any:
		for _, item := range v {
			collectRoundRecordsFromValue(item, seen)
		}
	case map[string]any:
		if round, ok := progressRoundFromMap(v); ok {
			seen[round] = true
		}
		for _, item := range v {
			collectRoundRecordsFromValue(item, seen)
		}
	}
}

func progressRoundFromMap(value map[string]any) (int, bool) {
	if value == nil || value["tool"] == nil {
		return 0, false
	}
	roundText := strings.TrimSpace(anyString(value["round"], ""))
	if len(roundText) < 2 || (roundText[0] != 'R' && roundText[0] != 'r') {
		return 0, false
	}
	round, err := strconv.Atoi(roundText[1:])
	if err != nil || round <= 0 {
		return 0, false
	}
	return round, true
}

func formatRoundList(rounds []int, limit int) string {
	if len(rounds) == 0 {
		return ""
	}
	if limit <= 0 || limit > len(rounds) {
		limit = len(rounds)
	}
	out := make([]string, 0, limit+1)
	for _, round := range rounds[:limit] {
		out = append(out, fmt.Sprintf("R%03d", round))
	}
	if len(rounds) > limit {
		out = append(out, fmt.Sprintf("...+%d", len(rounds)-limit))
	}
	return strings.Join(out, ",")
}

func extractOrderedStepExcerpt(text string, maxLines, maxChars int) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	out := []string{}
	for _, rawLine := range strings.Split(text, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || !orderedStepLineRe.MatchString(line) {
			continue
		}
		out = append(out, clipText(line, 360))
		if maxLines > 0 && len(out) >= maxLines {
			break
		}
	}
	if len(out) == 0 {
		return ""
	}
	joined := strings.Join(out, "\n")
	if maxChars > 0 && len(joined) > maxChars {
		joined = clipText(joined, maxChars)
	}
	return joined
}

func buildDroppedHistoryNotice(originalMessages, keptMessages []any, tools []map[string]any) string {
	if len(originalMessages) == 0 || len(tools) == 0 {
		return ""
	}
	dropped := len(originalMessages) - len(keptMessages)
	if dropped <= 0 {
		return ""
	}
	lines := []string{
		"<HISTORY COMPACTION NOTICE>",
		fmt.Sprintf("%d older message(s) were compacted out of the inline history.", dropped),
		"The original goal and latest tool result in TASK MEMORY remain authoritative.",
	}
	activity := collectRecentToolActivity(originalMessages, 4)
	if len(activity) > 0 {
		lines = append(lines, "Last known tool activity before/around compaction:")
		for _, item := range activity {
			lines = append(lines, "- "+clipText(item, 260))
		}
	}
	lines = append(lines, "</HISTORY COMPACTION NOTICE>")
	return strings.Join(lines, "\n")
}

func firstNonEmptyUserText(messages []any) string {
	for _, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok || stringValue(msg, "role", "") != "user" {
			continue
		}
		if text := strings.TrimSpace(extractUserTextOnly(msg["content"])); text != "" {
			return text
		}
	}
	return ""
}

func latestNonEmptyUserText(messages []any) string {
	for idx := len(messages) - 1; idx >= 0; idx-- {
		msg, ok := messages[idx].(map[string]any)
		if !ok || stringValue(msg, "role", "") != "user" {
			continue
		}
		if text := strings.TrimSpace(extractUserTextOnly(msg["content"])); text != "" {
			return text
		}
	}
	return ""
}

func latestToolResultSummary(messages []any) string {
	for idx := len(messages) - 1; idx >= 0; idx-- {
		msg, ok := messages[idx].(map[string]any)
		if !ok {
			continue
		}
		if summary := messageLatestToolResultSummary(msg); summary != "" {
			return summary
		}
	}
	return ""
}

func collectRecentToolActivity(messages []any, limit int) []string {
	if limit <= 0 {
		return nil
	}
	reversed := make([]string, 0, limit)
	for idx := len(messages) - 1; idx >= 0 && len(reversed) < limit; idx-- {
		msg, ok := messages[idx].(map[string]any)
		if !ok {
			continue
		}
		for _, summary := range reverseStrings(messageToolResultSummaries(msg)) {
			reversed = append(reversed, "result: "+summary)
			if len(reversed) >= limit {
				break
			}
		}
		if len(reversed) >= limit {
			break
		}
		for _, summary := range reverseStrings(messageToolCallSummaries(msg)) {
			reversed = append(reversed, "call: "+summary)
			if len(reversed) >= limit {
				break
			}
		}
	}
	return reverseStrings(reversed)
}

type toolCallActivity struct {
	Name      string
	Args      string
	Signature string
}

func buildRepeatedToolCallNotice(messages []any) string {
	call, count := latestRepeatedToolCallActivity(messages, 10)
	name := qwenSafeToolName(call.Name)
	if name == "" || count < 2 {
		return ""
	}
	return fmt.Sprintf(
		"REPETITION NOTICE: recent activity shows action %s was called %d times with the same arguments. Do not call that same action again unless new evidence or different arguments require it; use the latest result, move to the next distinct step, or record the limitation honestly.",
		name,
		count,
	)
}

func latestRepeatedToolCall(messages []any, limit int) (string, int) {
	call, count := latestRepeatedToolCallActivity(messages, limit)
	return call.Name, count
}

func latestRepeatedToolCallActivity(messages []any, limit int) (toolCallActivity, int) {
	calls := collectRecentToolCalls(messages, limit)
	if len(calls) < 2 {
		return toolCallActivity{}, 0
	}
	latest := calls[len(calls)-1]
	if latest.Name == "" || latest.Signature == "" {
		return toolCallActivity{}, 0
	}
	count := 1
	for idx := len(calls) - 2; idx >= 0; idx-- {
		call := calls[idx]
		if !strings.EqualFold(call.Name, latest.Name) || call.Signature != latest.Signature {
			break
		}
		count++
	}
	if count < 2 {
		return toolCallActivity{}, 0
	}
	return latest, count
}

func collectRecentToolCalls(messages []any, limit int) []toolCallActivity {
	if limit <= 0 {
		return nil
	}
	reversed := make([]toolCallActivity, 0, limit)
	for idx := len(messages) - 1; idx >= 0 && len(reversed) < limit; idx-- {
		msg, ok := messages[idx].(map[string]any)
		if !ok {
			continue
		}
		for _, call := range reverseToolCalls(messageToolCallActivities(msg)) {
			reversed = append(reversed, call)
			if len(reversed) >= limit {
				break
			}
		}
	}
	return reverseToolCalls(reversed)
}

func messageToolCallActivities(msg map[string]any) []toolCallActivity {
	out := []toolCallActivity{}
	for _, rawCall := range anyList(msg["tool_calls"]) {
		call, ok := rawCall.(map[string]any)
		if !ok {
			continue
		}
		name, args := toolCallNameAndArgs(call)
		if name != "" {
			out = append(out, makeToolCallActivity(name, args))
		}
	}
	for _, rawPart := range anyList(msg["content"]) {
		part, ok := rawPart.(map[string]any)
		if !ok || stringValue(part, "type", "") != "tool_use" {
			continue
		}
		name := stringValue(part, "name", "")
		args := ""
		if part["input"] != nil {
			raw, _ := json.Marshal(part["input"])
			args = string(raw)
		}
		if name != "" {
			out = append(out, makeToolCallActivity(name, args))
		}
	}
	if len(out) == 0 {
		text := strings.TrimSpace(ExtractContentText(msg["content"]))
		for _, match := range qnmlHistoryInvokeRe.FindAllStringSubmatch(text, -1) {
			if len(match) >= 3 {
				out = append(out, makeToolCallActivity(html.UnescapeString(match[1]), match[2]))
			}
		}
	}
	return out
}

func makeToolCallActivity(name, args string) toolCallActivity {
	name = strings.TrimSpace(name)
	return toolCallActivity{
		Name:      name,
		Args:      args,
		Signature: strings.ToLower(name) + "\x00" + normalizeToolCallArgs(args),
	}
}

func normalizeToolCallArgs(args string) string {
	args = strings.TrimSpace(args)
	if args == "" {
		return "{}"
	}
	var decoded any
	if json.Unmarshal([]byte(args), &decoded) == nil {
		normalized, _ := json.Marshal(decoded)
		return string(normalized)
	}
	return strings.Join(strings.Fields(args), " ")
}

func reverseToolCalls(values []toolCallActivity) []toolCallActivity {
	out := append([]toolCallActivity(nil), values...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func countToolEvents(messages []any) (int, int) {
	calls := 0
	results := 0
	for _, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		calls += len(messageToolCallSummaries(msg))
		results += len(messageToolResultSummaries(msg))
	}
	return calls, results
}

func messageToolCallSummaries(msg map[string]any) []string {
	out := []string{}
	for _, rawCall := range anyList(msg["tool_calls"]) {
		call, ok := rawCall.(map[string]any)
		if !ok {
			continue
		}
		name, args := toolCallNameAndArgs(call)
		if name != "" {
			summary := qwenSafeToolName(name)
			if args != "" {
				summary += " " + clipText(args, 160)
			}
			out = append(out, summary)
		}
	}
	for _, rawPart := range anyList(msg["content"]) {
		part, ok := rawPart.(map[string]any)
		if !ok || stringValue(part, "type", "") != "tool_use" {
			continue
		}
		name := stringValue(part, "name", "")
		args := ""
		if part["input"] != nil {
			raw, _ := json.Marshal(part["input"])
			args = string(raw)
		}
		if name != "" {
			summary := qwenSafeToolName(name)
			if args != "" {
				summary += " " + clipText(args, 160)
			}
			out = append(out, summary)
		}
	}
	return out
}

func messageToolResultSummaries(msg map[string]any) []string {
	role := stringValue(msg, "role", "")
	out := []string{}
	if role == "tool" {
		if text := strings.TrimSpace(ExtractContentText(msg["content"])); text != "" {
			out = append(out, clipText(text, 220))
		}
	}
	for _, rawPart := range anyList(msg["content"]) {
		part, ok := rawPart.(map[string]any)
		if !ok {
			continue
		}
		partType := stringValue(part, "type", "")
		if partType != "tool_result" && partType != "function_call_output" {
			continue
		}
		if text := strings.TrimSpace(contentToText(part["content"])); text != "" {
			out = append(out, clipText(text, 220))
		}
	}
	return out
}

func messageLatestToolResultSummary(msg map[string]any) string {
	summaries := messageToolResultSummaries(msg)
	if len(summaries) == 0 {
		return ""
	}
	return summaries[len(summaries)-1]
}

func reverseStrings(values []string) []string {
	out := append([]string(nil), values...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func renderToolHistory(messages []any, maxMessages int) []string {
	start := 0
	if len(messages) > maxMessages {
		start = len(messages) - maxMessages
	}
	out := []string{}
	for _, raw := range messages[start:] {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		role := stringValue(msg, "role", "user")
		switch role {
		case "system", "developer":
			// Claude Code already supplies a very large system prompt. The Python
			// bridge suppresses it in tool mode to keep the tool protocol dominant.
			continue
		case "assistant":
			if text := ExtractContentText(msg["content"]); text != "" {
				if shouldSkipAssistantHistoryText(text) {
					continue
				}
				out = append(out, "Assistant: "+clipText(text, 1200))
			} else if calls := renderAssistantToolCalls(msg["tool_calls"]); calls != "" {
				out = append(out, "Assistant: "+calls)
			}
		case "tool":
			content := ExtractContentText(msg["content"])
			id := stringValue(msg, "tool_call_id", "")
			out = append(out, renderToolResult(id, content))
		default:
			text := ExtractContentText(msg["content"])
			if text != "" {
				out = append(out, "Human: "+clipText(text, maxInt(1800, 40000-len(strings.Join(out, "\n\n")))))
			}
		}
	}
	return out
}

var toolAvailabilityPollutionRe = regexp.MustCompile(`(?i)\bTool\s+[A-Za-z0-9_.:-]+\s+does\s+not\s+exists?\.?`)

func shouldSkipAssistantHistoryText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}
	lowered := strings.ToLower(trimmed)
	if isToolAvailabilityPollutionOnly(trimmed) {
		return true
	}
	for _, prefix := range []string{
		"upstream stopped after a tool result with narration only",
		"upstream could not produce a valid next client tool call",
		"upstream produced invalid tool-availability text",
		"no recoverable assistant content was produced",
	} {
		if strings.HasPrefix(lowered, prefix) {
			return true
		}
	}
	return false
}

func isToolAvailabilityPollutionOnly(text string) bool {
	if !toolAvailabilityPollutionRe.MatchString(text) {
		return false
	}
	cleaned := toolAvailabilityPollutionRe.ReplaceAllString(text, "")
	cleaned = strings.Trim(cleaned, " \t\r\n.。!！?？;；,:，、")
	return cleaned == ""
}

func firstUserTaskLine(messages []any) string {
	for _, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok || stringValue(msg, "role", "") != "user" {
			continue
		}
		text := extractUserTextOnly(msg["content"])
		if strings.TrimSpace(text) != "" {
			return "Human (ORIGINAL TASK): " + clipText(text, 800)
		}
	}
	return ""
}

func historyContainsOriginal(history []string, original string) bool {
	if original == "" {
		return true
	}
	prefix := clipText(strings.TrimPrefix(original, "Human (ORIGINAL TASK): "), 60)
	for _, line := range history {
		if strings.Contains(line, prefix) {
			return true
		}
	}
	return false
}

func buildToolResultFollowupNotice(messages []any) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg, ok := messages[i].(map[string]any)
		if !ok {
			continue
		}
		role := stringValue(msg, "role", "")
		if role == "tool" || messageContainsToolResult(msg) {
			lines := []string{
				"[STATE NOTICE: MUST OBEY]",
				"The latest client message is a tool result, not a new user request.",
				"Use that result to continue from the current state or finish the task.",
				"All listed action names are still valid client-side QNML actions. Do not check native tool availability; emit QNML instead of availability-error prose.",
				"For ordered or numbered workflows, continue with the next unmet step from the latest confirmed tool result; do not skip ahead or substitute a later-step tool.",
				"If the latest result reports a successful Write/Edit/NotebookEdit, do NOT repeat the exact same write/edit payload for the same target.",
				"If the original task requires final checks or verification, the final answer is allowed only when recent tool results support that the checks passed; otherwise continue with a verification tool call or answer honestly that it is not verified.",
				"Do NOT restart the original task merely because it appears earlier in the prompt.",
			}
			if notice := buildRepeatedToolCallNotice(messages); notice != "" {
				lines = append(lines, notice)
			}
			return strings.Join(lines, "\n")
		}
		if role == "user" && strings.TrimSpace(extractUserTextOnly(msg["content"])) != "" {
			return ""
		}
		if role == "assistant" {
			return ""
		}
	}
	return ""
}

func messageContainsToolResult(msg map[string]any) bool {
	for _, raw := range anyList(msg["content"]) {
		part, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		t := stringValue(part, "type", "")
		if t == "tool_result" || t == "function_call_output" {
			return true
		}
	}
	return false
}

func latestMessageIsToolResult(messages []any) bool {
	for idx := len(messages) - 1; idx >= 0; idx-- {
		msg, ok := messages[idx].(map[string]any)
		if !ok {
			continue
		}
		role := stringValue(msg, "role", "")
		if role == "tool" || messageContainsToolResult(msg) {
			return true
		}
		if role == "user" || role == "assistant" || role == "system" || role == "developer" {
			if strings.TrimSpace(ExtractContentText(msg["content"])) != "" || len(anyList(msg["tool_calls"])) > 0 {
				return false
			}
		}
	}
	return false
}

func ExtractContentText(content any) string {
	switch v := content.(type) {
	case nil:
		return ""
	case string:
		return CompactSystemReminders(v)
	case []any:
		var parts []string
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			switch stringValue(m, "type", "") {
			case "text", "input_text", "output_text":
				if text := CompactSystemReminders(stringValue(m, "text", "")); text != "" {
					parts = append(parts, text)
				}
			case "image_url", "input_image":
				if text := renderAttachmentPlaceholder(m, true); text != "" {
					parts = append(parts, text)
				}
			case "input_file", "file":
				if text := renderAttachmentPlaceholder(m, false); text != "" {
					parts = append(parts, text)
				}
			case "tool_use":
				if rendered := renderQNMLToolCall(qwenSafeToolName(stringValue(m, "name", "")), mapValue(m["input"])); rendered != "" {
					parts = append(parts, rendered)
				}
			case "tool_result", "function_call_output":
				parts = append(parts, renderToolResult(firstString(m["tool_use_id"], m["call_id"], m["id"]), contentToText(m["content"])))
			}
		}
		return strings.Join(parts, "\n")
	default:
		raw, _ := json.Marshal(v)
		return string(raw)
	}
}

func NormalizeTools(value any) []map[string]any {
	tools := []map[string]any{}
	for _, raw := range anyList(value) {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if stringValue(m, "type", "") == "function" {
			if fn, ok := m["function"].(map[string]any); ok {
				tools = append(tools, fn)
				continue
			}
		}
		if stringValue(m, "name", "") != "" {
			tools = append(tools, m)
		}
	}
	return tools
}

func BuildToolInstructions(tools []map[string]any) string {
	profile := detectToolProfile(tools)
	prefix := buildCommonToolInstructionPrefix(profile)
	names := []string{}
	schemas := []string{}
	originalTools := sortToolsForPrompt(tools, profile)
	sortedTools := qwenSafePromptTools(originalTools)
	for _, tool := range sortedTools {
		name := stringValue(tool, "name", "")
		if name == "" {
			continue
		}
		names = append(names, name)
		params := tool["parameters"]
		if params == nil {
			params = tool["input_schema"]
		}
		schemas = append(schemas, fmt.Sprintf(
			"Action name: %s\nDescription: %s\nParameters: %s",
			name,
			obfuscateBareToolNames(trim(stringValue(tool, "description", ""), 120), originalTools),
			summarizeToolParameters(params),
		))
	}
	blocks := []string{
		obfuscateBareToolNames(strings.Join(prefix, "\n"), originalTools),
		obfuscateBareToolNames(buildProfileToolBlock(profile, sortedTools), originalTools),
		buildQNMLToolInstructions(sortedTools, names, schemas),
	}
	return strings.Join(blocks, "\n\n")
}

func qwenSafePromptTools(tools []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		name := stringValue(tool, "name", "")
		if name == "" {
			continue
		}
		clone := map[string]any{}
		for key, value := range tool {
			clone[key] = value
		}
		clone["name"] = qwenSafeToolName(name)
		out = append(out, clone)
	}
	return out
}

func qwenSafeToolName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	switch name {
	case "Read":
		return "fs_open_file"
	case "Write":
		return "fs_put_file"
	case "Edit":
		return "fs_patch_file"
	case "Bash":
		return "shell_run"
	case "Grep":
		return "text_search"
	case "Glob":
		return "path_find"
	case "NotebookEdit":
		return "notebook_patch"
	case "WebFetch":
		return "http_get_url"
	case "WebSearch":
		return "web_query"
	}
	if qwenSafeAliasIsCanonical(name) || strings.HasPrefix(name, "u_") {
		return name
	}
	return "u_" + name
}

func qwenSafeAliasIsCanonical(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "fs_open_file", "fs_put_file", "fs_patch_file", "shell_run", "text_search", "path_find", "notebook_patch", "http_get_url", "web_query":
		return true
	default:
		return false
	}
}

func obfuscateBareToolNames(text string, tools []map[string]any) string {
	if text == "" || len(tools) == 0 {
		return text
	}
	type replacement struct {
		raw  string
		safe string
	}
	replacements := []replacement{}
	seen := map[string]bool{}
	for _, tool := range tools {
		raw := stringValue(tool, "name", "")
		if raw == "" || seen[raw] {
			continue
		}
		safe := qwenSafeToolName(raw)
		if safe == "" || safe == raw {
			continue
		}
		seen[raw] = true
		replacements = append(replacements, replacement{raw: raw, safe: safe})
	}
	sort.SliceStable(replacements, func(i, j int) bool {
		return len(replacements[i].raw) > len(replacements[j].raw)
	})
	out := text
	for _, item := range replacements {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(item.raw) + `\b`)
		out = re.ReplaceAllString(out, item.safe)
	}
	return out
}

func buildQNMLToolInstructions(tools []map[string]any, names []string, schemas []string) string {
	available := strings.Join(names, ", ")
	schemaBlock := ""
	if len(schemas) > 0 {
		schemaBlock = "You have access to these tools:\n\n" + strings.Join(schemas, "\n\n") + "\n\n"
	}
	exampleTools := selectExampleTools(tools)
	exampleSingle := renderQNMLToolCall("TOOL_NAME", map[string]any{"ARG": "value"})
	exampleMulti := renderQNMLToolCall("TOOL_NAME", map[string]any{"ARG": "first value"})
	if len(exampleTools) >= 1 {
		exampleSingle = renderQNMLToolCall(stringValue(exampleTools[0], "name", "TOOL_NAME"), exampleToolParams(exampleTools[0]))
		exampleMulti = exampleSingle
	}
	if len(exampleTools) >= 2 {
		exampleMulti = exampleSingle + "\n\n" + renderQNMLToolCall(stringValue(exampleTools[1], "name", "TOOL_NAME_2"), exampleToolParams(exampleTools[1]))
	}
	return fmt.Sprintf(`=== QNML TOOL CALL PROTOCOL ===
%sQNML blocks are client-parsed text markers, not native function calls.
These action names are valid client-side tools even if the upstream model has no native tool registry.
Available action names: %s

FORMAT:
<|QNML|tool_calls>
  <|QNML|invoke name="TOOL_NAME">
    <|QNML|parameter name="ARG"><![CDATA[value]]></|QNML|parameter>
  </|QNML|invoke>
</|QNML|tool_calls>

RULES:
1) If calling tools, provide a parseable <|QNML|tool_calls> block. If no tool is needed, answer normally.
2) Put one or more <|QNML|invoke> nodes under the wrapper. Use exact tool and parameter names from the schema.
3) Strings use <![CDATA[...]]>; objects use nested XML elements; arrays repeat <item>; numbers/bools/null stay plain text.
4) Never emit empty required parameters, especially shell commands. If required info is unknown, ask normally.
5) After [Tool Result], continue with more QNML calls only if needed; otherwise answer normally.
6) Prefer direct project tools. Use Agent/task/scheduling/control tools only when clearly necessary or explicitly requested.
7) Shell-capable actions run in the client tool environment. Prefer workspace-relative paths and avoid unverified raw Windows paths in shell commands.
8) Do not claim an Available action name is unavailable; use QNML when you choose to call that action.
9) Path parameters such as path/file_path/filename must contain only the path string, with no prose, round labels, status text, or QNML/XML fragments.

CORRECT EXAMPLES:

%s

<|QNML|tool_calls>
%s
</|QNML|tool_calls>

Remember: the preferred tool-call form is <|QNML|tool_calls>...</|QNML|tool_calls>.
=== END QNML TOOL INSTRUCTIONS ===`, schemaBlock, available, exampleSingle, indentQNMLInvokes(exampleMulti))
}

func selectExampleTools(tools []map[string]any) []map[string]any {
	out := []map[string]any{}
	for _, tool := range tools {
		name := stringValue(tool, "name", "")
		if name == "" || isControlToolName(name) {
			continue
		}
		out = append(out, tool)
		if len(out) >= 2 {
			return out
		}
	}
	if len(out) == 0 {
		for _, tool := range tools {
			if stringValue(tool, "name", "") != "" {
				out = append(out, tool)
				if len(out) >= 2 {
					break
				}
			}
		}
	}
	return out
}

func indentQNMLInvokes(block string) string {
	lines := strings.Split(strings.TrimSpace(block), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "<|QNML|tool_calls>" || line == "</|QNML|tool_calls>" {
			continue
		}
		out = append(out, "  "+line)
	}
	return strings.Join(out, "\n")
}

func summarizeToolParameters(params any) string {
	schema := mapValue(params)
	if len(schema) == 0 {
		raw, _ := json.Marshal(params)
		return trim(string(raw), 240)
	}
	props, _ := schema["properties"].(map[string]any)
	required := map[string]bool{}
	for _, item := range anyList(schema["required"]) {
		name, _ := item.(string)
		if name != "" {
			required[name] = true
		}
	}
	if len(props) == 0 {
		if t := stringValue(schema, "type", ""); t != "" {
			return t
		}
		raw, _ := json.Marshal(params)
		return trim(string(raw), 240)
	}
	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		spec, _ := props[key].(map[string]any)
		typeSummary := summarizeSchemaType(spec)
		if required[key] {
			parts = append(parts, key+"!:"+typeSummary)
		} else {
			parts = append(parts, key+":"+typeSummary)
		}
	}
	return strings.Join(parts, ", ")
}

func summarizeSchemaType(spec map[string]any) string {
	if len(spec) == 0 {
		return "any"
	}
	if enumValues := anyList(spec["enum"]); len(enumValues) > 0 {
		items := make([]string, 0, len(enumValues))
		for _, item := range enumValues {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				items = append(items, s)
			} else {
				items = append(items, fmt.Sprint(item))
			}
		}
		return "enum(" + strings.Join(items, "|") + ")"
	}
	typ := stringValue(spec, "type", "")
	switch typ {
	case "array":
		items := mapValue(spec["items"])
		itemType := summarizeSchemaType(items)
		if itemType == "" {
			itemType = "any"
		}
		return "array<" + itemType + ">"
	case "object":
		props, _ := spec["properties"].(map[string]any)
		if len(props) == 0 {
			return "object"
		}
		keys := make([]string, 0, len(props))
		for key := range props {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if len(keys) > 3 {
			keys = keys[:3]
		}
		return "object{" + strings.Join(keys, ",") + "}"
	case "":
		return "any"
	default:
		return typ
	}
}

func toolPromptPriority(toolName string) (int, string) {
	key := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(toolName), "")
	preferred := map[string]int{
		"read": 0, "readfile": 0, "readfiletool": 0,
		"bash": 1, "terminal": 1, "powershell": 1, "shell": 1, "executecommand": 1, "runcommand": 1,
		"glob": 2, "grep": 3, "search": 3, "searchfiles": 3,
		"write": 4, "writefile": 4,
		"edit": 5, "editfile": 5,
		"webfetch": 6, "websearch": 7,
		"skillslist": 30, "skillview": 31, "skillmanage": 32,
		"patch": 60, "process": 70,
	}
	if v, ok := preferred[key]; ok {
		return v, toolName
	}
	if isControlToolName(toolName) {
		return 90, toolName
	}
	return 20, toolName
}

func buildFewShotBlock(tools []map[string]any) string {
	selected := pickFewShotTools(tools, 2)
	if len(selected) < 2 {
		return ""
	}
	calls := []string{}
	for _, tool := range selected {
		name := stringValue(tool, "name", "")
		if name == "" {
			continue
		}
		calls = append(calls, renderQNMLToolCall(name, exampleToolParams(tool)))
	}
	if len(calls) == 0 {
		return ""
	}
	return "Human: [FEW-SHOT WARM-UP] Demonstrate that you can emit multiple action markers from different categories.\n\n" +
		"Assistant: Understood. Here are example markers across action categories:\n\n" +
		strings.Join(calls, "\n\n")
}

func pickFewShotTools(tools []map[string]any, maxThirdParty int) []map[string]any {
	safe := []map[string]any{}
	for _, tool := range tools {
		name := stringValue(tool, "name", "")
		if !isControlToolName(name) && !isComplexExampleToolName(name) {
			safe = append(safe, tool)
		}
	}
	selected := []map[string]any{}
	for _, wants := range [][]string{
		{"read", "readfile", "fs_open_file"},
		{"bash", "terminal", "powershell", "shell", "shell_run", "executecommand", "runcommand"},
	} {
		for _, tool := range safe {
			name := stringValue(tool, "name", "")
			if normalizedNameIn(name, wants) {
				selected = append(selected, tool)
				break
			}
		}
	}
	namespaces := map[string][]map[string]any{}
	for _, tool := range safe {
		name := stringValue(tool, "name", "")
		if isCoreToolName(name) || alreadySelected(selected, name) {
			continue
		}
		ns := toolNamespace(name)
		namespaces[ns] = append(namespaces[ns], tool)
	}
	type group struct {
		ns    string
		items []map[string]any
	}
	groups := []group{}
	for ns, items := range namespaces {
		groups = append(groups, group{ns: ns, items: items})
	}
	sort.Slice(groups, func(i, j int) bool { return len(groups[i].items) > len(groups[j].items) })
	for _, group := range groups {
		if len(selected) >= 2+maxThirdParty {
			break
		}
		best := group.items[0]
		for _, item := range group.items[1:] {
			if len(stringValue(item, "description", "")) > len(stringValue(best, "description", "")) {
				best = item
			}
		}
		selected = append(selected, best)
	}
	if len(selected) == 0 && len(safe) > 0 {
		selected = append(selected, safe[0])
	}
	return selected
}

func isCoreToolName(name string) bool {
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, "u_") && len(name) > 2 {
		name = strings.TrimPrefix(name, "u_")
	}
	key := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(name), "")
	switch key {
	case "read", "readfile", "readfiletool", "write", "writefile", "bash", "executecommand",
		"runcommand", "terminal", "powershell", "shell", "listdir", "listdirectory", "grep", "glob",
		"search", "searchfiles", "edit", "editfile", "fsopenfile", "fsputfile", "fspatchfile",
		"shellrun", "textsearch", "pathfind", "httpgeturl", "webquery":
		return true
	default:
		return false
	}
}

func normalizedNameIn(name string, wants []string) bool {
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, "u_") && len(name) > 2 {
		name = strings.TrimPrefix(name, "u_")
	}
	key := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(name), "")
	nameCap := classifyToolCapability(name)
	for _, want := range wants {
		wantKey := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(want), "")
		if key == wantKey {
			return true
		}
		if nameCap != "" && nameCap == classifyToolCapability(want) {
			return true
		}
	}
	return false
}

func isComplexExampleToolName(name string) bool {
	key := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(name), "")
	switch key {
	case "patch", "applypatch", "process":
		return true
	default:
		return false
	}
}

func alreadySelected(selected []map[string]any, name string) bool {
	for _, tool := range selected {
		if strings.EqualFold(stringValue(tool, "name", ""), name) {
			return true
		}
	}
	return false
}

func toolNamespace(name string) string {
	if strings.HasPrefix(name, "mcp__") {
		parts := strings.Split(name, "__")
		if len(parts) >= 2 {
			return parts[0] + "__" + parts[1]
		}
	}
	if idx := strings.Index(name, "__"); idx > 0 {
		return name[:idx]
	}
	parts := strings.Split(name, "_")
	if len(parts) >= 3 {
		return parts[0]
	}
	return name
}

func exampleToolParams(tool map[string]any) map[string]any {
	name := stringValue(tool, "name", "")
	key := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(name), "")
	if fromSchema := exampleParamsFromSchema(tool); len(fromSchema) > 0 {
		return fromSchema
	}
	switch key {
	case "read", "readfile", "fsopenfile":
		return map[string]any{"file_path": "src/index.ts"}
	case "write", "writefile", "fsputfile":
		return map[string]any{"file_path": "output.txt", "content": "..."}
	case "bash", "executecommand", "runcommand", "terminal", "powershell", "shell", "shellrun":
		return map[string]any{"command": "ls -la"}
	case "glob", "pathfind":
		return map[string]any{"pattern": "**/*.go"}
	case "grep", "search", "searchfiles", "textsearch":
		return map[string]any{"pattern": "TODO"}
	case "edit", "editfile", "fspatchfile":
		return map[string]any{"file_path": "src/main.ts", "old_string": "old", "new_string": "new"}
	}
	params := firstNonNil(tool["parameters"], tool["input_schema"])
	props, _ := mapValue(params)["properties"].(map[string]any)
	out := map[string]any{}
	for key, rawSpec := range props {
		spec, _ := rawSpec.(map[string]any)
		switch stringValue(spec, "type", "string") {
		case "boolean":
			out[key] = true
		case "number", "integer":
			out[key] = 1
		case "array":
			out[key] = []any{}
		case "object":
			out[key] = map[string]any{}
		default:
			out[key] = "value"
		}
		if len(out) >= 2 {
			break
		}
	}
	if len(out) == 0 {
		out["input"] = "value"
	}
	return out
}

func exampleParamsFromSchema(tool map[string]any) map[string]any {
	params := firstNonNil(tool["parameters"], tool["input_schema"])
	schema := mapValue(params)
	props, _ := schema["properties"].(map[string]any)
	if len(props) == 0 {
		return nil
	}
	keys := []string{}
	seen := map[string]bool{}
	for _, item := range anyList(schema["required"]) {
		name, _ := item.(string)
		if name != "" && props[name] != nil && !seen[name] {
			keys = append(keys, name)
			seen[name] = true
		}
	}
	optional := make([]string, 0, len(props))
	for key := range props {
		if !seen[key] {
			optional = append(optional, key)
		}
	}
	sort.Strings(optional)
	keys = append(keys, optional...)
	out := map[string]any{}
	for _, key := range keys {
		spec, _ := props[key].(map[string]any)
		out[key] = exampleSchemaValue(key, spec)
		if len(out) >= 4 {
			break
		}
	}
	return out
}

func exampleSchemaValue(name string, spec map[string]any) any {
	key := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(name), "")
	switch key {
	case "command", "cmd", "script", "shellcommand":
		return "ls -la"
	case "filepath", "path", "filename", "file":
		return "src/index.ts"
	case "content", "text", "body":
		return "..."
	case "pattern", "query", "search":
		return "TODO"
	case "oldstring", "oldtext", "old":
		return "old"
	case "newstring", "newtext", "new":
		return "new"
	}
	switch stringValue(spec, "type", "string") {
	case "boolean":
		return true
	case "number", "integer":
		return 1
	case "array":
		return []any{}
	case "object":
		return map[string]any{}
	default:
		return "value"
	}
}

func renderAssistantToolCalls(raw any) string {
	var calls []string
	for _, item := range anyList(raw) {
		call, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, rawArgs := toolCallNameAndArgs(call)
		args := map[string]any{}
		if rawArgs != "" {
			_ = json.Unmarshal([]byte(rawArgs), &args)
		}
		if name != "" {
			calls = append(calls, renderQNMLToolCall(qwenSafeToolName(name), args))
		}
	}
	return strings.Join(calls, "\n\n")
}

func toolCallNameAndArgs(call map[string]any) (string, string) {
	fn, _ := call["function"].(map[string]any)
	name := firstNonEmpty(anyString(fn["name"], ""), anyString(call["name"], ""))
	args := firstNonEmpty(anyString(fn["arguments"], ""), structuredArgString(call["input"]))
	return name, args
}

func structuredArgString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(raw)
	}
}

func renderQNMLToolCall(name string, input map[string]any) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	if input == nil {
		input = map[string]any{}
	}
	var b strings.Builder
	b.WriteString("<|QNML|tool_calls>\n")
	b.WriteString("  <|QNML|invoke name=\"")
	b.WriteString(xmlEscape(name))
	b.WriteString("\">\n")
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		b.WriteString("    <|QNML|parameter name=\"")
		b.WriteString(xmlEscape(key))
		b.WriteString("\">")
		b.WriteString(renderQNMLValue(input[key]))
		b.WriteString("</|QNML|parameter>\n")
	}
	b.WriteString("  </|QNML|invoke>\n")
	b.WriteString("</|QNML|tool_calls>")
	return b.String()
}

func renderQNMLValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case string:
		return "<![CDATA[" + strings.ReplaceAll(v, "]]>", "]]]]><![CDATA[>") + "]]>"
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		raw, _ := json.Marshal(v)
		return string(raw)
	}
}

func renderToolResult(id, content string) string {
	idPart := ""
	if strings.TrimSpace(id) != "" {
		idPart = " id=" + strings.TrimSpace(id)
	}
	return "[Tool Result" + idPart + "]\n" + clipText(content, 6000) + "\n[/Tool Result]"
}

func extractUserTextOnly(content any) string {
	switch v := content.(type) {
	case string:
		return CompactSystemReminders(v)
	case []any:
		parts := []string{}
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if stringValue(m, "type", "") == "text" {
				parts = append(parts, CompactSystemReminders(stringValue(m, "text", "")))
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func contentToText(content any) string {
	switch v := content.(type) {
	case nil:
		return ""
	case string:
		return v
	case []any:
		parts := []string{}
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if text := stringValue(m, "text", ""); text != "" {
					parts = append(parts, text)
				}
			} else if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "\n")
	default:
		raw, _ := json.Marshal(v)
		return string(raw)
	}
}

func mapValue(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func firstString(values ...any) string {
	for _, value := range values {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func anyString(v any, fallback string) string {
	switch x := v.(type) {
	case string:
		if strings.TrimSpace(x) != "" {
			return strings.TrimSpace(x)
		}
	}
	return fallback
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clipText(text string, limit int) string {
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return strings.TrimSpace(text[:limit]) + "...[truncated]"
}

func xmlEscape(text string) string {
	replacer := strings.NewReplacer("&", "&amp;", `"`, "&quot;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(text)
}

func isControlToolName(name string) bool {
	key := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(name), "")
	switch key {
	case "agent", "askuserquestion", "croncreate", "crondelete", "cronlist",
		"enterplanmode", "exitplanmode", "enterworktree", "exitworktree",
		"monitor", "pushnotification", "schedulewakeup", "taskcreate",
		"taskdelete", "taskget", "tasklist", "taskoutput", "taskstop", "taskupdate",
		"delegatetask", "delegate", "subagent", "todo":
		return true
	default:
		return false
	}
}

func CompactSystemReminders(text string) string {
	re := regexp.MustCompile(`(?is)<system-reminder>(.*?)</system-reminder>`)
	return re.ReplaceAllStringFunc(text, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) < 2 {
			return "[system-reminder]"
		}
		first := strings.TrimSpace(strings.SplitN(sub[1], "\n", 2)[0])
		if first == "" {
			return "[system-reminder]"
		}
		return "[system-reminder: " + trim(first, 80) + "...]"
	})
}

func stringValue(m map[string]any, key, fallback string) string {
	if m == nil {
		return fallback
	}
	if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func boolValue(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	if s, ok := v.(string); ok {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "1", "true", "yes", "on":
			return true
		}
	}
	return false
}

func coerceBool(v any) *bool {
	switch value := v.(type) {
	case bool:
		return &value
	case string:
		lowered := strings.ToLower(strings.TrimSpace(value))
		switch lowered {
		case "1", "true", "yes", "on", "enable", "enabled", "auto":
			b := true
			return &b
		case "0", "false", "no", "off", "disable", "disabled", "none":
			b := false
			return &b
		}
	case float64:
		b := value != 0
		return &b
	case int:
		b := value != 0
		return &b
	}
	return nil
}

func anyList(v any) []any {
	if list, ok := v.([]any); ok {
		return list
	}
	return nil
}

func trim(text string, limit int) string {
	if limit <= 0 || len([]rune(text)) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit]) + "..."
}
