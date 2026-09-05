package toolcall

import (
	"regexp"
	"strings"
)

type SieveEvent struct {
	Type  string
	Text  string
	Calls []ParsedToolCall
}

// ToolSieve buffers answer text and detects textual tool calls as soon as a
// complete QNML/XML/JSON directive is available.
type ToolSieve struct {
	tools       []map[string]any
	toolNames   map[string]bool
	pending     string
	capture     string
	capturing   bool
	inFence     bool
	fenceChar   byte
	fenceLength int
}

func NewToolSieve(tools []map[string]any) *ToolSieve {
	names := map[string]bool{}
	for _, tool := range tools {
		if name := stringValue(tool, "name", ""); name != "" {
			names[name] = true
		}
	}
	return &ToolSieve{tools: tools, toolNames: names}
}

func (s *ToolSieve) ProcessChunk(chunk string) []SieveEvent {
	if s == nil || chunk == "" {
		return nil
	}
	s.pending += chunk
	events := []SieveEvent{}
	if s.capturing {
		s.capture += s.pending
		s.pending = ""
		prefix, calls, suffix, ready := s.consumeToolCapture(false)
		if ready && len(calls) > 0 {
			if prefix != "" {
				events = append(events, SieveEvent{Type: "content", Text: prefix})
			}
			events = append(events, SieveEvent{Type: "tool_calls", Calls: calls})
			s.pending = suffix
			s.capture = ""
			s.capturing = false
		}
		return events
	}
	start := s.findToolStart(s.pending)
	if start >= 0 {
		prefix := s.pending[:start]
		if prefix != "" {
			events = append(events, SieveEvent{Type: "content", Text: prefix})
			s.advanceMarkdownFence(prefix)
		}
		s.capture = s.pending[start:]
		s.pending = ""
		s.capturing = true
		prefix, calls, suffix, ready := s.consumeToolCapture(false)
		if ready && len(calls) > 0 {
			if prefix != "" {
				events = append(events, SieveEvent{Type: "content", Text: prefix})
			}
			events = append(events, SieveEvent{Type: "tool_calls", Calls: calls})
			s.pending = suffix
			s.capture = ""
			s.capturing = false
		}
		return events
	}
	safe, hold := s.splitSafeContent(s.pending)
	if safe != "" {
		events = append(events, SieveEvent{Type: "content", Text: safe})
		s.advanceMarkdownFence(safe)
	}
	s.pending = hold
	return events
}

func (s *ToolSieve) Flush() []SieveEvent {
	if s == nil {
		return nil
	}
	events := []SieveEvent{}
	if s.capturing && s.capture != "" {
		prefix, calls, suffix, ready := s.consumeToolCapture(true)
		if ready && len(calls) > 0 {
			if prefix != "" {
				events = append(events, SieveEvent{Type: "content", Text: prefix})
			}
			events = append(events, SieveEvent{Type: "tool_calls", Calls: calls})
			if suffix != "" {
				events = append(events, SieveEvent{Type: "content", Text: suffix})
			}
		}
		s.capture = ""
		s.capturing = false
	}
	if s.pending != "" {
		events = append(events, SieveEvent{Type: "content", Text: s.pending})
		s.advanceMarkdownFence(s.pending)
		s.pending = ""
	}
	return events
}

func (s *ToolSieve) consumeToolCapture(force bool) (string, []ParsedToolCall, string, bool) {
	if s.capture == "" {
		return "", nil, "", false
	}
	if !force && hasStreamingToolEnvelopeStart(s.capture) && !looksStructurallyClosed(s.capture) {
		return "", nil, "", false
	}
	if !force && functionNameRe.MatchString(s.capture) && functionArgumentsRe.MatchString(s.capture) && !looksStructurallyClosed(s.capture) {
		return "", nil, "", false
	}
	calls := ParseToolCalls(s.capture, s.tools)
	if len(calls) == 0 {
		return "", nil, "", false
	}
	prefix := ""
	if first := firstToolMarkerIndex(s.capture); first > 0 {
		prefix = strings.TrimSpace(s.capture[:first])
	}
	return prefix, calls, "", true
}

func (s *ToolSieve) findToolStart(text string) int {
	if text == "" {
		return -1
	}
	inFence := s.inFence
	fenceChar := s.fenceChar
	fenceLength := s.fenceLength
	lineStart := 0
	for i := 0; i < len(text); {
		ch := text[i]
		if ch == '\n' {
			lineStart = i + 1
			i++
			continue
		}
		atLineIndent := strings.Trim(text[lineStart:i], " \t\r") == ""
		if atLineIndent && (strings.HasPrefix(text[i:], "```") || strings.HasPrefix(text[i:], "~~~")) {
			runChar := text[i]
			runLen := 0
			for i+runLen < len(text) && text[i+runLen] == runChar {
				runLen++
			}
			if runLen >= 3 {
				if !inFence {
					inFence = true
					fenceChar = runChar
					fenceLength = runLen
				} else if runChar == fenceChar && runLen >= fenceLength {
					lineEnd := strings.IndexByte(text[i:], '\n')
					tail := text[i+runLen:]
					if lineEnd >= 0 {
						tail = text[i+runLen : i+lineEnd]
					}
					if strings.TrimSpace(tail) == "" {
						inFence = false
						fenceChar = 0
						fenceLength = 0
					}
				}
				next := strings.IndexByte(text[i:], '\n')
				if next < 0 {
					return -1
				}
				i += next + 1
				lineStart = i
				continue
			}
		}
		if inFence {
			i++
			continue
		}
		if ch == '`' {
			runLen := 0
			for i+runLen < len(text) && text[i+runLen] == '`' {
				runLen++
			}
			tail := text[i+runLen:]
			if markerMatchStart(tail) == 0 {
				close := strings.Index(tail, strings.Repeat("`", runLen))
				newline := strings.IndexByte(tail, '\n')
				if close < 0 || (newline >= 0 && newline < close) {
					return -1
				}
			}
			i += max(1, runLen)
			continue
		}
		if markerMatchStart(text[i:]) == 0 {
			return i
		}
		i++
	}
	return -1
}

