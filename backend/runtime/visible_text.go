package runtime

import (
	"regexp"
	"strings"
)

var hiddenBlocks = []*regexp.Regexp{
	regexp.MustCompile(`(?is)<think>.*?</think>`),
	regexp.MustCompile(`(?is)<tool_use\b.*?</tool_use>`),
	regexp.MustCompile(`(?is)<tool_call\b.*?</tool_call>`),
	regexp.MustCompile(`(?is)<system-reminder>.*?</system-reminder>`),
}

func VisibleText(text string) string {
	out := text
	for _, re := range hiddenBlocks {
		out = re.ReplaceAllString(out, "")
	}
	out = strings.ReplaceAll(out, "\r\n", "\n")
	return strings.TrimSpace(out)
}
