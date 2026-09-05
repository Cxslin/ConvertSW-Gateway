package toolcall

import "strings"

// StreamState accumulates streamed text and extracts tool calls from the final
// assembled answer.
type StreamState struct {
	builder strings.Builder
}

func (s *StreamState) AddDelta(text string) {
	if text != "" {
		s.builder.WriteString(text)
	}
}

func (s *StreamState) Text() string {
	return s.builder.String()
}

func (s *StreamState) Parse(tools []map[string]any) []ParsedToolCall {
	return ParseToolCalls(s.Text(), tools)
}

func (s *StreamState) Reset() {
	s.builder.Reset()
}
