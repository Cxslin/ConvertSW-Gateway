package services

import "strings"

type WorkspaceContext struct {
	Root  string
	Files []Attachment
}

func (c WorkspaceContext) Render() string {
	parts := []string{}
	if strings.TrimSpace(c.Root) != "" {
		parts = append(parts, "Workspace: "+c.Root)
	}
	for _, file := range c.Files {
		if strings.TrimSpace(file.Text) != "" {
			parts = append(parts, "File: "+file.Name+"\n"+file.Text)
		}
	}
	return strings.Join(parts, "\n\n")
}
