package services

import "strings"

type ContextAttachmentManager struct {
	attachments []Attachment
}

func NewContextAttachmentManager() *ContextAttachmentManager {
	return &ContextAttachmentManager{}
}

func (m *ContextAttachmentManager) Add(attachment Attachment) {
	attachment.Name = NormalizeAttachmentName(attachment.Name)
	m.attachments = append(m.attachments, attachment)
}

func (m *ContextAttachmentManager) PromptBlock() string {
	blocks := []string{}
	for _, item := range m.attachments {
		if strings.TrimSpace(item.Text) == "" {
			continue
		}
		blocks = append(blocks, "File: "+item.Name+"\n"+item.Text)
	}
	return strings.Join(blocks, "\n\n")
}
