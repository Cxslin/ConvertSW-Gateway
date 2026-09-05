package services

import "strings"

type IncrementalTextStreamer struct {
	builder strings.Builder
	sent    int
}

func (s *IncrementalTextStreamer) Append(text string) string {
	s.builder.WriteString(text)
	all := s.builder.String()
	if s.sent >= len(all) {
		return ""
	}
	delta := all[s.sent:]
	s.sent = len(all)
	return delta
}

func (s *IncrementalTextStreamer) Text() string {
	return s.builder.String()
}

func (s *IncrementalTextStreamer) Reset() {
	s.builder.Reset()
	s.sent = 0
}
