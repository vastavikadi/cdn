package main

import (
	"mime"
)

type FileCategory string

const (
	CategoryImage     FileCategory = "image"
	CategoryScripts   FileCategory = "scripts"
	CategoryStyle     FileCategory = "style"
	CategoryDocuments FileCategory = "documents"
	CategoryUnknown   FileCategory = "unknown"
)

func GetFileCategory(contentType string) FileCategory {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return CategoryUnknown
	}

	switch mediaType {
	case "image/jpeg", "image/png", "image/gif":
		return CategoryImage
	case "application/javascript", "text/javascript":
		return CategoryScripts
	case "text/css":
		return CategoryStyle

	case "application/pdf",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return CategoryDocuments

	default:
		return CategoryUnknown
	}

}
