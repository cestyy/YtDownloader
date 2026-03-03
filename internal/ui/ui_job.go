package app

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"YtDownloader/internal/ytdlp"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type JobUI struct {
	Root           *fyne.Container
	ProgBar        *widget.ProgressBar
	StatusLbl      *widget.Label
	BtnCancel      *widget.Button
	BtnPauseResume *widget.Button
	BtnOpenDir     *widget.Button
	ExpandBtn      *widget.Button
	ChildBox       *fyne.Container
	ChildProgs     []float64
	ChildStats     []string
	ChildList      *widget.List
	ActiveIndices  []int
}

type DownloadJob struct {
	Title            string
	URL              string
	DlFormat         string
	OutputDir        string
	FormatSelect     string
	BrowserSelect    string
	CookiesFilePath  string
	AllowPl          bool
	UseSb            bool
	Naming           string
	SelectedItemsStr string

	PlaylistEntries []ytdlp.PlaylistEntry
	PlaylistChecks  []bool

	Status   string
	Progress float64
	Speed    string
	ETA      string

	Thumbnail fyne.Resource

	Ctx    context.Context
	Cancel context.CancelFunc

	CustomArgs string
	EmbedMeta  bool

	TargetDir string

	TouchedFiles []string
	TouchedMu    sync.Mutex

	UI *JobUI
}

func (j *DownloadJob) cleanupTouchedFiles() {
	j.TouchedMu.Lock()
	defer j.TouchedMu.Unlock()
	for _, f := range j.TouchedFiles {
		_ = os.Remove(f + ".part")
		_ = os.Remove(f + ".ytdl")
		_ = os.Remove(f + ".temp")
	}
}

func findListIdx(activeIndices []int, realIdx int) int {
	for li, ai := range activeIndices {
		if ai == realIdx {
			return li
		}
	}
	return -1
}

