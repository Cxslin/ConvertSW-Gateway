package toolcall

import (
	"encoding/json"
	"html"
	"regexp"
	"slices"
	"strings"
)

var (
	toolCallsBlockRe = regexp.MustCompile(`(?is)<tool_calls\b[^>]*>(.*?)</tool_calls\s*>`)
	invokeBlockRe    = regexp.MustCompile(`(?is)<invoke\b([^>]*)>(.*?)</invoke\s*>`)
	parameterBlockRe = regexp.MustCompile(`(?is)<parameter\b([^>]*)>(.*?)</parameter\s*>`)
	nameAttrRe       = regexp.MustCompile(`(?is)(?:^|[\s|])name\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s|/>]+))`)
	directNameAttrRe = regexp.MustCompile(`(?is)^\s*=\s*(?:"([^"]*)"|'([^']*)'|([A-Za-z0-9_.:-]+))`)
	bareNameAttrRe   = regexp.MustCompile(`(?is)^\s*([A-Za-z0-9_.:-]+)\s*$`)
	cdataRe          = regexp.MustCompile(`(?is)<!\[CDATA\[(.*?)\]\]>`)
	cdataSpanRe      = regexp.MustCompile(`(?is)<!\[CDATA\[[\s\S]*?\]\]>`)
	xmlTagRe         = regexp.MustCompile(`(?is)<\s*(/?)\s*([^<>]*?)\s*(/?)\s*>`)
	toolLocalNameRe  = regexp.MustCompile(`(?i)(tool\s*[-_ ]\s*calls|toolcalls|invoke|parameter)`)
	toolNameSepRe    = regexp.MustCompile(`[\s_-]+`)
	zeroWidthRe      = regexp.MustCompile(`[\x{200b}\x{200c}\x{200d}\x{feff}]`)
	qnmlChildBlockRe = regexp.MustCompile(`(?is)<([A-Za-z_][A-Za-z0-9_.:-]*)\b[^>]*>(.*?)</\s*([A-Za-z_][A-Za-z0-9_.:-]*)\s*>`)
	looseAttrRe      = regexp.MustCompile(`(?is)\b(?:name|parameter)\s*=\s*(?:"([^"]*)"|'([^']*)'|([A-Za-z0-9_.:-]+))`)
	qnmlDebrisRe     = regexp.MustCompile(`(?is)(?:</?\s*)?\|?\s*QNML\s*\|?\s*(?:tool_calls|tool-calls|toolcalls|invoke|parameter)\s*>?`)
)

var qnmlRawStringParamNames = map[string]bool{
	"content": true, "command": true, "cmd": true, "script": true,
	"code": true, "prompt": true, "file_content": true,
	"old_string": true, "new_string": true, "insert_text": true,
	"patch": true, "pattern": true, "text": true, "query": true,
	"url": true, "path": true, "file_path": true,
}

var markupReplacements = map[string]string{
	"＜": "<", "＞": ">", "／": "/", "∕": "/", "⁄": "/", "＝": "=",
	"｜": "|", "│": "|", "┃": "|", "▏": "|", "▕": "|",
	"“": `"`, "”": `"`, "„": `"`, "‟": `"`,
	"‘": `'`, "’": `'`, "‛": `'`,
	"﹤": "<", "﹥": ">",
	"Ο": "O", "ο": "o", "О": "O", "о": "o",
	"А": "A", "а": "a", "С": "C", "с": "c",
	"Е": "E", "е": "e", "Т": "T", "т": "t",
	"М": "M", "м": "m", "Ѕ": "S", "ѕ": "s",
	"Ι": "I", "І": "I", "і": "i", "Ν": "N", "η": "n",
}

var tagNameMap = map[string]string{
	"tool_calls": "tool_calls",
	"toolcalls":  "tool_calls",
	"tool_call":  "tool_calls",
	"tool-call":  "tool_calls",
	"invoke":     "invoke",
	"parameter":  "parameter",
}

