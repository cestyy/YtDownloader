package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"YtDownloader/internal/ytdlp"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type MainWindow struct {
	App    fyne.App
	Window fyne.Window
	Cli    *ytdlp.Runner
	State  *State
	Logger *UILogger

	OutDirLabel      *widget.Label
	UrlEntry         *widget.Entry
	Status           *widget.Label
	DownloadProgress *widget.ProgressBar
	ProgressInfo     *widget.Label
	Busy             *widget.ProgressBarInfinite

	PreviewTitle *widget.Label
	PreviewImg   *canvas.Image

	FormatSelect  *widget.Select
	BrowserSelect *widget.Select
	ThemeSelect   *widget.Select
	ResSelect     *widget.Select
	ExtSelect     *widget.Select

	CheckSponsorBlock *widget.Check
	CheckRedownload   *widget.Check
	NamingSelect      *widget.Select

	PlaylistTitle    string
	PlaylistEntries  []ytdlp.PlaylistEntry
	PlaylistChecks   []bool
	PlaylistStatuses []string
	PlaylistList     *widget.List
	BtnSelectAll     *widget.Button
	BtnUnselectAll   *widget.Button
	SelectedCount    *widget.Label
	PlaylistPanel    *fyne.Container

	PreviewContainer *fyne.Container
	RightPanelCards  *fyne.Container

	BtnDownload   *widget.Button
	BtnCancel     *widget.Button
	BtnOpenFolder *widget.Button
	BtnChooseDir  *widget.Button
	BtnBest       *widget.Button

	ToolsStatus    *widget.Label
	ToolsBusy      *widget.ProgressBar
	BtnToolsFolder *widget.Button
	BtnToolsUpdate *widget.Button
	BtnToolsCancel *widget.Button

	FormatList         *widget.List
	FormatListTapBlock bool

	FormatsAll           []ytdlp.Format
	FormatsView          []ytdlp.Format
	CurrentVideoDuration float64

	DlMu               sync.Mutex
	LastDownloadedFile string
	DlCancel           context.CancelFunc

	ProgMu       sync.Mutex
	LastProgPct  float64
	LastProgLog  time.Time
	ProgLogEvery time.Duration
	ProgStep     float64

	UpdMu       sync.Mutex
	UpdCancel   context.CancelFunc
	UpdRunning  bool
	Downloading bool

	ProcessMu sync.Mutex
	CancelJob context.CancelFunc
	Debounce  *time.Timer
}

func ShowMainWindow(a fyne.App, cli *ytdlp.Runner) fyne.Window {
	savedTheme := a.Preferences().StringWithFallback("Theme", "Dark")
	a.Settings().SetTheme(&customTheme{themeName: savedTheme})

	w := a.NewWindow("YtDownloader")
	w.Resize(fyne.NewSize(1200, 740))
	w.SetFixedSize(true)

	mw := &MainWindow{
		App:          a,
		Window:       w,
		Cli:          cli,
		State:        &State{OutputDir: defaultDownloadsDir()},
		Logger:       NewUILogger(900),
		ProgLogEvery: 1500 * time.Millisecond,
		ProgStep:     1.0,
		LastProgPct:  -1,
	}

	mw.setupWidgets()
	mw.bindEvents()

	w.SetContent(mw.buildLayout())
	return w
}

