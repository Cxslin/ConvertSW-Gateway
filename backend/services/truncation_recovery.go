package services

import (
	"regexp"
	"strings"
)

func RecoverTruncatedJSON(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}
	openBraces := strings.Count(text, "{") - strings.Count(text, "}")
	openBrackets := strings.Count(text, "[") - strings.Count(text, "]")
	for openBrackets > 0 {
		text += "]"
		openBrackets--
	}
	for openBraces > 0 {
		text += "}"
		openBraces--
	}
	return text
}

var (
	qnmlToolCallsOpenRe    = regexp.MustCompile(`(?is)<\s*\|\s*QNML\s*\|\s*tool_calls\b[^>]*>`)
	qnmlToolCallsCloseRe   = regexp.MustCompile(`(?is)<\s*/\s*\|\s*QNML\s*\|\s*tool_calls\s*>`)
	qnmlInvokeOpenRe       = regexp.MustCompile(`(?is)<\s*\|\s*QNML\s*\|\s*invoke\b[^>]*>`)
	qnmlInvokeCloseRe      = regexp.MustCompile(`(?is)<\s*/\s*\|\s*QNML\s*\|\s*invoke\s*>`)
	qnmlParameterOpenRe    = regexp.MustCompile(`(?is)<\s*\|\s*QNML\s*\|\s*parameter\b[^>]*>`)
	qnmlParameterCloseRe   = regexp.MustCompile(`(?is)<\s*/\s*\|\s*QNML\s*\|\s*parameter\s*>`)
	legacyToolCallsOpenRe  = regexp.MustCompile(`(?is)<\s*tool_calls\b[^>]*>`)
	legacyToolCallsCloseRe = regexp.MustCompile(`(?is)<\s*/\s*tool_calls\s*>`)
	legacyInvokeOpenRe     = regexp.MustCompile(`(?is)<\s*invoke\b[^>]*>`)
	legacyInvokeCloseRe    = regexp.MustCompile(`(?is)<\s*/\s*invoke\s*>`)
	legacyParameterOpenRe  = regexp.MustCompile(`(?is)<\s*parameter\b[^>]*>`)
	legacyParameterCloseRe = regexp.MustCompile(`(?is)<\s*/\s*parameter\s*>`)
	legacyToolCallOpenRe   = regexp.MustCompile(`(?is)<\s*tool_call\b[^>]*>`)
	legacyToolCallCloseRe  = regexp.MustCompile(`(?is)<\s*/\s*tool_call\s*>`)
	hashToolCallOpenRe     = regexp.MustCompile(`(?is)##\s*TOOL_CALL\s*##`)
	hashToolCallCloseRe    = regexp.MustCompile(`(?is)##\s*END_CALL\s*##`)
	cdataOpenRe            = regexp.MustCompile(`(?is)<!\[CDATA\[`)
	cdataCloseRe           = regexp.MustCompile(`(?is)\]\]>`)
	partialToolMarkerRe    = regexp.MustCompile(`(?is)(?:<\s*/?\s*(?:\|\s*QNML(?:\s*\|\s*(?:tool_calls|invoke|parameter)?)?|tool_calls?|invoke|parameter)|##\s*(?:TOOL_CALL|END_CALL)?)\s*$`)
)

func IsToolCallTruncated(text string) bool {
	trimmed := strings.TrimRight(text, " \t\r\n")
	if trimmed == "" {
		return false
	}
	if partialToolMarkerRe.MatchString(trimmed) {
		return true
	}
	if !HasTextualToolMarker(trimmed) {
		return false
	}
	return hasUnclosed(qnmlToolCallsOpenRe, qnmlToolCallsCloseRe, trimmed) ||
		hasUnclosed(qnmlInvokeOpenRe, qnmlInvokeCloseRe, trimmed) ||
		hasUnclosed(qnmlParameterOpenRe, qnmlParameterCloseRe, trimmed) ||
		hasUnclosed(legacyToolCallsOpenRe, legacyToolCallsCloseRe, trimmed) ||
		hasUnclosed(legacyInvokeOpenRe, legacyInvokeCloseRe, trimmed) ||
		hasUnclosed(legacyParameterOpenRe, legacyParameterCloseRe, trimmed) ||
		hasUnclosed(legacyToolCallOpenRe, legacyToolCallCloseRe, trimmed) ||
		hasUnclosed(hashToolCallOpenRe, hashToolCallCloseRe, trimmed) ||
		countMatches(cdataOpenRe, trimmed) > countMatches(cdataCloseRe, trimmed)
}

func hasUnclosed(openRe, closeRe *regexp.Regexp, text string) bool {
	return countMatches(openRe, text) > countMatches(closeRe, text)
}

func countMatches(re *regexp.Regexp, text string) int {
	return len(re.FindAllStringIndex(text, -1))
}

func DeduplicateContinuation(existing, continuation string) string {
	if existing == "" || continuation == "" {
		return continuation
	}
	maxOverlap := minInt(500, minInt(len(existing), len(continuation)))
	if maxOverlap >= 10 {
		for length := maxOverlap; length >= 10; length-- {
			if strings.HasSuffix(existing, continuation[:length]) {
				return continuation[length:]
			}
		}
	}
	tailLines := lastNonNilLines(existing, 20)
	contLines := strings.Split(continuation, "\n")
	if len(tailLines) > 0 && len(contLines) > 0 {
		first := strings.TrimSpace(contLines[0])
		if first != "" {
			for i := range tailLines {
				if strings.TrimSpace(tailLines[i]) != first {
					continue
				}
				matched := 1
				for k := 1; k < len(contLines) && i+k < len(tailLines); k++ {
					if strings.TrimSpace(contLines[k]) != strings.TrimSpace(tailLines[i+k]) {
						break
					}
					matched++
				}
				if matched >= 2 {
					return strings.Join(contLines[matched:], "\n")
				}
			}
		}
	}
	return continuation
}

func BuildContinuationPrompt(partialResponse string, anchorChars int) (string, string) {
	if anchorChars <= 0 {
		anchorChars = 2000
	}
	anchor := partialResponse
	if len(anchor) > anchorChars {
		anchor = anchor[len(anchor)-anchorChars:]
	}
	assistantContext := anchor
	if len(partialResponse) > anchorChars {
		assistantContext = "...\n" + anchor
	}
	tail := anchor
	if len(tail) > 300 {
		tail = tail[len(tail)-300:]
	}
	followup := "Your previous response was cut off in the middle of a QNML/tool-call block. The last part was:\n\n" +
		"```\n..." + tail + "\n```\n\n" +
		"Continue EXACTLY from where you stopped. DO NOT repeat any content already generated. " +
		"DO NOT restart the response. Output ONLY the remaining QNML/tool-call text, starting immediately from the cut-off point."
	return assistantContext, followup
}

func lastNonNilLines(text string, limit int) []string {
	lines := strings.Split(text, "\n")
	if limit <= 0 || len(lines) <= limit {
		return lines
	}
	return lines[len(lines)-limit:]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