func (mw *MainWindow) buildJobUI(job *DownloadJob) *JobUI {
	ui := &JobUI{}

	titleLbl := widget.NewLabelWithStyle(job.Title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	titleLbl.Truncation = fyne.TextTruncateEllipsis

	ui.ProgBar = widget.NewProgressBar()
	ui.ProgBar.Min, ui.ProgBar.Max = 0, 100

	ui.StatusLbl = widget.NewLabel(StatusQueued)
	ui.StatusLbl.TextStyle = fyne.TextStyle{Italic: true}
	ui.StatusLbl.Truncation = fyne.TextTruncateEllipsis

	ui.BtnCancel = widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		mw.JobsMu.Lock()
		if job.Status == StatusCancelled || job.Status == StatusDone {
			mw.JobsMu.Unlock()
			return
		}
		job.Status = StatusCancelled
		mw.JobsMu.Unlock()

		if job.Cancel != nil {
			job.Cancel()
		}

		go job.cleanupTouchedFiles()

		fyne.Do(func() {
			ui.StatusLbl.SetText(StatusCancelled)
			ui.BtnCancel.Disable()
			ui.BtnPauseResume.Disable()
		})
		mw.updateDownloadsBadge()
	})

	ui.BtnPauseResume = widget.NewButtonWithIcon("", theme.MediaPauseIcon(), func() {
		mw.JobsMu.Lock()
		switch job.Status {
		case StatusDownloading, StatusStarting:
			job.Status = StatusPaused
			cancelFn := job.Cancel
			mw.JobsMu.Unlock()

			if cancelFn != nil {
				cancelFn()
			}
			fyne.Do(func() {
				ui.BtnPauseResume.SetIcon(theme.MediaPlayIcon())
				ui.StatusLbl.SetText(StatusPaused)
			})

		case StatusPaused:
			job.Status = StatusQueued
			ctx, cancel := context.WithCancel(context.Background())
			job.Ctx = ctx
			job.Cancel = cancel
			mw.JobsMu.Unlock()

			fyne.Do(func() {
				ui.BtnPauseResume.SetIcon(theme.MediaPauseIcon())
				ui.StatusLbl.SetText(StatusQueued)
			})

			mw.ProcessMu.Lock()
			if mw.Debounce != nil {
				mw.Debounce.Reset(100 * time.Millisecond)
			}
			mw.ProcessMu.Unlock()

			mw.updateDownloadsBadge()
			go mw.processJob(job)

		default:
			mw.JobsMu.Unlock()
		}
	})
	ui.BtnPauseResume.Hide()

	ui.BtnOpenDir = widget.NewButtonWithIcon("", theme.FolderOpenIcon(), nil)
	ui.BtnOpenDir.Hide()

	var rightControls *fyne.Container
	if job.AllowPl {
		ui.ExpandBtn = widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), nil)
		rightControls = container.NewHBox(ui.BtnOpenDir, ui.ExpandBtn, ui.BtnPauseResume, ui.BtnCancel)
	} else {
		rightControls = container.NewHBox(ui.BtnOpenDir, ui.BtnPauseResume, ui.BtnCancel)
	}

	progRow := container.NewBorder(nil, nil, nil, rightControls, ui.ProgBar)
	mainContent := container.NewVBox(titleLbl, progRow, ui.StatusLbl)

	img := canvas.NewImageFromResource(theme.FileImageIcon())
	img.SetMinSize(fyne.NewSize(80, 45))
	img.FillMode = canvas.ImageFillContain

	if job.Thumbnail != nil {
		img.Resource = job.Thumbnail
	} else if job.AllowPl && len(job.PlaylistEntries) > 0 {
		firstID := ""
		for i, chk := range job.PlaylistChecks {
			if chk && i < len(job.PlaylistEntries) {
				firstID = job.PlaylistEntries[i].ID
				break
			}
		}
		if firstID == "" {
			firstID = job.PlaylistEntries[0].ID
		}

		cached, ok := thumbCache.Load(firstID)
		if ok && cached != "loading" {
			if res, isRes := cached.(fyne.Resource); isRes {
				img.Resource = res
			}
		} else if !ok {
			thumbCache.Store(firstID, "loading")
			go func(id string) {
				thumbURL := fmt.Sprintf("https://img.youtube.com/vi/%s/mqdefault.jpg", id)
				res := loadRemoteImageResource(thumbURL)
				if res != nil {
					thumbCache.Store(id, res)
					fyne.Do(func() {
						img.Resource = res
						img.Refresh()
					})
				} else {
					thumbCache.Delete(id)
				}
			}(firstID)
		}
	}

	left := container.NewHBox(img)

	if job.AllowPl {
		ui.ChildBox = container.NewVBox()
		ui.ChildBox.Hide()

		ui.ExpandBtn.OnTapped = func() {
			if ui.ChildBox.Hidden {
				ui.ChildBox.Show()
				ui.ExpandBtn.SetIcon(theme.MenuDropUpIcon())
			} else {
				ui.ChildBox.Hide()
				ui.ExpandBtn.SetIcon(theme.MenuDropDownIcon())
			}
		}

		ui.ChildProgs = make([]float64, len(job.PlaylistEntries))
		ui.ChildStats = make([]string, len(job.PlaylistEntries))

		var activeIndices []int
		for i, chk := range job.PlaylistChecks {
			if chk {
				ui.ChildStats[i] = StatusQueued
				activeIndices = append(activeIndices, i)
			}
		}
		ui.ActiveIndices = activeIndices

		ui.ChildList = widget.NewList(
			func() int { return len(activeIndices) },
			func() fyne.CanvasObject {
				cTitle := widget.NewLabel("Title")
				cTitle.Truncation = fyne.TextTruncateEllipsis
				cProg := widget.NewProgressBar()
				cProg.Min, cProg.Max = 0, 100
				cStat := widget.NewLabel(StatusQueued)
				cStat.TextStyle = fyne.TextStyle{Italic: true}

				childImg := canvas.NewImageFromResource(theme.FileImageIcon())
				childImg.SetMinSize(fyne.NewSize(40, 22))
				childImg.FillMode = canvas.ImageFillContain

				cProgRow := container.NewBorder(nil, nil, nil, cStat, cProg)
				cRow := container.NewBorder(nil, nil, nil, nil, container.NewVBox(cTitle, cProgRow))
				leftContent := container.NewHBox(widget.NewLabel("    └─ "), childImg)
				return container.NewBorder(nil, nil, leftContent, nil, cRow)
			},
			func(li widget.ListItemID, o fyne.CanvasObject) {
				realIdx := activeIndices[li]
				entry := job.PlaylistEntries[realIdx]

				c := o.(*fyne.Container)
				leftContent := c.Objects[1].(*fyne.Container)
				childImg := leftContent.Objects[1].(*canvas.Image)

				cRow := c.Objects[0].(*fyne.Container)
				vbox := cRow.Objects[0].(*fyne.Container)
				cTitle := vbox.Objects[0].(*widget.Label)
				cProgRow := vbox.Objects[1].(*fyne.Container)
				cStat := cProgRow.Objects[1].(*widget.Label)
				cProg := cProgRow.Objects[0].(*widget.ProgressBar)

				cTitle.SetText(fmt.Sprintf("%d. %s", realIdx+1, entry.Title))
				cStat.SetText(ui.ChildStats[realIdx])
				cProg.SetValue(ui.ChildProgs[realIdx])

				childImg.Resource = theme.FileImageIcon()
				cached, ok := thumbCache.Load(entry.ID)
				if ok && cached != "loading" {
					if res, isRes := cached.(fyne.Resource); isRes {
						childImg.Resource = res
					}
				} else if !ok {
					thumbCache.Store(entry.ID, "loading")
					thumbURL := fmt.Sprintf("https://img.youtube.com/vi/%s/mqdefault.jpg", entry.ID)

					go func(id, url string, listIdx widget.ListItemID) {
						res := loadRemoteImageResource(url)
						if res != nil {
							thumbCache.Store(id, res)
							fyne.Do(func() { ui.ChildList.RefreshItem(listIdx) })
						} else {
							thumbCache.Delete(id)
						}
					}(entry.ID, thumbURL, li)
				}
				childImg.Refresh()
			},
		)

		targetWidth := float32(800)
		if mw.Window != nil {
			targetWidth = mw.Window.Canvas().Size().Width - 250
			if targetWidth < 400 {
				targetWidth = 400
			}
		}
		listWrap := container.NewGridWrap(fyne.NewSize(targetWidth, 300), ui.ChildList)
		ui.ChildBox.Add(listWrap)
	}

	header := container.NewBorder(nil, nil, left, nil, mainContent)

	if job.AllowPl {
		ui.Root = container.NewVBox(header, ui.ChildBox, widget.NewSeparator())
	} else {
		ui.Root = container.NewVBox(header, widget.NewSeparator())
	}

	return ui
}