// ParseQNMLToolCalls parses Qwen's preferred QNML tool-call envelope:
//
//	<|QNML|tool_calls>
//	  <|QNML|invoke name="Bash">
//	    <|QNML|parameter name="command"><![CDATA[pwd]]></|QNML|parameter>
//	  </|QNML|invoke>
//	</|QNML|tool_calls>
func ParseQNMLToolCalls(text string, tools []map[string]any) []ParsedToolCall {
	if strings.TrimSpace(text) == "" || len(tools) == 0 {
		return nil
	}
	allowed := map[string]string{}
	for _, tool := range tools {
		name := stringValue(tool, "name", "")
		if name != "" {
			allowed[strings.ToLower(name)] = name
		}
	}
	if len(allowed) == 0 {
		return nil
	}
	return parseQNMLToolCalls(text, allowed, tools)
}

func parseQNMLToolCalls(text string, allowed map[string]string, tools []map[string]any) []ParsedToolCall {
	calls := []ParsedToolCall{}
	for _, candidate := range extractQNMLCandidates(text) {
		for _, invoke := range invokeBlockRe.FindAllStringSubmatch(candidate, -1) {
			if len(invoke) < 3 {
				continue
			}
			name := canonicalToolName(extractNameAttr(invoke[1]), allowed)
			if name == "" {
				continue
			}
			input := parseQNMLParameters(invoke[2])
			if len(input) == 0 {
				input = parseQNMLFallbackInput(invoke[2])
			}
			calls = append(calls, ParsedToolCall{
				ID:    "call_" + randomID()[:12],
				Name:  name,
				Input: input,
			})
		}
	}
	if len(calls) == 0 {
		calls = append(calls, parseMangledQNMLToolCalls(text, allowed, tools)...)
	}
	return calls
}

func canonicalizeQNML(text string) string {
	text = canonicalizeMarkupOutsideCDATA(text)
	text = rewriteToolTagsOutsideCDATA(text)
	return repairMissingParameterCloses(text)
}

func canonicalizeMarkupOutsideCDATA(text string) string {
	if text == "" {
		return text
	}
	var out strings.Builder
	last := 0
	for _, loc := range cdataSpanRe.FindAllStringIndex(text, -1) {
		out.WriteString(canonicalizeMarkupPiece(text[last:loc[0]]))
		out.WriteString(text[loc[0]:loc[1]])
		last = loc[1]
	}
	out.WriteString(canonicalizeMarkupPiece(text[last:]))
	return out.String()
}

func canonicalizeMarkupPiece(piece string) string {
	for from, to := range markupReplacements {
		piece = strings.ReplaceAll(piece, from, to)
	}
	piece = strings.ReplaceAll(piece, "\u3000", " ")
	piece = strings.ReplaceAll(piece, "\u00a0", " ")
	piece = strings.ReplaceAll(piece, "▁", " ")
	return zeroWidthRe.ReplaceAllString(piece, "")
}

func rewriteToolTagsOutsideCDATA(text string) string {
	if text == "" {
		return text
	}
	var out strings.Builder
	last := 0
	for _, loc := range cdataSpanRe.FindAllStringIndex(text, -1) {
		out.WriteString(rewriteToolTags(text[last:loc[0]]))
		out.WriteString(text[loc[0]:loc[1]])
		last = loc[1]
	}
	out.WriteString(rewriteToolTags(text[last:]))
	return out.String()
}

func rewriteToolTags(text string) string {
	return xmlTagRe.ReplaceAllStringFunc(text, func(raw string) string {
		match := xmlTagRe.FindStringSubmatch(raw)
		if len(match) < 4 {
			return raw
		}
		if tag := canonicalToolTag(match[1] != "", match[2], match[3] != ""); tag != "" {
			return tag
		}
		return raw
	})
}

