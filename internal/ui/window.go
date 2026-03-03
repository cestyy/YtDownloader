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
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type MainWindow struct {
	App    fyne.App
	Window fyne.Window
	Cli    *ytdlp.Runner
	State  *State
	Logger *UILogger

	OutDirLabel *widget.Label
	UrlEntry    *widget.Entry
	Status      *widget.Label
	Busy        *widget.ProgressBarInfinite

	PreviewTitle *widget.Label
	PreviewImg   *canvas.Image

	FormatSelect  *widget.Select
	BrowserSelect *widget.Select
	ThemeSelect   *widget.Select
	ResSelect     *widget.Select
	ExtSelect     *widget.Select

	CheckSponsorBlock *widget.Check
	CheckRedownload   *widget.Check
	CheckEmbedMeta    *widget.Check
	NamingSelect      *widget.Select

	CookiesFileLabel *widget.Label
	CookiesFilePath  string
	BtnCookiesSelect *widget.Button
	BtnCookiesClear  *widget.Button

	CustomArgsEntry *widget.Entry

	ConcurrentSelect *widget.Select

	PlaylistTitle string

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
	BtnOpenFolder *widget.Button
	BtnChooseDir  *widget.Button
	BtnBest       *widget.Button
	BtnBestAudio  *widget.Button

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

	Jobs          []*DownloadJob
	JobsMu        sync.Mutex
	QueueBox      *fyne.Container
	BtnClearQueue *widget.Button
	DlSemaphore   chan struct{}

	Tabs         *container.AppTabs
	DownloadsTab *container.TabItem
	HistoryTab   *container.TabItem
	History      *DownloadHistory
}

func ShowMainWindow(a fyne.App, cli *ytdlp.Runner) fyne.Window {
	InitLang(a)

	savedTheme := a.Preferences().StringWithFallback("Theme", "Dark")
	a.Settings().SetTheme(&customTheme{themeName: savedTheme})

	w := a.NewWindow("YtDownloader")
	w.Resize(fyne.NewSize(1200, 740))
	w.SetFixedSize(true)

	concurrency := concurrencyFromPref(a.Preferences().StringWithFallback("Concurrency", "3"))

	mw := &MainWindow{
		App:          a,
		Window:       w,
		Cli:          cli,
		State:        &State{OutputDir: defaultDownloadsDir()},
		Logger:       NewUILogger(900),
		ProgLogEvery: 1500 * time.Millisecond,
		ProgStep:     1.0,
		LastProgPct:  -1,
		QueueBox:     container.NewVBox(),
		DlSemaphore:  make(chan struct{}, concurrency),
		History:      NewDownloadHistory(a),
	}

	mw.setupWidgets()
	mw.bindEvents()

	w.SetContent(mw.buildLayout())
	return w
}

func concurrencyFromPref(s string) int {
	switch s {
	case "1":
		return 1
	case "2":
		return 2
	case "4":
		return 4
	case "5":
		return 5
	default:
		return 3
	}
}

