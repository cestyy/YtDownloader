package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func formatBytes(b int64) string {
	if b <= 0 {
		return "? MB"
	}
	mb := float64(b) / (1024 * 1024)
	return fmt.Sprintf("%.1f MB", mb)
}

func formatDuration(sec float64) string {
	if sec <= 0 {
		return "?:??"
	}
	d := time.Duration(sec * float64(time.Second))
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func emptyToDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func parsePercent(s string) float64 {
	var cleaned strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			cleaned.WriteRune(r)
		}
	}
	if cleaned.Len() == 0 {
		return -1
	}
	f, err := strconv.ParseFloat(cleaned.String(), 64)
	if err != nil {
		return -1
	}
	return f
}
