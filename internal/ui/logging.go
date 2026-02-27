package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

type fileLogger struct {
	f *os.File
}

func initFileLogging(appName string) (*fileLogger, string, error) {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir, err = os.UserHomeDir()
		if err != nil {
			return nil, "", err
		}
		dir = filepath.Join(dir, "."+appName)
	}

	logDir := filepath.Join(dir, appName, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, "", err
	}

	logPath := filepath.Join(logDir, "app.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, "", err
	}

	log.SetOutput(f)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	log.Printf("---- START %s ----", time.Now().Format(time.RFC3339))
	return &fileLogger{f: f}, logPath, nil
}

func (l *fileLogger) Close() {
	if l == nil || l.f == nil {
		return
	}
	_ = l.f.Sync()
	_ = l.f.Close()
}

func logPanic(err any) {
	log.Printf("PANIC: %v\n%s", err, string(debug.Stack()))
}

func logErrorf(format string, args ...any) {
	log.Printf("ERROR: "+format, args...)
}

func logInfof(format string, args ...any) {
	log.Printf("INFO: "+format, args...)
}

func logWhere() string {
	return fmt.Sprintf("go=%s", runtimeVersion())
}

func runtimeVersion() string {
	return "unknown"
}
