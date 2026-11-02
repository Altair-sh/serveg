package http

import (
	"mime"
	"path"
)

const MimeDefault = "application/octet-stream"
const TextPlain = "text/plain; charset=utf-8"
const TextHtml = "text/html; charset=utf-8"

func GetMimeType(filePath string) string {
	t := mime.TypeByExtension(path.Ext(filePath))
	if t == "" {
		t = MimeDefault
	}
	return t
}
