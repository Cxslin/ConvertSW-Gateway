package services

import "time"

type UpstreamUpload struct {
	LocalFile  StoredFile
	UpstreamID string
	URL        string
	CreatedAt  time.Time
}

func NewUpstreamUpload(file StoredFile, upstreamID, url string) UpstreamUpload {
	return UpstreamUpload{LocalFile: file, UpstreamID: upstreamID, URL: url, CreatedAt: time.Now()}
}

func (u UpstreamUpload) FilePayload() map[string]any {
	return map[string]any{
		"id":        u.UpstreamID,
		"name":      u.LocalFile.Name,
		"url":       u.URL,
		"mime_type": u.LocalFile.MimeType,
		"size":      u.LocalFile.Size,
	}
}