func canonicalToolTag(closing bool, body string, selfClosing bool) string {
	body = strings.TrimSpace(body)
	if body == "" || strings.HasPrefix(body, "![CDATA") {
		return ""
	}
	if strings.HasSuffix(body, "/") {
		body = strings.TrimSpace(strings.TrimSuffix(body, "/"))
		selfClosing = true
	}
	name, attrsStart := detectToolLocalName(body)
	if name == "" {
		return ""
	}
	if closing {
		return "</" + name + ">"
	}
	attrs := ""
	attrText := ""
	if attrsStart < len(body) {
		attrText = strings.Trim(body[attrsStart:], " \t\r\n|")
	}
	if attrName := extractNameAttr(attrText); attrName != "" && (name == "invoke" || name == "parameter") {
		attrs = ` name="` + html.EscapeString(attrName) + `"`
	}
	suffix := ""
	if selfClosing {
		suffix = "/"
	}
	return "<" + name + attrs + suffix + ">"
}

func detectToolLocalName(body string) (string, int) {
	for _, loc := range toolLocalNameRe.FindAllStringSubmatchIndex(body, -1) {
		if len(loc) < 4 {
			continue
		}
		start, end := loc[2], loc[3]
		before := body[:start]
		after := body[end:]
		if !toolNamePrefixAllowed(before) || !toolNameSuffixAllowed(after) {
			continue
		}
		raw := strings.ToLower(toolNameSepRe.ReplaceAllString(body[start:end], "_"))
		if name := tagNameMap[raw]; name != "" {
			return name, end
		}
	}
	return "", 0
}

func toolNamePrefixAllowed(prefix string) bool {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return true
	}
	if strings.ContainsAny(prefix, "|_- ") {
		return true
	}
	return regexp.MustCompile(`(?i)^(?:q?n?d?s?ml|dsmart|agent|tool|[a-z0-9]{1,32})$`).MatchString(prefix)
}

func toolNameSuffixAllowed(suffix string) bool {
	suffix = strings.TrimLeft(suffix, " \t\r\n")
	if suffix == "" {
		return true
	}
	if strings.HasPrefix(strings.ToLower(suffix), "name=") {
		return true
	}
	return strings.ContainsRune(" \t\r\n|/>=", rune(suffix[0]))
}

func repairMissingParameterCloses(text string) string {
	if !strings.Contains(strings.ToLower(text), "<parameter") {
		return text
	}
	for {
		lowered := strings.ToLower(text)
		var out strings.Builder
		cursor := 0
		changed := false
		for cursor < len(text) {
			openRel := strings.Index(lowered[cursor:], "<parameter")
			if openRel < 0 {
				out.WriteString(text[cursor:])
				break
			}
			open := cursor + openRel
			openerEndRel := strings.Index(text[open:], ">")
			if openerEndRel < 0 {
				out.WriteString(text[cursor:])
				break
			}
			openerEnd := open + openerEndRel + 1
			out.WriteString(text[cursor:openerEnd])
			opener := text[open:openerEnd]
			if strings.HasSuffix(strings.TrimSpace(opener), "/>") {
				cursor = openerEnd
				continue
			}
			tailLower := lowered[openerEnd:]
			boundary := firstNonNegativeIndex(
				strings.Index(tailLower, "<parameter"),
				strings.Index(tailLower, "</invoke"),
			)
			if boundary < 0 {
				cursor = openerEnd
				continue
			}
			paramClose := strings.Index(tailLower, "</parameter")
			if paramClose >= 0 && paramClose < boundary {
				cursor = openerEnd
				continue
			}
			boundaryAbs := openerEnd + boundary
			out.WriteString(text[openerEnd:boundaryAbs])
			out.WriteString("</parameter>")
			cursor = boundaryAbs
			changed = true
		}
		if !changed {
			return text
		}
		text = out.String()
	}
}

func firstNonNegativeIndex(values ...int) int {
	best := -1
	for _, value := range values {
		if value >= 0 && (best < 0 || value < best) {
			best = value
		}
	}
	return best
}

