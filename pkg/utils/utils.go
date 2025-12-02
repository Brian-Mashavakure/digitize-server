package utils

import (
	"encoding/base64"
	"path/filepath"
	"strings"
)

func IsValidImageType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}
	return validExtensions[ext]
}

func EncodeImageToBase64(imageData []byte) string {
	return base64.StdEncoding.EncodeToString(imageData)
}