func (mw *MainWindow) setupWidgets() {
	mw.OutDirLabel = widget.NewLabel(mw.State.OutputDir)
	mw.UrlEntry = widget.NewEntry()
	mw.UrlEntry.SetPlaceHolder("Paste YouTube link…")

	mw.Status = widget.NewLabel("Ready")
	mw.DownloadProgress = widget.NewProgressBar()
	mw.DownloadProgress.Min, mw.DownloadProgress.Max = 0, 100
	mw.DownloadProgress.SetValue(0)

	mw.ProgressInfo = widget.NewLabel("")
	mw.ProgressInfo.Alignment = fyne.TextAlignTrailing
	mw.ProgressInfo.TextStyle = fyne.TextStyle{Italic: true}

	mw.Busy = widget.NewProgressBarInfinite()
	mw.Busy.Hide()

	mw.PreviewTitle = widget.NewLabel("—")
	mw.PreviewTitle.Wrapping = fyne.TextWrapWord
	mw.PreviewImg = canvas.NewImageFromResource(nil)
	mw.PreviewImg.FillMode = canvas.ImageFillContain
	mw.PreviewImg.SetMinSize(fyne.NewSize(560, 310))

	mw.FormatSelect = widget.NewSelect([]string{"mp4", "mkv", "webm", "avi", "flv", "mp3"}, nil)
	mw.FormatSelect.SetSelected("mp4")

	mw.BrowserSelect = widget.NewSelect([]string{"none", "chrome", "edge", "firefox", "opera", "brave", "safari", "vivaldi"}, nil)
	mw.BrowserSelect.SetSelected("none")

	savedTheme := mw.App.Preferences().StringWithFallback("Theme", "Dark")
	mw.ThemeSelect = widget.NewSelect([]string{"Dark", "Light", "Pink", "Ocean"}, func(s string) {
		mw.App.Settings().SetTheme(&customTheme{themeName: s})
		mw.App.Preferences().SetString("Theme", s)
	})
	mw.ThemeSelect.SetSelected(savedTheme)

	mw.BtnDownload = widget.NewButton("Download selected", nil)
	mw.BtnDownload.Disable()
	mw.BtnCancel = widget.NewButton("Cancel", nil)
	mw.BtnCancel.Disable()
	mw.BtnOpenFolder = widget.NewButton("Open folder", nil)
	mw.BtnOpenFolder.Disable()
	mw.BtnChooseDir = widget.NewButton("Select Directory", nil)
	mw.BtnBest = widget.NewButton("Download best", nil)

	mw.ResSelect = widget.NewSelect([]string{"All", "4K", "1440p", "1080p", "720p", "480p", "Audio Only"}, nil)
	mw.ResSelect.SetSelected("All")
	mw.ExtSelect = widget.NewSelect([]string{"All", "mp4", "webm", "m4a"}, nil)
	mw.ExtSelect.SetSelected("All")

	mw.ToolsStatus = widget.NewLabel("Tools: ready")
	mw.ToolsBusy = widget.NewProgressBar()
	mw.ToolsBusy.Min, mw.ToolsBusy.Max = 0, 100
	mw.ToolsBusy.SetValue(0)
	mw.ToolsBusy.Hide()

	mw.BtnToolsFolder = widget.NewButton("Tools folder", nil)
	mw.BtnToolsUpdate = widget.NewButton("Update tools", nil)
	mw.BtnToolsCancel = widget.NewButton("Cancel update", nil)
	mw.BtnToolsCancel.Disable()

	mw.CheckSponsorBlock = widget.NewCheck("Remove Sponsor (SponsorBlock)", func(b bool) {
		mw.App.Preferences().SetBool("SponsorBlock", b)
	})
	mw.CheckSponsorBlock.SetChecked(mw.App.Preferences().BoolWithFallback("SponsorBlock", false))

	mw.CheckRedownload = widget.NewCheck("Force redownload (if already Ready)", func(b bool) {
		mw.App.Preferences().SetBool("Redownload", b)
	})
	mw.CheckRedownload.SetChecked(mw.App.Preferences().BoolWithFallback("Redownload", false))

	mw.SelectedCount = widget.NewLabel("")
	mw.SelectedCount.TextStyle = fyne.TextStyle{Bold: true}

	mw.BtnSelectAll = widget.NewButton("Select All", func() {
		if len(mw.PlaylistChecks) == 0 {
			return
		}
		for i := range mw.PlaylistChecks {
			mw.PlaylistChecks[i] = true
		}
		mw.PlaylistList.Refresh()
		mw.updateSelectedCount()
	})

	mw.BtnUnselectAll = widget.NewButton("Unselect all", func() {
		if len(mw.PlaylistChecks) == 0 {
			return
		}
		for i := range mw.PlaylistChecks {
			mw.PlaylistChecks[i] = false
		}
		mw.PlaylistList.Refresh()
		mw.updateSelectedCount()
	})

	namingOption := []string{"Default (Title [ID])", "Author - Title", "Title (Year)"}
	mw.NamingSelect = widget.NewSelect(namingOption, func(s string) {
		mw.App.Preferences().SetString("Naming", s)
	})
	mw.NamingSelect.SetSelected(mw.App.Preferences().StringWithFallback("Naming", namingOption[0]))

	mw.setupFormatList()
	mw.setupPlaylistList()
}

func (mw *MainWindow) updateSelectedCount() {
	if mw.SelectedCount == nil {
		return
	}
	count := 0
	for _, c := range mw.PlaylistChecks {
		if c {
			count++
		}
	}
	fyne.Do(func() {
		mw.SelectedCount.SetText(fmt.Sprintf("Selected: %d / %d", count, len(mw.PlaylistEntries)))
	})
}

