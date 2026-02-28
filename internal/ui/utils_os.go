package app

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func openFolderDirect(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return fmt.Errorf("empty dir")
	}
	if absDir, err := filepath.Abs(dir); err == nil {
		dir = absDir
	}

	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", dir).Start()
	case "darwin":
		return exec.Command("open", dir).Start()
	default:
		return exec.Command("xdg-open", dir).Start()
	}
}

func showFileInFolder(filePath string, fallbackDir string) error {
	if filePath == "" {
		return openFolderDirect(fallbackDir)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", "/select,", absPath).Start()
	case "darwin":
		return exec.Command("open", "-R", absPath).Start()
	default:
		dir := filepath.Dir(absPath)
		return exec.Command("xdg-open", dir).Start()
	}
}

func playDoneSound() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("powershell", "-c", "[System.Media.SystemSounds]::Asterisk.Play()")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Start()
	case "darwin":
		exec.Command("afplay", "/System/Library/Sounds/Glass.aiff").Start()
	case "linux":
		exec.Command("paplay", "/usr/share/sounds/freedesktop/stereo/complete.oga").Start()
	}
}