func extractQNMLCandidates(text string) []string {
	canonical := canonicalizeQNML(stripMarkdownFencedToolExamples(text))
	candidates := []string{}
	for _, match := range toolCallsBlockRe.FindAllStringSubmatch(canonical, -1) {
		if len(match) == 2 {
			candidates = append(candidates, match[1])
		}
	}
	if len(candidates) > 0 {
		return candidates
	}
	lowered := strings.ToLower(canonical)
	if open := strings.Index(lowered, "<tool_calls"); open >= 0 {
		tail := canonical[open:]
		tailLower := lowered[open:]
		if !strings.Contains(tailLower, "</tool_calls") && strings.Contains(tailLower, "</invoke") {
			candidates = append(candidates, tail+"</tool_calls>")
		}
	}
	if invoke := strings.Index(lowered, "<invoke"); invoke >= 0 {
		candidates = append(candidates, canonical[invoke:])
	}
	return candidates
}

func stripMarkdownFencedToolExamples(text string) string {
	if text == "" {
		return text
	}
	lines := strings.SplitAfter(text, "\n")
	out := strings.Builder{}
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	inCDATA := false
	for _, line := range lines {
		if inCDATA {
			out.WriteString(line)
			if strings.Contains(line, "]]>") {
				inCDATA = false
			}
			continue
		}
		cdataIdx := strings.Index(line, "<![CDATA[")
		fenceIdx := firstFenceIndex(line)
		if cdataIdx >= 0 && (fenceIdx < 0 || cdataIdx < fenceIdx) {
			out.WriteString(line)
			if !strings.Contains(line[cdataIdx:], "]]>") {
				inCDATA = true
			}
			continue
		}
		stripped := strings.TrimLeft(line, " \t\r")
		if !inFence {
			if runChar, runLen := fenceRun(stripped); runLen >= 3 {
				inFence = true
				fenceChar = runChar
				fenceLen = runLen
				continue
			}
			out.WriteString(line)
			continue
		}
		runChar, runLen := fenceRun(stripped)
		if runLen >= fenceLen && runChar == fenceChar && strings.TrimSpace(stripped[runLen:]) == "" {
			inFence = false
			fenceChar = 0
			fenceLen = 0
		}
	}
	return out.String()
}

func firstFenceIndex(line string) int {
	indexes := []int{}
	if idx := strings.Index(line, "```"); idx >= 0 {
		indexes = append(indexes, idx)
	}
	if idx := strings.Index(line, "~~~"); idx >= 0 {
		indexes = append(indexes, idx)
	}
	if len(indexes) == 0 {
		return -1
	}
	return slices.Min(indexes)
}

func fenceRun(stripped string) (byte, int) {
	if stripped == "" || stripped[0] != '`' && stripped[0] != '~' {
		return 0, 0
	}
	ch := stripped[0]
	run := 0
	for run < len(stripped) && stripped[run] == ch {
		run++
	}
	if run < 3 {
		return 0, 0
	}
	return ch, run
}

func extractNameAttr(attrs string) string {
	for _, re := range []*regexp.Regexp{nameAttrRe, directNameAttrRe, bareNameAttrRe} {
		match := re.FindStringSubmatch(attrs)
		if len(match) == 0 {
			continue
		}
		for _, item := range match[1:] {
			if item != "" {
				return html.UnescapeString(strings.TrimSpace(item))
			}
		}
	}
	return ""
}

func parseQNMLParameters(body string) map[string]any {
	out := map[string]any{}
	for _, param := range parameterBlockRe.FindAllStringSubmatch(body, -1) {
		if len(param) < 3 {
			continue
		}
		name := strings.TrimSpace(extractNameAttr(param[1]))
		if name == "" {
			continue
		}
		out[name] = decodeQNMLValue(param[2], name)
	}
	if len(out) == 0 {
		out = parseQNMLDirectChildParameters(body)
	}
	return out
}

func parseQNMLDirectChildParameters(body string) map[string]any {
	out := map[string]any{}
	for _, child := range qnmlChildBlockRe.FindAllStringSubmatch(body, -1) {
		if len(child) < 4 {
			continue
		}
		openName := strings.TrimSpace(child[1])
		closeName := strings.TrimSpace(child[3])
		if openName == "" || !strings.EqualFold(openName, closeName) {
			continue
		}
		name := strings.ToLower(openName)
		switch name {
		case "tool_calls", "toolcalls", "invoke", "parameter":
			continue
		}
		out[openName] = decodeQNMLValue(child[2], openName)
	}
	return out
}

