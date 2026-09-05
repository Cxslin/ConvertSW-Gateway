package upstream

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// Event is a normalized chunk emitted by Qwen's SSE stream.
type Event struct {
	Type          string
	Phase         string
	Content       string
	ReasoningText string
	Status        string
	Extra         map[string]any
	Raw           map[string]any
}

// ConsumeSSE parses server-sent events from r and invokes onEvent for every
// normalized upstream message.
func ConsumeSSE(r io.Reader, onEvent func(Event) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	var block strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			if err := ParseSSEBlock(block.String(), onEvent); err != nil {
				return err
			}
			block.Reset()
			continue
		}
		block.WriteString(line)
		block.WriteByte('\n')
	}
	if strings.TrimSpace(block.String()) != "" {
		if err := ParseSSEBlock(block.String(), onEvent); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// DeepSeekStreamParser tracks the current phase ("thinking" vs "answer")
// across SSE chunks of a DeepSeek chat completion response.
type DeepSeekStreamParser struct {
	currentPhase string
	finished     bool
}

func NewDeepSeekStreamParser(thinkingEnabled bool) *DeepSeekStreamParser {
	phase := "answer"
	if thinkingEnabled {
		phase = "thinking"
	}
	return &DeepSeekStreamParser{
		currentPhase: phase,
		finished:     false,
	}
}

// ParseSSEBlock decodes one SSE block with data: JSON payloads using the parser state.
func (p *DeepSeekStreamParser) ParseSSEBlock(block string, onEvent func(Event) error) error {
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(data), &obj); err != nil {
			continue
		}
		for _, event := range p.Parse(obj) {
			if err := onEvent(event); err != nil {
				return err
			}
		}
	}
	return nil
}

// Parse processes an upstream JSON object and returns any emitted normalized events.
func (p *DeepSeekStreamParser) Parse(obj map[string]any) []Event {
	if p.finished {
		return nil
	}

	// Standard OpenAI format fallback if choices is present
	if _, ok := obj["choices"]; ok {
		return ParseQwenEvent(obj)
	}

	// Check if finished via top-level status
	if st := strings.ToUpper(firstString(obj["status"])); st == "FINISHED" {
		p.finished = true
		return []Event{{Type: "done", Status: "finished", Raw: obj}}
	}

	path := firstString(obj["p"])
	pLower := strings.ToLower(path)

	// Filter out non-content fields (title, session, token_usage, click_behavior, elapsed_secs, etc.)
	if strings.Contains(pLower, "title") || strings.Contains(pLower, "session") ||
		strings.Contains(pLower, "token_usage") || strings.Contains(pLower, "elapsed_secs") ||
		strings.Contains(pLower, "quasi_status") {
		return nil
	}

	// Check if this chunk is setting status to FINISHED
	if strings.HasSuffix(pLower, "status") {
		if val, ok := obj["v"].(string); ok && strings.ToUpper(val) == "FINISHED" {
			p.finished = true
			return []Event{{Type: "done", Status: "finished", Raw: obj}}
		}
	}

	events := make([]Event, 0, 2)

	// Case 1: Fragments array appended or updated (e.g. {"p":"response/fragments","o":"APPEND","v":[...]})
	if arr, ok := obj["v"].([]any); ok && strings.Contains(pLower, "fragments") {
		for _, item := range arr {
			if frag, ok := item.(map[string]any); ok {
				fType := strings.ToUpper(firstString(frag["type"]))
				fContent := firstString(frag["content"])
				if fType == "THINK" {
					p.currentPhase = "thinking"
				} else if fType == "RESPONSE" {
					p.currentPhase = "answer"
				}
				if fContent != "" {
					evt := Event{
						Type:  "delta",
						Phase: p.currentPhase,
						Raw:   obj,
					}
					if p.currentPhase == "thinking" {
						evt.ReasoningText = fContent
						evt.Content = fContent
					} else {
						evt.Content = fContent
					}
					events = append(events, evt)
				}
			}
		}
		if len(events) > 0 {
			return events
		}
	}

	// Case 2: Object with response (initial payload {"v":{"response":{...}}})
	if vObj, ok := obj["v"].(map[string]any); ok {
		if resp, ok := vObj["response"].(map[string]any); ok {
			if st := strings.ToUpper(firstString(resp["status"])); st == "FINISHED" {
				p.finished = true
				return []Event{{Type: "done", Status: "finished", Raw: obj}}
			}
			if frags, ok := resp["fragments"].([]any); ok {
				for _, f := range frags {
					if frag, ok := f.(map[string]any); ok {
						fType := strings.ToUpper(firstString(frag["type"]))
						fContent := firstString(frag["content"])
						if fType == "THINK" {
							p.currentPhase = "thinking"
						} else if fType == "RESPONSE" {
							p.currentPhase = "answer"
						}
						if fContent != "" {
							evt := Event{
								Type:  "delta",
								Phase: p.currentPhase,
								Raw:   obj,
							}
							if p.currentPhase == "thinking" {
								evt.ReasoningText = fContent
								evt.Content = fContent
							} else {
								evt.Content = fContent
							}
							events = append(events, evt)
						}
					}
				}
				if len(events) > 0 {
					return events
				}
			}
		}
	}

	// Case 3: String delta (e.g. {"p":"response/fragments/-1/content","o":"APPEND","v":"..."} or {"v":"..."})
	if strVal, ok := obj["v"].(string); ok && strVal != "" {
		if strings.ToUpper(strVal) == "FINISHED" {
			p.finished = true
			return []Event{{Type: "done", Status: "finished", Raw: obj}}
		}
		if strings.Contains(pLower, "think") {
			p.currentPhase = "thinking"
		} else if strings.Contains(pLower, "response") && !strings.Contains(pLower, "fragments/-1") {
			p.currentPhase = "answer"
		}

		evt := Event{
			Type:  "delta",
			Phase: p.currentPhase,
			Raw:   obj,
		}
		if p.currentPhase == "thinking" {
			evt.ReasoningText = strVal
			evt.Content = strVal
		} else {
			evt.Content = strVal
		}
		return []Event{evt}
	}

	return nil
}