func (s *ToolSieve) advanceMarkdownFence(text string) {
	if text == "" {
		return
	}
	for _, line := range strings.SplitAfter(text, "\n") {
		stripped := strings.TrimLeft(line, " \t\r")
		if !strings.HasPrefix(stripped, "```") && !strings.HasPrefix(stripped, "~~~") {
			continue
		}
		runChar := stripped[0]
		runLen := 0
		for runLen < len(stripped) && stripped[runLen] == runChar {
			runLen++
		}
		if runLen < 3 {
			continue
		}
		if !s.inFence {
			s.inFence = true
			s.fenceChar = runChar
			s.fenceLength = runLen
			continue
		}
		if runChar == s.fenceChar && runLen >= s.fenceLength && strings.TrimSpace(stripped[runLen:]) == "" {
			s.inFence = false
			s.fenceChar = 0
			s.fenceLength = 0
		}
	}
}

func (s *ToolSieve) splitSafeContent(text string) (string, string) {
	if holdStart := inlineToolExampleHoldStart(text); holdStart >= 0 {
		return text[:holdStart], text[holdStart:]
	}
	const holdLen = 64
	if len(text) <= holdLen {
		return "", text
	}
	return text[:len(text)-holdLen], text[len(text)-holdLen:]
}

func inlineToolExampleHoldStart(text string) int {
	for i := 0; i < len(text); {
		pos := strings.IndexByte(text[i:], '`')
		if pos < 0 {
			return -1
		}
		pos += i
		runLen := 0
		for pos+runLen < len(text) && text[pos+runLen] == '`' {
			runLen++
		}
		tailStart := pos + runLen
		if markerMatchStart(text[tailStart:]) == 0 {
			tail := text[tailStart:]
			close := strings.Index(tail, strings.Repeat("`", runLen))
			newline := strings.IndexByte(tail, '\n')
			if close < 0 || (newline >= 0 && newline < close) {
				return pos
			}
		}
		i = tailStart
	}
	return -1
}

var (
	markerStartPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?is)^<\s*(?:\|\s*)?QNML(?:\s*\|\s*|\s+)?(?:tool_calls|tool-calls|toolcalls|invoke|parameter)?`),
		regexp.MustCompile(`(?is)^＜\s*(?:\|\s*)?QNML`),
		regexp.MustCompile(`(?is)^<\s*tool_calls\b`),
		regexp.MustCompile(`(?is)^<\s*invoke\b`),
		regexp.MustCompile(`(?is)^<\s*tool_call\b`),
		regexp.MustCompile(`(?is)^\{\s*"tool_calls"`),
		regexp.MustCompile(`(?is)^\{\s*"name"\s*:`),
		regexp.MustCompile(`(?is)^##\s*TOOL_CALL##`),
		regexp.MustCompile(`(?is)^function\.name\s*:`),
	}
	functionNameRe      = regexp.MustCompile(`(?is)function\.name\s*:`)
	functionArgumentsRe = regexp.MustCompile(`(?is)function\.arguments\s*:`)
	structuralCloseRe   = regexp.MustCompile(`(?m)\n\s*[\]}]\s*$`)
)

func markerMatchStart(text string) int {
	for _, re := range markerStartPatterns {
		if loc := re.FindStringIndex(text); loc != nil {
			return loc[0]
		}
	}
	return -1
}

func firstToolMarkerIndex(text string) int {
	best := -1
	for i := 0; i < len(text); i++ {
		if markerMatchStart(text[i:]) == 0 {
			if best < 0 || i < best {
				best = i
			}
		}
	}
	return best
}

func looksStructurallyClosed(text string) bool {
	return structuralCloseRe.MatchString(text) || strings.Contains(strings.ToLower(text), "</|qnml|tool_calls>") || strings.Contains(strings.ToLower(text), "</tool_calls>")
}

func hasStreamingToolEnvelopeStart(text string) bool {
	lowered := strings.ToLower(text)
	return strings.Contains(lowered, "<|qnml|tool_calls") ||
		strings.Contains(lowered, "<tool_calls") ||
		strings.Contains(lowered, "<|qnml|invoke") ||
		strings.Contains(lowered, "<invoke")
}
