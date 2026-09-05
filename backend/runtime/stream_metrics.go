package runtime

import "time"

type StreamMetrics struct {
	StartedAt      time.Time
	FirstTokenAt   time.Time
	CompletedAt    time.Time
	Chunks         int
	ContentBytes   int
	ReasoningBytes int
}

func (m *StreamMetrics) Start(now time.Time) {
	m.StartedAt = now
}

func (m *StreamMetrics) Observe(content, reasoning string, now time.Time) {
	if m.StartedAt.IsZero() {
		m.Start(now)
	}
	if m.FirstTokenAt.IsZero() && (content != "" || reasoning != "") {
		m.FirstTokenAt = now
	}
	m.Chunks++
	m.ContentBytes += len(content)
	m.ReasoningBytes += len(reasoning)
}

func (m *StreamMetrics) Complete(now time.Time) {
	m.CompletedAt = now
}