func parseQNMLFallbackInput(body string) map[string]any {
	body = strings.TrimSpace(body)
	if body == "" {
		return map[string]any{}
	}
	var decoded any
	if err := json.Unmarshal([]byte(body), &decoded); err == nil {
		if m, ok := NormalizeToolInput(decoded).(map[string]any); ok {
			return m
		}
	}
	if kv := ParseTextKVInput(body); len(kv) > 0 {
		return kv
	}
	return map[string]any{"input": decodeQNMLValue(body, "input")}
}

type looseQNMLAttr struct {
	Name    string
	Value   any
	Pos     int
	IsTool  bool
	RawName string
}

func parseMangledQNMLToolCalls(text string, allowed map[string]string, tools []map[string]any) []ParsedToolCall {
	if !hasMangledQNMLSignal(text) {
		return nil
	}
	attrs := extractLooseQNMLAttrs(text, allowed)
	if len(attrs) == 0 {
		return nil
	}
	calls := []ParsedToolCall{}
	for i, attr := range attrs {
		if !attr.IsTool {
			continue
		}
		nextToolPos := len(text)
		for j := i + 1; j < len(attrs); j++ {
			if attrs[j].IsTool {
				nextToolPos = attrs[j].Pos
				break
			}
		}
		input := map[string]any{}
		for j := i + 1; j < len(attrs); j++ {
			field := attrs[j]
			if field.Pos >= nextToolPos {
				break
			}
			if field.IsTool || strings.TrimSpace(field.Name) == "" || field.Value == nil {
				continue
			}
			input[field.Name] = field.Value
		}
		input = filterLooseInputForTool(attr.Name, input, tools)
		if len(input) == 0 && len(requiredToolArgs(attr.Name, tools)) > 0 {
			continue
		}
		call := ParsedToolCall{ID: "call_" + randomID()[:12], Name: attr.Name, Input: input}
		call.Input = CoerceToolInput(call.Name, call.Input, tools)
		if missingRequiredArgs(call.Name, call.Input, tools) {
			continue
		}
		calls = append(calls, call)
	}
	return calls
}

func hasMangledQNMLSignal(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	lowered := strings.ToLower(canonicalizeMarkupOutsideCDATA(text))
	return strings.Contains(lowered, "qnml") &&
		(strings.Contains(lowered, "<![cdata[") || strings.Contains(lowered, "tool_calls") || strings.Contains(lowered, "invoke") || strings.Contains(lowered, "parameter"))
}

func extractLooseQNMLAttrs(text string, allowed map[string]string) []looseQNMLAttr {
	text = canonicalizeMarkupOutsideCDATA(stripMarkdownFencedToolExamples(text))
	spans := cdataSpanRe.FindAllStringIndex(text, -1)
	matches := looseAttrRe.FindAllStringSubmatchIndex(text, -1)
	attrs := []looseQNMLAttr{}
	for idx, match := range matches {
		if len(match) < 8 || indexInsideSpans(match[0], spans) {
			continue
		}
		rawName := firstMatchedGroup(text, match, 2, 4, 6)
		rawName = html.UnescapeString(strings.TrimSpace(rawName))
		if rawName == "" {
			continue
		}
		if toolName := canonicalToolName(rawName, allowed); toolName != "" {
			attrs = append(attrs, looseQNMLAttr{Name: toolName, Pos: match[0], IsTool: true, RawName: rawName})
			continue
		}
		segmentEnd := len(text)
		for next := idx + 1; next < len(matches); next++ {
			if indexInsideSpans(matches[next][0], spans) {
				continue
			}
			segmentEnd = matches[next][0]
			break
		}
		value, ok := firstCDATAValueBetween(text, match[1], segmentEnd)
		if !ok {
			continue
		}
		cleaned := cleanMangledQNMLValue(value)
		rawString := qnmlRawStringParamNames[strings.ToLower(rawName)]
		attrs = append(attrs, looseQNMLAttr{
			Name:    rawName,
			Value:   coerceQNMLScalar(cleaned, rawString),
			Pos:     match[0],
			RawName: rawName,
		})
	}
	return attrs
}

