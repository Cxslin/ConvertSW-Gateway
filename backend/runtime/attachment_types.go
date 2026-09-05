package runtime

type AttachmentKind string

const (
	AttachmentText  AttachmentKind = "text"
	AttachmentImage AttachmentKind = "image"
	AttachmentPDF   AttachmentKind = "pdf"
	AttachmentFile  AttachmentKind = "file"
)

type Attachment struct {
	ID       string
	Name     string
	Kind     AttachmentKind
	MimeType string
	Text     string
	URL      string
}
