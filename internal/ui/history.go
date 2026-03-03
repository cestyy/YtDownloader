package app

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type HistoryEntry struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	FilePath  string    `json:"file_path"`
	OutputDir string    `json:"output_dir"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
}

type DownloadHistory struct {
	mu      sync.Mutex
	Entries []HistoryEntry `json:"entries"`
	app     fyne.App
}

func NewDownloadHistory(a fyne.App) *DownloadHistory {
	h := &DownloadHistory{app: a}
	h.Load()
	return h
}

func (h *DownloadHistory) storagePath() string {

	basePath := h.app.Storage().RootURI().Path()
	return filepath.Join(basePath, "history.json")
}

func (h *DownloadHistory) Load() {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := os.ReadFile(h.storagePath())
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &h.Entries)
}

func (h *DownloadHistory) Save() {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := json.MarshalIndent(h.Entries, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(h.storagePath()), 0755)
	_ = os.WriteFile(h.storagePath(), data, 0644)
}

func (h *DownloadHistory) Add(entry HistoryEntry) {
	h.mu.Lock()
	entry.ID = time.Now().Format("20060102150405")
	entry.Timestamp = time.Now()
	h.Entries = append([]HistoryEntry{entry}, h.Entries...)

	if len(h.Entries) > 500 {
		h.Entries = h.Entries[:500]
	}
	h.mu.Unlock()
	h.Save()
}

func (h *DownloadHistory) Remove(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, entry := range h.Entries {
		if entry.ID == id {
			h.Entries = append(h.Entries[:i], h.Entries[i+1:]...)
			break
		}
	}
}

func (h *DownloadHistory) Clear() {
	h.mu.Lock()
	h.Entries = nil
	h.mu.Unlock()
	h.Save()
}

func extractVideoID(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u == nil {
		return ""
	}
	if strings.Contains(u.Host, "youtube.com") {
		return u.Query().Get("v")
	} else if strings.Contains(u.Host, "youtu.be") {
		return strings.TrimPrefix(u.Path, "/")
	}
	return ""
}

func (mw *MainWindow) buildHistoryTab() *fyne.Container {
	list := widget.NewList(
		func() int {
			mw.History.mu.Lock()
			defer mw.History.mu.Unlock()
			return len(mw.History.Entries)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabel("Title")
			title.TextStyle.Bold = true
			title.Truncation = fyne.TextTruncateEllipsis

			urlLbl := widget.NewLabel("URL")
			urlLbl.TextStyle.Italic = true
			urlLbl.Truncation = fyne.TextTruncateEllipsis

			dateLbl := widget.NewLabel("Date")

			btnOpen := widget.NewButton(T("history_folder"), nil)
			btnPlay := widget.NewButton(T("history_play"), nil)
			btnRemove := widget.NewButton(T("history_remove"), nil)

			buttons := container.NewHBox(btnOpen, btnPlay, btnRemove)
			right := container.NewVBox(dateLbl, buttons)

			img := canvas.NewImageFromResource(theme.FileImageIcon())
			img.SetMinSize(fyne.NewSize(80, 45))
			img.FillMode = canvas.ImageFillContain

			return container.NewBorder(nil, nil, img, right, container.NewVBox(title, urlLbl))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			mw.History.mu.Lock()
			if int(i) >= len(mw.History.Entries) {
				mw.History.mu.Unlock()
				return
			}
			entry := mw.History.Entries[i]
			mw.History.mu.Unlock()

			c := o.(*fyne.Container)
			center := c.Objects[0].(*fyne.Container)
			title := center.Objects[0].(*widget.Label)
			urlLbl := center.Objects[1].(*widget.Label)

			right := c.Objects[2].(*fyne.Container)
			dateLbl := right.Objects[0].(*widget.Label)
			buttons := right.Objects[1].(*fyne.Container)
			btnOpen := buttons.Objects[0].(*widget.Button)
			btnPlay := buttons.Objects[1].(*widget.Button)
			btnRemove := buttons.Objects[2].(*widget.Button)

			title.SetText(entry.Title)
			urlLbl.SetText(entry.URL)
			dateLbl.SetText(entry.Timestamp.Format("2006-01-02 15:04"))

			if entry.Status == "Error" {
				title.SetText("❌ " + entry.Title)
				btnPlay.Disable()
			} else {
				btnPlay.Enable()
			}

			btnOpen.OnTapped = func() {
				_ = showFileInFolder(entry.FilePath, entry.OutputDir)
			}
			btnPlay.OnTapped = func() {
				_ = openFolderDirect(entry.FilePath)
			}
			btnRemove.OnTapped = func() {
				mw.History.Remove(entry.ID)
				mw.History.Save()

			}
		},
	)

	list.UpdateItem = func(i widget.ListItemID, o fyne.CanvasObject) {
		mw.History.mu.Lock()
		if int(i) >= len(mw.History.Entries) {
			mw.History.mu.Unlock()
			return
		}
		entry := mw.History.Entries[i]
		mw.History.mu.Unlock()

		c := o.(*fyne.Container)
		img := c.Objects[1].(*canvas.Image)
		center := c.Objects[0].(*fyne.Container)
		title := center.Objects[0].(*widget.Label)
		urlLbl := center.Objects[1].(*widget.Label)

		right := c.Objects[2].(*fyne.Container)
		dateLbl := right.Objects[0].(*widget.Label)
		buttons := right.Objects[1].(*fyne.Container)
		btnOpen := buttons.Objects[0].(*widget.Button)
		btnPlay := buttons.Objects[1].(*widget.Button)
		btnRemove := buttons.Objects[2].(*widget.Button)

		title.SetText(entry.Title)
		urlLbl.SetText(entry.URL)
		dateLbl.SetText(entry.Timestamp.Format("2006-01-02 15:04"))

		img.Resource = theme.FileImageIcon()
		vid := extractVideoID(entry.URL)
		if vid != "" {
			cached, ok := thumbCache.Load(vid)
			if ok && cached != "loading" {
				if res, isRes := cached.(fyne.Resource); isRes {
					img.Resource = res
				}
			} else if !ok {
				thumbCache.Store(vid, "loading")
				thumbURL := fmt.Sprintf("https://img.youtube.com/vi/%s/mqdefault.jpg", vid)
				go func(id, tUrl string, lIdx widget.ListItemID) {
					res := loadRemoteImageResource(tUrl)
					if res != nil {
						thumbCache.Store(id, res)
						fyne.Do(func() {
							list.RefreshItem(lIdx)
						})
					} else {
						thumbCache.Delete(id)
					}
				}(vid, thumbURL, i)
			}
		}
		img.Refresh()

		if entry.Status == "Error" {
			title.SetText("❌ " + entry.Title)
			btnPlay.Disable()
		} else {
			btnPlay.Enable()
		}

		btnOpen.OnTapped = func() {
			_ = showFileInFolder(entry.FilePath, entry.OutputDir)
		}
		btnPlay.OnTapped = func() {
			_ = openFolderDirect(entry.FilePath)
		}
		btnRemove.OnTapped = func() {
			mw.History.Remove(entry.ID)
			mw.History.Save()
			list.Refresh()
		}
	}

	btnClear := widget.NewButton(T("history_clear"), func() {
		mw.History.Clear()
		list.Refresh()
	})

	top := container.NewHBox(
		widget.NewLabelWithStyle(T("history_title"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		btnClear,
	)

	return container.NewBorder(top, nil, nil, nil, list)
}
