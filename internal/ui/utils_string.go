package app

import "strings"

func sanitizeFileName(name string) string {
	invalid := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range invalid {
		name = strings.ReplaceAll(name, char, "_")
	}
	return strings.TrimSpace(name)
}
