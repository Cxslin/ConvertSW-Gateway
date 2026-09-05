package services

import (
	"path/filepath"
	"strings"
)

type Attachment struct {
	ID       string
	Name     string
	Path     string
	MimeType string
	Size     int64
	Text     string
}

func NormalizeAttachmentName(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "." || name == string(filepath.Separator) {
		return "attachment"
	}
	return name
}

func AttachmentKind(name, mimeType string) string {
	lowered := strings.ToLower(mimeType + " " + filepath.Ext(name))
	switch {
	case strings.Contains(lowered, "image") || strings.Contains(lowered, ".png") || strings.Contains(lowered, ".jpg") || strings.Contains(lowered, ".webp"):
		return "image"
	case strings.Contains(lowered, "pdf") || strings.Contains(lowered, ".pdf"):
		return "pdf"
	case strings.Contains(lowered, "sheet") || strings.Contains(lowered, ".xlsx") || strings.Contains(lowered, ".csv"):
		return "spreadsheet"
	default:
		return "text"
	}
}