func firstMatchedGroup(text string, match []int, pairs ...int) string {
	for _, startIndex := range pairs {
		if startIndex+1 >= len(match) {
			continue
		}
		start, end := match[startIndex], match[startIndex+1]
		if start >= 0 && end >= start {
			return text[start:end]
		}
	}
	return ""
}

func indexInsideSpans(index int, spans [][]int) bool {
	for _, span := range spans {
		if len(span) == 2 && index >= span[0] && index < span[1] {
			return true
		}
	}
	return false
}

func firstCDATAValueBetween(text string, start, end int) (string, bool) {
	if start < 0 {
		start = 0
	}
	if end > len(text) || end < 0 {
		end = len(text)
	}
	if start > end {
		return "", false
	}
	for _, part := range cdataRe.FindAllStringSubmatch(text[start:end], -1) {
		if len(part) == 2 {
			return part[1], true
		}
	}
	return "", false
}

func cleanMangledQNMLValue(value string) string {
	value = canonicalizeMarkupPiece(value)
	value = qnmlDebrisRe.ReplaceAllString(value, "")
	value = strings.ReplaceAll(value, "</|", "")
	value = strings.ReplaceAll(value, "<|", "")
	return strings.TrimSpace(value)
}

func filterLooseInputForTool(name string, input map[string]any, tools []map[string]any) map[string]any {
	if len(input) == 0 {
		return input
	}
	props := schemaProperties(toolSchema(name, tools))
	if len(props) == 0 {
		return input
	}
	out := map[string]any{}
	for key, value := range input {
		if _, ok := props[key]; ok {
			out[key] = value
		}
	}
	return out
}

func decodeQNMLValue(raw string, paramName string) any {
	rawString := qnmlRawStringParamNames[strings.ToLower(strings.TrimSpace(paramName))]
	if parts := cdataRe.FindAllStringSubmatch(raw, -1); len(parts) > 0 {
		var joined strings.Builder
		for _, part := range parts {
			if len(part) == 2 {
				joined.WriteString(part[1])
			}
		}
		if rawString {
			return joined.String()
		}
		return coerceQNMLScalar(joined.String(), false)
	}
	if !rawString {
		if nested, ok := parseQNMLNestedValue(raw); ok {
			return nested
		}
	}
	return coerceQNMLScalar(raw, rawString)
}

func coerceQNMLScalar(raw string, rawString bool) any {
	value := html.UnescapeString(strings.TrimSpace(raw))
	if rawString {
		return value
	}
	switch strings.ToLower(value) {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err == nil {
		return NormalizeToolInput(decoded)
	}
	return value
}

func parseQNMLNestedValue(raw string) (any, bool) {
	text := strings.TrimSpace(raw)
	if text == "" || !strings.Contains(text, "<") {
		return nil, false
	}
	matches := qnmlChildBlockRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil, false
	}
	position := 0
	names := []string{}
	values := []any{}
	for _, match := range matches {
		if strings.TrimSpace(text[position:match[0]]) != "" {
			return nil, false
		}
		openName := text[match[2]:match[3]]
		closeName := text[match[6]:match[7]]
		if !strings.EqualFold(openName, closeName) {
			return nil, false
		}
		body := text[match[4]:match[5]]
		name := strings.TrimSpace(openName)
		names = append(names, name)
		values = append(values, decodeQNMLValue(body, name))
		position = match[1]
	}
	if strings.TrimSpace(text[position:]) != "" {
		return nil, false
	}
	if len(names) == 0 {
		return nil, false
	}
	allItems := true
	for _, name := range names {
		if !strings.EqualFold(name, "item") {
			allItems = false
			break
		}
	}
	if allItems {
		return values, true
	}
	out := map[string]any{}
	for i, name := range names {
		if existing, ok := out[name]; ok {
			if list, ok := existing.([]any); ok {
				out[name] = append(list, values[i])
			} else {
				out[name] = []any{existing, values[i]}
			}
		} else {
			out[name] = values[i]
		}
	}
	return out, true
}
