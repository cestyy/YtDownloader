package app

import (
	"os"
	"path/filepath"
)

func defaultDownloadsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "Downloads")
}