// ParseSSEBlock decodes one SSE block with data: JSON payloads using a default parser.
func ParseSSEBlock(block string, onEvent func(Event) error) error {
	parser := NewDeepSeekStreamParser(false)
	return parser.ParseSSEBlock(block, onEvent)
}

// ParseDeepSeekEvent parses streaming events from DeepSeek Android API.
func ParseDeepSeekEvent(obj map[string]any) []Event {
	parser := NewDeepSeekStreamParser(false)
	return parser.Parse(obj)
}

// ParseQwenEvent normalizes the different text fields Qwen uses across models.
func ParseQwenEvent(obj map[string]any) []Event {
	events := []Event{}
	if choices, ok := obj["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			delta, _ := choice["delta"].(map[string]any)
			phase := firstString(delta["phase"])
			if phase == "" {
				phase = "answer"
			}
			content := firstString(delta["content"])
			extra, _ := delta["extra"].(map[string]any)
			reasoning := extractReasoning(delta, extra)
			if reasoning != "" {
				content = reasoning
				if phase == "answer" {
					phase = "thinking_summary"
				}
			}
			events = append(events, Event{
				Type:          "delta",
				Phase:         phase,
				Content:       content,
				ReasoningText: reasoning,
				Status:        firstString(delta["status"]),
				Extra:         extra,
				Raw:           obj,
			})
			return events
		}
	}
	content := firstString(obj["content"], obj["answer"], obj["text"], obj["delta"])
	reasoning := firstString(obj["reasoning_content"], obj["reasoning"], obj["thinking"])
	status := firstString(obj["status"])
	eventType := firstString(obj["event"], obj["type"], status)
	if content != "" || reasoning != "" || eventType != "" {
		phase := eventType
		if phase == "" {
			phase = "answer"
		}
		if reasoning != "" {
			content = reasoning
			if phase == "answer" {
				phase = "thinking_summary"
			}
		}
		events = append(events, Event{Type: firstNonEmpty(eventType, "delta"), Phase: phase, Content: content, ReasoningText: reasoning, Status: status, Raw: obj})
	}
	if data, ok := obj["data"].(map[string]any); ok {
		events = append(events, ParseQwenEvent(data)...)
	}
	if msg, ok := obj["message"].(map[string]any); ok {
		events = append(events, ParseQwenEvent(msg)...)
	}
	return events
}

func extractReasoning(delta map[string]any, extra map[string]any) string {
	if delta == nil {
		return ""
	}
	values := []any{
		delta["reasoning_content"],
		delta["reasoning"],
		delta["reasoning_text"],
		delta["thinking"],
		delta["thoughts"],
	}
	if extra != nil {
		values = append(values, extra["reasoning_content"], extra["reasoning"], extra["reasoning_text"], extra["thinking"], extra["thoughts"])
	}
	return firstString(values...)
}

func firstString(values ...any) string {
	for _, value := range values {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
