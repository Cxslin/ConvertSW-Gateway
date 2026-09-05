package services

import (
	"regexp"
	"sort"
	"strings"
)

type ToolNameMap struct {
	Forward map[string]string
	Reverse map[string]string
}

var explicitToolNameAliases = map[string]string{
	"Read":         "fs_open_file",
	"Write":        "fs_put_file",
	"Edit":         "fs_patch_file",
	"Bash":         "shell_run",
	"Grep":         "text_search",
	"Glob":         "path_find",
	"NotebookEdit": "notebook_patch",
	"WebFetch":     "http_get_url",
	"WebSearch":    "web_query",
}

var reverseToolNameAliases = func() map[string]string {
	out := make(map[string]string, len(explicitToolNameAliases))
	for raw, alias := range explicitToolNameAliases {
		out[alias] = raw
	}
	return out
}()

const qwenSafeAutoPrefix = "u_"

func ToQwenToolName(name string) string {
	if name == "" {
		return name
	}
	if alias, ok := explicitToolNameAliases[name]; ok {
		return alias
	}
	if _, ok := reverseToolNameAliases[name]; ok {
		return name
	}
	if strings.HasPrefix(name, qwenSafeAutoPrefix) {
		return name
	}
	return qwenSafeAutoPrefix + name
}

func FromQwenToolName(name string) string {
	if name == "" {
		return name
	}
	if raw, ok := reverseToolNameAliases[name]; ok {
		return raw
	}
	if strings.HasPrefix(name, qwenSafeAutoPrefix) && len(name) > len(qwenSafeAutoPrefix) {
		return strings.TrimPrefix(name, qwenSafeAutoPrefix)
	}
	return name
}

func ObfuscateToolNames(names []string) ToolNameMap {
	out := ToolNameMap{Forward: map[string]string{}, Reverse: map[string]string{}}
	for _, name := range names {
		alias := ToQwenToolName(name)
		out.Forward[name] = alias
		out.Reverse[alias] = name
	}
	return out
}

func ObfuscateBareToolNames(text string) string {
	if text == "" {
		return text
	}
	names := make([]string, 0, len(explicitToolNameAliases))
	for name := range explicitToolNameAliases {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return len(names[i]) > len(names[j])
	})
	pattern := regexp.MustCompile(`\b(` + strings.Join(regexpQuoteAll(names), "|") + `)\b`)
	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		if alias, ok := explicitToolNameAliases[match]; ok {
			return alias
		}
		return match
	})
}

func regexpQuoteAll(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, regexp.QuoteMeta(value))
	}
	return out
}
