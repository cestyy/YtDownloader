package app

import (
	"fmt"
	"image/color"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type LogLevel int

const (
	LogInfo LogLevel = iota
	LogWarn
	LogErr
	LogDbg
)

type logEntry struct {
	lvl LogLevel
	msg string
	ts  time.Time
}

type UILogger struct {
	mu     sync.Mutex
	items  []logEntry
	max    int
	filter string

	box    *fyne.Container
	scroll *container.Scroll

	filterSel *widget.Select
	btnCopy   *widget.Button
	btnClear  *widget.Button
}

func NewUILogger(max int) *UILogger {
	if max <= 0 {
		max = 500
	}
	l := &UILogger{
		max:    max,
		filter: "All",
		items:  make([]logEntry, 0, max),
		box:    container.NewVBox(),
	}
	l.scroll = container.NewVScroll(l.box)

	l.filterSel = widget.NewSelect([]string{"All", "Info", "Warn", "Error", "Debug"}, func(s string) {
		l.SetFilter(s)
	})
	l.filterSel.SetSelected("All")

	l.btnCopy = widget.NewButton("Copy all", func() {})
	l.btnClear = widget.NewButton("Clear log", func() {})

	return l
}

func (l *UILogger) Controls(w fyne.Window) fyne.CanvasObject {
	l.btnCopy.OnTapped = func() {
		w.Clipboard().SetContent(l.AllText())
	}
	l.btnClear.OnTapped = func() {
		l.Clear()
	}
	return container.NewHBox(l.filterSel, l.btnCopy, l.btnClear)
}

func (l *UILogger) Widget() fyne.CanvasObject {
	return l.scroll
}

func (l *UILogger) SetFilter(s string) {
	if s == "" {
		s = "All"
	}
	l.mu.Lock()
	l.filter = s
	l.mu.Unlock()
	l.rebuild()
}

func (l *UILogger) Clear() {
	l.mu.Lock()
	l.items = l.items[:0]
	l.mu.Unlock()
	fyne.Do(func() {
		l.box.Objects = nil
		l.box.Refresh()
		l.scroll.ScrollToTop()
		l.scroll.Refresh()
	})
}

func (l *UILogger) AllText() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var b strings.Builder
	for _, e := range l.items {
		b.WriteString(fmt.Sprintf("%s %s | %s\n", e.ts.Format(time.RFC3339), l.levelTag(e.lvl), e.msg))
	}
	return b.String()
}

func (l *UILogger) Info(s string) { l.add(LogInfo, s) }
func (l *UILogger) Warn(s string) { l.add(LogWarn, s) }
func (l *UILogger) Err(s string)  { l.add(LogErr, s) }
func (l *UILogger) Dbg(s string)  { l.add(LogDbg, s) }

func (l *UILogger) add(level LogLevel, s string) {
	s = strings.TrimRight(s, "\r\n")
	if strings.TrimSpace(s) == "" {
		return
	}

	e := logEntry{lvl: level, msg: s, ts: time.Now()}

	l.mu.Lock()
	l.items = append(l.items, e)
	if len(l.items) > l.max {
		l.items = l.items[len(l.items)-l.max:]
	}
	show := l.passesFilter(level)
	l.mu.Unlock()

	if !show {
		return
	}

	fyne.Do(func() {
		line := canvas.NewText(fmt.Sprintf("%s %s | %s", e.ts.Format("15:04:05"), l.levelTag(level), e.msg), l.levelColor(level))
		line.TextSize = theme.TextSize()
		line.TextStyle = fyne.TextStyle{Monospace: true}
		line.Alignment = fyne.TextAlignLeading
		l.box.Add(line)
		l.box.Refresh()
		l.scroll.ScrollToBottom()
		l.scroll.Refresh()
	})
}

func (l *UILogger) rebuild() {
	l.mu.Lock()
	filter := l.filter
	copyItems := make([]logEntry, 0, len(l.items))
	for _, it := range l.items {
		if l.passesFilterWith(filter, it.lvl) {
			copyItems = append(copyItems, it)
		}
	}
	l.mu.Unlock()

	fyne.Do(func() {
		l.box.Objects = nil
		for _, e := range copyItems {
			line := canvas.NewText(fmt.Sprintf("%s %s | %s", e.ts.Format("15:04:05"), l.levelTag(e.lvl), e.msg), l.levelColor(e.lvl))
			line.TextSize = theme.TextSize()
			line.TextStyle = fyne.TextStyle{Monospace: true}
			line.Alignment = fyne.TextAlignLeading
			l.box.Add(line)
		}
		l.box.Refresh()
		l.scroll.ScrollToBottom()
		l.scroll.Refresh()
	})
}

func (l *UILogger) passesFilter(level LogLevel) bool {
	return l.passesFilterWith(l.filter, level)
}

func (l *UILogger) passesFilterWith(filter string, level LogLevel) bool {
	switch filter {
	case "Info":
		return level == LogInfo
	case "Warn":
		return level == LogWarn
	case "Error":
		return level == LogErr
	case "Debug":
		return level == LogDbg
	default:
		return true
	}
}

func (l *UILogger) levelTag(level LogLevel) string {
	switch level {
	case LogErr:
		return "ERR"
	case LogWarn:
		return "WRN"
	case LogDbg:
		return "DBG"
	default:
		return "INF"
	}
}

func (l *UILogger) levelColor(level LogLevel) color.Color {
	switch level {
	case LogErr:
		return theme.Color(theme.ColorNameError)
	case LogWarn:
		return theme.Color(theme.ColorNameWarning)
	case LogDbg:
		return theme.Color(theme.ColorNameDisabled)
	default:
		return theme.Color(theme.ColorNameForeground)
	}
}