func (mw *MainWindow) setupFormatList() {
	mw.FormatList = widget.NewList(
		func() int { return len(mw.FormatsView) },
		func() fyne.CanvasObject {
			l1 := widget.NewLabel("")
			l1.TextStyle = fyne.TextStyle{Bold: true}
			l2 := widget.NewLabel("")
			l2.Wrapping = fyne.TextWrapWord
			return container.NewVBox(l1, l2)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			f := mw.FormatsView[i]
			v := o.(*fyne.Container)
			l1 := v.Objects[0].(*widget.Label)
			l2 := v.Objects[1].(*widget.Label)

			res := "Audio"
			if f.Height > 0 {
				res = fmt.Sprintf("%dp", f.Height)
			} else if f.Width > 0 {
				res = fmt.Sprintf("%dp", f.Width)
			}

			sizeBytes := f.Filesize
			if sizeBytes == 0 {
				sizeBytes = f.FilesizeApprox
			}
			l1.SetText(fmt.Sprintf("%s  •  %s  •  Time: %s", res, formatBytes(sizeBytes), formatDuration(mw.CurrentVideoDuration)))

			fps := ""
			if f.FPS > 0 {
				fps = fmt.Sprintf(" | fps: %v", f.FPS)
			}
			l2.SetText(fmt.Sprintf("fmt: %s | ext: %s%s | v: %s a: %s", emptyToDash(f.FormatID), f.Ext, fps, emptyToDash(f.VCodec), emptyToDash(f.ACodec)))
		},
	)
}

var thumbCache sync.Map

func (mw *MainWindow) setupPlaylistList() {
	mw.PlaylistList = widget.NewList(
		func() int { return len(mw.PlaylistEntries) },
		func() fyne.CanvasObject {
			chk := widget.NewCheck("", nil)

			img := canvas.NewImageFromResource(theme.FileImageIcon())
			img.SetMinSize(fyne.NewSize(80, 45))
			img.FillMode = canvas.ImageFillContain

			title := widget.NewLabel("Title")
			title.TextStyle.Bold = true
			title.Truncation = fyne.TextTruncateEllipsis

			author := widget.NewLabel("Author")
			author.TextStyle = fyne.TextStyle{Italic: true}

			statusLbl := widget.NewLabel("")
			statusLbl.TextStyle = fyne.TextStyle{Bold: true}

			bottomRow := container.NewHBox(author, layout.NewSpacer(), statusLbl)
			vbox := container.NewVBox(title, bottomRow)

			return container.NewBorder(nil, nil, container.NewHBox(chk, img), nil, vbox)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			entry := mw.PlaylistEntries[i]
			c := o.(*fyne.Container)

			leftHbox := c.Objects[1].(*fyne.Container)
			chk := leftHbox.Objects[0].(*widget.Check)
			img := leftHbox.Objects[1].(*canvas.Image)

			vbox := c.Objects[0].(*fyne.Container)
			title := vbox.Objects[0].(*widget.Label)

			bottomRow := vbox.Objects[1].(*fyne.Container)
			author := bottomRow.Objects[0].(*widget.Label)
			statusLbl := bottomRow.Objects[2].(*widget.Label)

			title.SetText(fmt.Sprintf("%d. %s", i+1, entry.Title))
			author.SetText(entry.Uploader)

			statusLbl.SetText(mw.PlaylistStatuses[i])

			chk.Checked = mw.PlaylistChecks[i]
			chk.OnChanged = func(b bool) {
				mw.PlaylistChecks[i] = b
				mw.updateSelectedCount()
			}
			chk.Refresh()

			img.Resource = theme.FileImageIcon()

			cached, ok := thumbCache.Load(entry.ID)
			if ok && cached != "loading" {
				if res, isRes := cached.(fyne.Resource); isRes {
					img.Resource = res
				}
			} else if !ok {
				thumbCache.Store(entry.ID, "loading")
				thumbURL := fmt.Sprintf("https://img.youtube.com/vi/%s/mqdefault.jpg", entry.ID)

				go func(id, url string, index widget.ListItemID) {
					res := loadRemoteImageResource(url)
					if res != nil {
						thumbCache.Store(id, res)
						fyne.Do(func() {
							if mw.PlaylistList != nil {
								mw.PlaylistList.RefreshItem(index)
							}
						})
					} else {
						thumbCache.Delete(id)
					}
				}(entry.ID, thumbURL, i)
			}

			img.Refresh()
		},
	)
}