func (mw *MainWindow) setupWidgets() {
	mw.OutDirLabel = widget.NewLabel(mw.State.OutputDir)
	mw.OutDirLabel.Truncation = fyne.TextTruncateEllipsis
	mw.UrlEntry = widget.NewEntry()
	mw.UrlEntry.SetPlaceHolder(T("paste_url"))

	mw.Status = widget.NewLabel(T("ready"))
	mw.Busy = widget.NewProgressBarInfinite()
	mw.Busy.Hide()

	mw.PreviewTitle = widget.NewLabel("—")
	mw.PreviewTitle.Truncation = fyne.TextTruncateEllipsis
	mw.PreviewImg = canvas.NewImageFromResource(nil)
	mw.PreviewImg.FillMode = canvas.ImageFillContain
	mw.PreviewImg.SetMinSize(fyne.NewSize(560, 310))
	mw.PreviewImg.Hide()

	mw.FormatSelect = widget.NewSelect([]string{"mp4", "mkv", "webm", "avi", "flv", "mp3"}, nil)
	mw.FormatSelect.SetSelected("mp4")

	mw.BrowserSelect = widget.NewSelect([]string{"none", "chrome", "edge", "firefox", "opera", "brave", "safari", "vivaldi"}, nil)
	mw.BrowserSelect.SetSelected("none")

	mw.CookiesFileLabel = widget.NewLabel(T("no_cookies"))
	mw.CookiesFileLabel.Truncation = fyne.TextTruncateEllipsis
	mw.BtnCookiesSelect = widget.NewButton(T("select_cookies"), nil)
	mw.BtnCookiesClear = widget.NewButton(T("clear"), nil)
	mw.BtnCookiesClear.Disable()

	mw.CustomArgsEntry = widget.NewEntry()
	mw.CustomArgsEntry.SetPlaceHolder(T("custom_args_eg"))

	savedTheme := mw.App.Preferences().StringWithFallback("Theme", "Dark")
	mw.ThemeSelect = widget.NewSelect([]string{"Dark", "Light", "Pink", "Ocean"}, func(s string) {
		mw.App.Settings().SetTheme(&customTheme{themeName: s})
		mw.App.Preferences().SetString("Theme", s)
	})
	mw.ThemeSelect.SetSelected(savedTheme)

	savedConcurrent := mw.App.Preferences().StringWithFallback("Concurrency", "3")
	mw.ConcurrentSelect = widget.NewSelect([]string{"1", "2", "3", "4", "5"}, func(s string) {
		mw.App.Preferences().SetString("Concurrency", s)
		cap := concurrencyFromPref(s)
		mw.DlSemaphore = make(chan struct{}, cap)
	})
	mw.ConcurrentSelect.SetSelected(savedConcurrent)

	mw.BtnDownload = widget.NewButton(T("download_selected"), nil)
	mw.BtnDownload.Disable()

	mw.BtnBest = widget.NewButton(T("best_video"), nil)
	mw.BtnBest.Disable()

	mw.BtnBestAudio = widget.NewButton(T("best_audio"), nil)
	mw.BtnBestAudio.Disable()

	mw.BtnOpenFolder = widget.NewButton(T("open_folder"), nil)
	mw.BtnChooseDir = widget.NewButton(T("select_dir"), nil)

	mw.ResSelect = widget.NewSelect([]string{"All", "4K", "1440p", "1080p", "720p", "480p", "Audio Only"}, nil)
	mw.ResSelect.SetSelected("All")
	mw.ExtSelect = widget.NewSelect([]string{"All", "mp4", "webm", "m4a"}, nil)
	mw.ExtSelect.SetSelected("All")

	mw.ToolsStatus = widget.NewLabel(T("tools_ready"))
	mw.ToolsBusy = widget.NewProgressBar()
	mw.ToolsBusy.Min, mw.ToolsBusy.Max = 0, 100
	mw.ToolsBusy.SetValue(0)
	mw.ToolsBusy.Hide()

	mw.BtnToolsFolder = widget.NewButton(T("tools_folder"), nil)
	mw.BtnToolsUpdate = widget.NewButton(T("update_tools"), nil)
	mw.BtnToolsCancel = widget.NewButton(T("cancel_update"), nil)
	mw.BtnToolsCancel.Disable()

	mw.CheckSponsorBlock = widget.NewCheck(T("sponsorblock"), func(b bool) {
		mw.App.Preferences().SetBool("SponsorBlock", b)
	})
	mw.CheckSponsorBlock.SetChecked(mw.App.Preferences().BoolWithFallback("SponsorBlock", false))

	mw.CheckRedownload = widget.NewCheck(T("redownload"), func(b bool) {
		mw.App.Preferences().SetBool("Redownload", b)
	})
	mw.CheckRedownload.SetChecked(mw.App.Preferences().BoolWithFallback("Redownload", false))

	mw.CheckEmbedMeta = widget.NewCheck(T("embed_meta"), func(b bool) {
		mw.App.Preferences().SetBool("EmbedMeta", b)
	})
	mw.CheckEmbedMeta.SetChecked(mw.App.Preferences().BoolWithFallback("EmbedMeta", false))

	mw.SelectedCount = widget.NewLabel("")
	mw.SelectedCount.TextStyle = fyne.TextStyle{Bold: true}

	mw.BtnSelectAll = widget.NewButton(T("select_all"), func() {
		if len(mw.PlaylistChecks) == 0 {
			return
		}
		for i := range mw.PlaylistChecks {
			mw.PlaylistChecks[i] = true
		}
		mw.PlaylistList.Refresh()
		mw.updateSelectedCount()
	})

	mw.BtnUnselectAll = widget.NewButton(T("unselect_all"), func() {
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

	mw.BtnClearQueue = widget.NewButtonWithIcon(T("clear_finished"), theme.DeleteIcon(), func() {
		mw.JobsMu.Lock()
		var active []*DownloadJob
		var toRemove []fyne.CanvasObject
		var toClean []*DownloadJob

		for _, j := range mw.Jobs {
			if j.Status == StatusQueued || j.Status == StatusDownloading || j.Status == StatusStarting {
				active = append(active, j)
			} else {
				if j.UI != nil && j.UI.Root != nil {
					toRemove = append(toRemove, j.UI.Root)
				}
				if j.AllowPl {
					for _, entry := range j.PlaylistEntries {
						thumbCache.Delete(entry.ID)
					}
				}
				if j.Status != StatusDone {
					toClean = append(toClean, j)
				}
			}
		}
		mw.Jobs = active
		mw.JobsMu.Unlock()

		for _, j := range toClean {
			go j.cleanupTouchedFiles()
		}

		fyne.Do(func() {
			for _, o := range toRemove {
				mw.QueueBox.Remove(o)
			}
			mw.QueueBox.Refresh()
		})
		mw.updateDownloadsBadge()
	})

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
		mw.SelectedCount.SetText(fmt.Sprintf(T("selected_count"), count, len(mw.PlaylistEntries)))
	})
}
