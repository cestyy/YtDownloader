package app

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"io"
	"net/http"
	neturl "net/url"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"YtDownloader/internal/bundled"
	"YtDownloader/internal/ytdlp"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

func ShowMainWindow(a fyne.App, cli *ytdlp.Runner) fyne.Window {
	const appName = "YtDownloader"

	a.Settings().SetTheme(&customTheme{dark: true})

	w := a.NewWindow("YtDownloader")
	w.Resize(fyne.NewSize(1200, 740))
	w.SetFixedSize(true)

	st := &State{OutputDir: defaultDownloadsDir()}
	outDirLabel := widget.NewLabel(st.OutputDir)

	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("Paste YouTube link…")

	status := widget.NewLabel("Ready")
	downloadProgress := widget.NewProgressBar()
	downloadProgress.Min, downloadProgress.Max = 0, 100
	downloadProgress.SetValue(0)

	busy := widget.NewProgressBarInfinite()
	busy.Hide()

	logger := NewUILogger(900)

	previewTitle := widget.NewLabel("—")
	previewTitle.Wrapping = fyne.TextWrapWord

	previewImg := canvas.NewImageFromResource(nil)
	previewImg.FillMode = canvas.ImageFillContain
	previewImg.SetMinSize(fyne.NewSize(560, 310))

	formatsAll := make([]ytdlp.Format, 0)
	formatsView := make([]ytdlp.Format, 0)
	var currentVideoDuration float64

	mergeFormat := "mp4"
	formatSelect := widget.NewSelect([]string{"mp4", "mkv", "webm", "avi", "flv", "mp3"}, func(s string) {
		mergeFormat = s
	})
	formatSelect.SetSelected("mp4")

	selectedBrowser := "none"
	browserSelect := widget.NewSelect([]string{"none", "chrome", "edge", "firefox", "opera", "brave", "safari", "vivaldi"}, func(s string) {
		selectedBrowser = s
	})
	browserSelect.SetSelected("none")

	themeSelect := widget.NewSelect([]string{"Dark", "Light"}, func(s string) {
		a.Settings().SetTheme(&customTheme{dark: s == "Dark"})
	})
	themeSelect.SetSelected("Dark")

	btnDownload := widget.NewButton("Download selected", func() {})
	btnDownload.Disable()

	btnCancel := widget.NewButton("Cancel", func() {})
	btnCancel.Disable()

	var (
		dlMu               sync.Mutex
		lastDownloadedFile string
	)

	btnOpenFolder := widget.NewButton("Open folder", func() {
		dlMu.Lock()
		target := lastDownloadedFile
		dlMu.Unlock()
		_ = showFileInFolder(target, st.OutputDir)
	})
	btnOpenFolder.Disable()

	btnChooseDir := widget.NewButton("Select Directory", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			st.OutputDir = uri.Path()
			outDirLabel.SetText(st.OutputDir)

			dlMu.Lock()
			lastDownloadedFile = ""
			dlMu.Unlock()
			btnOpenFolder.Disable()
		}, w)
	})

	btnBest := widget.NewButton("Download best", func() {})

	setStatus := func(s string) {
		fyne.Do(func() { status.SetText(s) })
	}
	setDownloadProgress := func(v float64) {
		fyne.Do(func() { downloadProgress.SetValue(v) })
	}

	progressLine := func(p ytdlp.Progress) (string, float64) {
		pct := parsePercent(p.Pct)
		line := fmt.Sprintf("[download] %s  %s  eta:%v",
			emptyToDash(p.Pct),
			emptyToDash(p.Spd),
			p.Eta,
		)
		return line, pct
	}

	var (
		dlCancel context.CancelFunc

		progMu       sync.Mutex
		lastProgPct  float64 = -1
		lastProgLog  time.Time
		progLogEvery = 1500 * time.Millisecond
		progStep     = 1.0
	)

	shouldLogProgress := func(pct float64) bool {
		if pct < 0 {
			return false
		}
		progMu.Lock()
		defer progMu.Unlock()
		now := time.Now()
		if lastProgPct < 0 {
			lastProgPct = pct
			lastProgLog = now
			return true
		}
		if pct-lastProgPct >= progStep || now.Sub(lastProgLog) >= progLogEvery || pct == 100 {
			lastProgPct = pct
			lastProgLog = now
			return true
		}
		return false
	}

	resetProgressThrottle := func() {
		progMu.Lock()
		lastProgPct = -1
		lastProgLog = time.Time{}
		progMu.Unlock()
	}

	var formatList *widget.List
	var formatListTapBlock bool
	var applyFilter func()

	// ДВА ВЫПАДАЮЩИХ СПИСКА ДЛЯ ФИЛЬТРОВ
	resSelect := widget.NewSelect([]string{"All", "4K", "1440p", "1080p", "720p", "480p", "Audio Only"}, func(s string) {
		if applyFilter != nil {
			applyFilter()
			fyne.Do(func() {
				st.SelectedFmt = ""
				btnDownload.Disable()
				formatList.UnselectAll()
				formatList.Refresh()
			})
		}
	})
	resSelect.SetSelected("All")

	extSelect := widget.NewSelect([]string{"All", "mp4", "webm", "m4a"}, func(s string) {
		if applyFilter != nil {
			applyFilter()
			fyne.Do(func() {
				st.SelectedFmt = ""
				btnDownload.Disable()
				formatList.UnselectAll()
				formatList.Refresh()
			})
		}
	})
	extSelect.SetSelected("All")

	filterUI := container.NewVBox(
		widget.NewLabelWithStyle("Filters:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("Resolution (Quality):"), resSelect),
			container.NewVBox(widget.NewLabel("Format (Extension):"), extSelect),
		),
	)

	var (
		updMu       sync.Mutex
		updCancel   context.CancelFunc
		updRunning  bool
		downloading bool
	)

	toolsStatus := widget.NewLabel("Tools: ready")
	toolsBusy := widget.NewProgressBarInfinite()
	toolsBusy.Hide()

	btnToolsFolder := widget.NewButton("Tools folder", func() {
		t, err := bundled.AppPathsForUI(appName)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		_ = openFolderDirect(t)
	})
	btnToolsUpdate := widget.NewButton("Update tools", func() {})
	btnToolsCancel := widget.NewButton("Cancel update", func() {})

	btnToolsCancel.Disable()

	setToolsBusy := func(on bool, msg string) {
		fyne.Do(func() {
			if msg != "" {
				toolsStatus.SetText(msg)
			}
			if on {
				toolsBusy.Show()
				toolsBusy.Start()
				btnToolsUpdate.Disable()
				btnToolsCancel.Enable()
			} else {
				toolsBusy.Stop()
				toolsBusy.Hide()
				btnToolsUpdate.Enable()
				btnToolsCancel.Disable()
			}
		})
	}

	startToolsUpdate := func() {
		updMu.Lock()
		if updRunning || downloading {
			updMu.Unlock()
			return
		}
		updRunning = true
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		updCancel = cancel
		updMu.Unlock()

		setToolsBusy(true, "Tools: updating…")
		logger.Dbg("--- TOOLS UPDATE ---")

		go func() {
			tools, err := bundled.EnsureToolsAutoUpdate(ctx, appName, true)

			updMu.Lock()
			updRunning = false
			c := updCancel
			updCancel = nil
			updMu.Unlock()
			if c != nil {
				c()
			}

			if err != nil {
				setToolsBusy(false, "Tools: update failed")
				fyne.Do(func() { dialog.ShowError(err, w) })
				logger.Warn("Tools update failed: " + err.Error())
				return
			}

			updMu.Lock()
			if downloading {
				updMu.Unlock()
				setToolsBusy(false, "Tools: updated (will apply after download)")
				logger.Info("Tools updated, will apply after current download")
				return
			}
			updMu.Unlock()

			cli.YtDlpPath = tools.YtDlpPath
			cli.FFmpegDir = tools.BinDir

			setToolsBusy(false, "Tools: updated")
			logger.Info("Tools updated")
		}()
	}

	btnToolsUpdate.OnTapped = func() { startToolsUpdate() }

	btnToolsCancel.OnTapped = func() {
		updMu.Lock()
		c := updCancel
		updCancel = nil
		updMu.Unlock()
		if c != nil {
			c()
			setToolsBusy(false, "Tools: update cancelled")
			logger.Warn("Tools update cancelled")
		}
	}

	setDownloading := func(d bool) {
		updMu.Lock()
		downloading = d
		updMu.Unlock()

		fyne.Do(func() {
			if d {
				formatListTapBlock = true
				btnDownload.Disable()
				btnBest.Disable()
				btnChooseDir.Disable()
				btnCancel.Enable()
				btnOpenFolder.Disable()
				urlEntry.Disable()
				btnToolsUpdate.Disable()
				btnToolsFolder.Disable()
				formatSelect.Disable()
				themeSelect.Disable()
				browserSelect.Disable()
				resSelect.Disable()
				extSelect.Disable()
			} else {
				formatListTapBlock = false
				btnBest.Enable()
				btnChooseDir.Enable()
				btnCancel.Disable()
				urlEntry.Enable()
				btnToolsFolder.Enable()
				formatSelect.Enable()
				themeSelect.Enable()
				browserSelect.Enable()
				resSelect.Enable()
				extSelect.Enable()
				updMu.Lock()
				running := updRunning
				updMu.Unlock()
				if !running {
					btnToolsUpdate.Enable()
				}
				if strings.TrimSpace(st.SelectedFmt) != "" {
					btnDownload.Enable()
				}
			}
		})
	}

	btnCancel.OnTapped = func() {
		dlMu.Lock()
		c := dlCancel
		dlCancel = nil
		dlMu.Unlock()
		if c != nil {
			logger.Warn("Cancelling download…")
			c()
		}
	}

	btnBest.OnTapped = func() {
		u := strings.TrimSpace(urlEntry.Text)
		if u == "" {
			dialog.ShowInformation("No URL", "Paste a YouTube link.", w)
			return
		}
		if st.OutputDir == "" {
			dialog.ShowInformation("No output dir", "Select output directory.", w)
			return
		}

		setStatus("Downloading best…")
		setDownloadProgress(0)
		resetProgressThrottle()
		logger.Dbg("--- DOWNLOAD BEST ---")
		setDownloading(true)

		ctx, cancel := context.WithCancel(context.Background())
		dlMu.Lock()
		dlCancel = cancel
		lastDownloadedFile = ""
		dlMu.Unlock()

		dlFormat := "bestvideo+bestaudio/best"
		if mergeFormat == "mp3" {
			dlFormat = "bestaudio/best"
		}

		go func(url string) {
			resultPath, err := cli.Download(ctx, url, dlFormat, st.OutputDir, mergeFormat, selectedBrowser,
				func(p ytdlp.Progress) {
					_, pct := progressLine(p)
					if pct >= 0 {
						setDownloadProgress(pct)
						if shouldLogProgress(pct) {
							line, _ := progressLine(p)
							logger.Dbg(line)
						}
					}
				},
				func(line string) { logger.Info(line) },
			)

			dlMu.Lock()
			dlCancel = nil
			if err == nil && resultPath != "" {
				lastDownloadedFile = resultPath
			}
			dlMu.Unlock()
			setDownloading(false)

			if err != nil {
				logger.Err("Download error: " + err.Error())
				if ctx.Err() != nil {
					setStatus("Cancelled")
				} else {
					setStatus("Download failed")
				}
				return
			}
			setDownloadProgress(100)
			setStatus("Done ✅")
			fyne.Do(func() { btnOpenFolder.Enable() })
			playDoneSound()
		}(u)
	}

	applyFilter = func() {
		selRes := resSelect.Selected
		selExt := extSelect.Selected

		if selRes == "All" && selExt == "All" {
			formatsView = formatsAll
			return
		}

		out := make([]ytdlp.Format, 0, len(formatsAll))
		for _, f := range formatsAll {
			resMatch := false
			switch selRes {
			case "All":
				resMatch = true
			case "4K":
				resMatch = f.Height >= 2160
			case "1440p":
				resMatch = f.Height >= 1440 && f.Height < 2160
			case "1080p":
				resMatch = f.Height >= 1080 && f.Height < 1440
			case "720p":
				resMatch = f.Height >= 720 && f.Height < 1080
			case "480p":
				resMatch = f.Height > 0 && f.Height < 720
			case "Audio Only":
				resMatch = f.VCodec == "" || f.VCodec == "none"
			}

			extMatch := false
			switch selExt {
			case "All":
				extMatch = true
			default:
				extMatch = f.Ext == selExt
			}

			if resMatch && extMatch {
				out = append(out, f)
			}
		}
		formatsView = out
	}

	formatList = widget.NewList(
		func() int { return len(formatsView) },
		func() fyne.CanvasObject {
			l1 := widget.NewLabel("")
			l1.TextStyle = fyne.TextStyle{Bold: true}
			l2 := widget.NewLabel("")
			l2.Wrapping = fyne.TextWrapWord
			return container.NewVBox(l1, l2)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			f := formatsView[i]
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
			sizeStr := formatBytes(sizeBytes)
			durStr := formatDuration(currentVideoDuration)

			l1.SetText(fmt.Sprintf("%s  •  %s  •  Time: %s", res, sizeStr, durStr))

			fps := ""
			if f.FPS > 0 {
				fps = fmt.Sprintf(" | fps: %v", f.FPS)
			}
			l2.SetText(fmt.Sprintf("fmt: %s | ext: %s%s | v: %s a: %s", emptyToDash(f.FormatID), f.Ext, fps, emptyToDash(f.VCodec), emptyToDash(f.ACodec)))
		},
	)

	formatList.OnSelected = func(id widget.ListItemID) {
		if formatListTapBlock {
			formatList.UnselectAll()
			return
		}
		if id >= 0 && id < len(formatsView) {
			st.SelectedFmt = formatsView[id].FormatID
			btnDownload.Enable()
			logger.Dbg("Selected format: " + st.SelectedFmt)
		}
	}

	btnDownload.OnTapped = func() {
		u := strings.TrimSpace(urlEntry.Text)
		if u == "" {
			dialog.ShowInformation("No URL", "Paste a YouTube link.", w)
			return
		}
		if strings.TrimSpace(st.SelectedFmt) == "" {
			dialog.ShowInformation("No format", "Select a format from the list.", w)
			return
		}
		if st.OutputDir == "" {
			dialog.ShowInformation("No output dir", "Select output directory.", w)
			return
		}

		setStatus("Downloading…")
		setDownloadProgress(0)
		resetProgressThrottle()
		logger.Dbg("--- DOWNLOAD ---")
		logger.Dbg("format_id=" + st.SelectedFmt)
		setDownloading(true)

		ctx, cancel := context.WithCancel(context.Background())
		dlMu.Lock()
		dlCancel = cancel
		lastDownloadedFile = ""
		dlMu.Unlock()

		dlFormat := st.SelectedFmt
		if mergeFormat != "mp3" {
			for _, f := range formatsAll {
				if f.FormatID == st.SelectedFmt {
					hasV := f.VCodec != "" && f.VCodec != "none"
					hasA := f.ACodec != "" && f.ACodec != "none"
					if hasV && !hasA {
						dlFormat = st.SelectedFmt + "+bestaudio"
					}
					break
				}
			}
		}

		go func(url, fmtID string) {
			resultPath, err := cli.Download(ctx, url, fmtID, st.OutputDir, mergeFormat, selectedBrowser,
				func(p ytdlp.Progress) {
					_, pct := progressLine(p)
					if pct >= 0 {
						setDownloadProgress(pct)
						if shouldLogProgress(pct) {
							line, _ := progressLine(p)
							logger.Dbg(line)
						}
					}
				},
				func(line string) { logger.Info(line) },
			)

			dlMu.Lock()
			dlCancel = nil
			if err == nil && resultPath != "" {
				lastDownloadedFile = resultPath
			}
			dlMu.Unlock()
			setDownloading(false)

			if err != nil {
				logger.Err("Download error: " + err.Error())
				if ctx.Err() != nil {
					setStatus("Cancelled")
				} else {
					setStatus("Download failed")
				}
				return
			}
			setDownloadProgress(100)
			setStatus("Done ✅")
			fyne.Do(func() { btnOpenFolder.Enable() })
			playDoneSound()
		}(u, dlFormat)
	}

	topRow := container.NewBorder(nil, nil, widget.NewLabel("URL:"), nil, urlEntry)

	dirRow := container.NewHBox(
		widget.NewLabel("Save to:"),
		outDirLabel,
		layout.NewSpacer(),
		btnChooseDir,
		btnOpenFolder,
	)

	btnRow := container.NewHBox(btnDownload, btnBest, btnCancel, layout.NewSpacer())

	leftTop := container.NewVBox(
		topRow,
		dirRow,
		btnRow,
		widget.NewSeparator(),
		filterUI,
		busy,
	)

	left := container.NewBorder(leftTop, nil, nil, nil, container.NewMax(formatList))

	rightTop := container.NewVBox(
		widget.NewLabel("Status:"),
		status,
		downloadProgress,
		widget.NewSeparator(),
		widget.NewLabel("Preview:"),
		previewTitle,
		previewImg,
	)

	right := container.NewBorder(rightTop, nil, nil, nil, nil)

	mainSplit := container.NewHSplit(left, right)
	mainSplit.Offset = 0.50

	settingsView := container.NewVBox(
		widget.NewLabelWithStyle("Application Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Appearance"),
		themeSelect,
		widget.NewSeparator(),
		widget.NewLabel("Output Format (video/audio)"),
		formatSelect,
		widget.NewSeparator(),
		widget.NewLabel("Bypass YouTube Bot Check"),
		browserSelect,
		widget.NewSeparator(),
		widget.NewLabel("Tools (yt-dlp & ffmpeg)"),
		container.NewHBox(toolsStatus, toolsBusy),
		container.NewHBox(btnToolsUpdate, btnToolsCancel, btnToolsFolder),
		widget.NewSeparator(),
		widget.NewLabel("System Logs"),
		logger.Controls(w),
	)

	settingsLayout := container.NewBorder(settingsView, nil, nil, nil, container.NewMax(logger.Widget()))

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Main", theme.HomeIcon(), mainSplit),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), settingsLayout),
	)
	tabs.SetTabLocation(container.TabLocationLeading)

	w.SetContent(tabs)

	var (
		mu        sync.Mutex
		cancelJob context.CancelFunc
		debounce  *time.Timer
	)

	resetUIForEmpty := func() {
		fyne.Do(func() {
			btnDownload.Disable()
			st.SelectedFmt = ""
			formatsAll = nil
			formatsView = nil
			currentVideoDuration = 0
			resSelect.SetSelected("All")
			extSelect.SetSelected("All")
			formatList.UnselectAll()
			formatList.Refresh()
			previewTitle.SetText("—")
			previewImg.Resource = nil
			previewImg.Refresh()
			busy.Hide()
			setStatus("Ready")
			setDownloadProgress(0)
			btnOpenFolder.Disable()
		})
	}

	startProcess := func(url string) {
		mu.Lock()
		if cancelJob != nil {
			cancelJob()
			cancelJob = nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancelJob = cancel
		mu.Unlock()

		fyne.Do(func() {
			busy.Show()
			btnDownload.Disable()
			st.SelectedFmt = ""
			setStatus("Loading…")
			previewImg.Resource = nil
			previewImg.Refresh()
		})
		logger.Dbg("--- PROCESS URL --- " + url)

		go func(myURL string, myCtx context.Context) {
			info, err := cli.FetchInfo(myCtx, myURL, selectedBrowser)
			if myCtx.Err() != nil {
				return
			}

			if err != nil {
				logger.Err("Error: " + err.Error())
				fyne.Do(func() {
					if strings.TrimSpace(urlEntry.Text) == myURL {
						busy.Hide()
						setStatus("Failed to load")
					}
				})
				return
			}

			formatsAll = info.Formats
			currentVideoDuration = info.Duration

			fyne.Do(func() {
				if strings.TrimSpace(urlEntry.Text) != myURL {
					return
				}
				applyFilter()
				formatList.UnselectAll()
				formatList.Refresh()
				setStatus(fmt.Sprintf("Found formats: %d", len(formatsView)))
				busy.Hide()
				if info.Title != "" {
					previewTitle.SetText(info.Title)
				} else {
					previewTitle.SetText("—")
				}
				btnOpenFolder.Disable()
			})

			loaded := false
			for _, thumbURL := range pickThumbCandidates(info) {
				res := loadRemoteImageResource(thumbURL)
				if res == nil {
					continue
				}
				fyne.Do(func() {
					if strings.TrimSpace(urlEntry.Text) != myURL {
						return
					}
					previewImg.Resource = res
					previewImg.Refresh()
				})
				loaded = true
				break
			}
			if !loaded {
				logger.Warn("Preview not loaded")
			}
		}(url, ctx)
	}

	urlEntry.OnChanged = func(s string) {
		u := strings.TrimSpace(s)

		mu.Lock()
		if debounce != nil {
			debounce.Stop()
			debounce = nil
		}
		mu.Unlock()

		if u == "" {
			mu.Lock()
			if cancelJob != nil {
				cancelJob()
				cancelJob = nil
			}
			mu.Unlock()
			resetUIForEmpty()
			return
		}

		mu.Lock()
		debounce = time.AfterFunc(450*time.Millisecond, func() { startProcess(u) })
		mu.Unlock()
	}

	return w
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

func pickThumbCandidates(info *ytdlp.VideoInfo) []string {
	type cand struct {
		url string
		px  int
	}

	isLikelyImage := func(u string) bool {
		u = strings.ToLower(u)
		return strings.Contains(u, ".jpg") || strings.Contains(u, ".jpeg") ||
			strings.Contains(u, ".png") || strings.Contains(u, ".webp")
	}

	addBoth := func(out *[]cand, u string, px int) {
		u = strings.TrimSpace(u)
		if u == "" {
			return
		}
		*out = append(*out, cand{url: u, px: px})
		lu := strings.ToLower(u)
		if strings.Contains(lu, ".webp") {
			*out = append(*out, cand{url: strings.ReplaceAll(u, ".webp", ".jpg"), px: px - 1})
			*out = append(*out, cand{url: strings.ReplaceAll(u, ".webp", ".jpeg"), px: px - 2})
		}
	}

	list := make([]cand, 0, len(info.Thumbnails)+4)
	for _, t := range info.Thumbnails {
		u := strings.TrimSpace(t.URL)
		if u == "" {
			continue
		}
		ext := strings.ToLower(strings.TrimSpace(t.Ext))
		if ext == "jpg" || ext == "jpeg" || ext == "png" || ext == "webp" || isLikelyImage(u) {
			px := 0
			if t.Width > 0 && t.Height > 0 {
				px = t.Width * t.Height
			}
			addBoth(&list, u, px)
		}
	}

	if u := strings.TrimSpace(info.Thumbnail); u != "" && isLikelyImage(u) {
		addBoth(&list, u, 0)
	}

	for i := 0; i < len(list); i++ {
		best := i
		for j := i + 1; j < len(list); j++ {
			if list[j].px > list[best].px {
				best = j
			}
		}
		list[i], list[best] = list[best], list[i]
	}

	seen := make(map[string]struct{}, len(list))
	out := make([]string, 0, len(list))
	for _, c := range list {
		if c.url == "" {
			continue
		}
		if _, ok := seen[c.url]; ok {
			continue
		}
		seen[c.url] = struct{}{}
		out = append(out, c.url)
	}
	return out
}

func loadRemoteImageResource(url string) fyne.Resource {
	client := &http.Client{Timeout: 12 * time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil || len(b) == 0 {
		return nil
	}

	name := "thumb"
	ext := ".jpg"
	if u, err := neturl.Parse(url); err == nil {
		if e := path.Ext(u.Path); e != "" {
			ext = e
		}
	}
	name += ext

	return fyne.NewStaticResource(name, bytes.Clone(b))
}

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

type customTheme struct {
	dark bool
}

func (t *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if t.dark {
		switch name {
		case theme.ColorNameBackground:
			return color.RGBA{R: 15, G: 15, B: 20, A: 255}
		case theme.ColorNameInputBackground, theme.ColorNameButton:
			return color.RGBA{R: 28, G: 32, B: 40, A: 255}
		case theme.ColorNamePrimary:
			return color.RGBA{R: 30, G: 144, B: 255, A: 255}
		case theme.ColorNameForeground:
			return color.RGBA{R: 240, G: 240, B: 240, A: 255}
		}
	} else {
		switch name {
		case theme.ColorNameBackground:
			return color.RGBA{R: 250, G: 250, B: 250, A: 255}
		case theme.ColorNameInputBackground, theme.ColorNameButton:
			return color.White
		case theme.ColorNamePrimary:
			return color.RGBA{R: 220, G: 53, B: 69, A: 255}
		case theme.ColorNameForeground:
			return color.RGBA{R: 20, G: 20, B: 20, A: 255}
		}
	}

	v := theme.VariantLight
	if t.dark {
		v = theme.VariantDark
	}
	return theme.DefaultTheme().Color(name, v)
}

func (t *customTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *customTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *customTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
