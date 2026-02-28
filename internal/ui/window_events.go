package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"YtDownloader/internal/ytdlp"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (mw *MainWindow) bindEvents() {
	mw.UrlEntry.OnChanged = mw.onUrlChanged

	mw.ResSelect.OnChanged = mw.onFilterChanged
	mw.ExtSelect.OnChanged = mw.onFilterChanged

	mw.FormatList.OnSelected = mw.onFormatSelected

	mw.BtnOpenFolder.OnTapped = mw.onOpenFolder
	mw.BtnChooseDir.OnTapped = mw.onChooseDir
	mw.BtnCancel.OnTapped = mw.onCancel
	mw.BtnBest.OnTapped = mw.onDownloadBest
	mw.BtnDownload.OnTapped = mw.onDownloadSelected

	mw.BtnToolsUpdate.OnTapped = mw.onToolsUpdate
	mw.BtnToolsCancel.OnTapped = mw.onToolsCancel
	mw.BtnToolsFolder.OnTapped = mw.onToolsFolder
}

func (mw *MainWindow) setStatus(s string) {
	fyne.Do(func() { mw.Status.SetText(s) })
}

func (mw *MainWindow) setDownloadProgress(v float64) {
	fyne.Do(func() { mw.DownloadProgress.SetValue(v) })
}

func (mw *MainWindow) setProgressInfo(spd, eta string) {
	fyne.Do(func() {
		if spd == "" && eta == "" {
			mw.ProgressInfo.SetText("")
		} else {
			etaSec, _ := strconv.ParseFloat(eta, 64)
			mw.ProgressInfo.SetText(fmt.Sprintf("Speed: %s | ETA: %s", emptyToDash(spd), formatDuration(etaSec)))
		}
	})
}

func (mw *MainWindow) progressLine(p ytdlp.Progress) (string, float64) {
	pct := parsePercent(p.Pct)
	return fmt.Sprintf("[download] %s  %s  eta:%v", emptyToDash(p.Pct), emptyToDash(p.Spd), p.Eta), pct
}

func (mw *MainWindow) shouldLogProgress(pct float64) bool {
	if pct < 0 {
		return false
	}
	mw.ProgMu.Lock()
	defer mw.ProgMu.Unlock()
	now := time.Now()
	if mw.LastProgPct < 0 {
		mw.LastProgPct = pct
		mw.LastProgLog = now
		return true
	}
	if pct-mw.LastProgPct >= mw.ProgStep || now.Sub(mw.LastProgLog) >= mw.ProgLogEvery || pct == 100 {
		mw.LastProgPct = pct
		mw.LastProgLog = now
		return true
	}
	return false
}

func (mw *MainWindow) resetProgressThrottle() {
	mw.ProgMu.Lock()
	mw.LastProgPct = -1
	mw.LastProgLog = time.Time{}
	mw.ProgMu.Unlock()
}

func (mw *MainWindow) setDownloading(d bool) {
	mw.UpdMu.Lock()
	mw.Downloading = d
	mw.UpdMu.Unlock()

	fyne.Do(func() {
		if d {
			mw.FormatListTapBlock = true
			mw.BtnDownload.Disable()
			mw.BtnBest.Disable()
			mw.BtnChooseDir.Disable()
			mw.BtnCancel.Enable()
			mw.BtnOpenFolder.Disable()
			mw.UrlEntry.Disable()
			mw.BtnToolsUpdate.Disable()
			mw.BtnToolsFolder.Disable()
			mw.FormatSelect.Disable()
			mw.ThemeSelect.Disable()
			mw.BrowserSelect.Disable()
			mw.ResSelect.Disable()
			mw.ExtSelect.Disable()
			mw.NamingSelect.Disable()
			mw.CheckSponsorBlock.Disable()
			mw.BtnSelectAll.Disable()
			mw.BtnUnselectAll.Disable()
		} else {
			mw.FormatListTapBlock = false
			mw.BtnBest.Enable()
			mw.BtnChooseDir.Enable()
			mw.BtnCancel.Disable()
			mw.UrlEntry.Enable()
			mw.BtnToolsFolder.Enable()
			mw.FormatSelect.Enable()
			mw.ThemeSelect.Enable()
			mw.BrowserSelect.Enable()
			mw.ResSelect.Enable()
			mw.ExtSelect.Enable()
			mw.NamingSelect.Enable()
			mw.CheckSponsorBlock.Enable()
			mw.BtnSelectAll.Enable()
			mw.BtnUnselectAll.Enable()
			mw.UpdMu.Lock()
			if !mw.UpdRunning {
				mw.BtnToolsUpdate.Enable()
			}
			mw.UpdMu.Unlock()
			if strings.TrimSpace(mw.State.SelectedFmt) != "" || len(mw.PlaylistEntries) > 0 {
				mw.BtnDownload.Enable()
			}
		}
	})
}

func (mw *MainWindow) onUrlChanged(s string) {
	u := strings.TrimSpace(s)
	mw.ProcessMu.Lock()
	if mw.Debounce != nil {
		mw.Debounce.Stop()
		mw.Debounce = nil
	}
	mw.ProcessMu.Unlock()

	if u == "" {
		mw.ProcessMu.Lock()
		if mw.CancelJob != nil {
			mw.CancelJob()
			mw.CancelJob = nil
		}
		mw.ProcessMu.Unlock()
		mw.resetUIForEmpty()
		return
	}

	mw.ProcessMu.Lock()
	mw.Debounce = time.AfterFunc(450*time.Millisecond, func() { mw.startProcess(u) })
	mw.ProcessMu.Unlock()
}

func (mw *MainWindow) resetUIForEmpty() {
	fyne.Do(func() {
		mw.BtnDownload.Disable()
		mw.State.SelectedFmt = ""
		mw.FormatsAll = nil
		mw.FormatsView = nil
		mw.CurrentVideoDuration = 0

		mw.PlaylistEntries = nil
		mw.PlaylistChecks = nil

		if mw.RightPanelCards != nil && mw.PreviewContainer != nil {
			mw.RightPanelCards.Objects = []fyne.CanvasObject{mw.PreviewContainer}
			mw.RightPanelCards.Refresh()
		}

		mw.ResSelect.SetSelected("All")
		mw.ExtSelect.SetSelected("All")
		mw.FormatList.UnselectAll()
		mw.FormatList.Refresh()
		mw.PreviewTitle.SetText("—")
		mw.PreviewImg.Resource = nil
		mw.PreviewImg.Refresh()
		mw.Busy.Hide()
		mw.setStatus("Ready")
		mw.setDownloadProgress(0)
		mw.BtnOpenFolder.Disable()
	})
}

func (mw *MainWindow) startProcess(url string) {
	mw.ProcessMu.Lock()
	if mw.CancelJob != nil {
		mw.CancelJob()
		mw.CancelJob = nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	mw.CancelJob = cancel
	mw.ProcessMu.Unlock()

	fyne.Do(func() {
		mw.Busy.Show()
		mw.BtnDownload.Disable()
		mw.State.SelectedFmt = ""
		mw.setStatus("Loading…")
		mw.PreviewImg.Resource = nil
		mw.PreviewImg.Refresh()
	})

	mw.Logger.Dbg("--- PROCESS URL --- " + url)

	go func(myURL string, myCtx context.Context) {
		info, err := mw.Cli.FetchInfo(myCtx, myURL, mw.BrowserSelect.Selected)
		if myCtx.Err() != nil {
			return
		}

		if err != nil {
			mw.Logger.Err("Error: " + err.Error())
			fyne.Do(func() {
				if strings.TrimSpace(mw.UrlEntry.Text) == myURL {
					mw.Busy.Hide()
					mw.setStatus("Failed to load")
				}
			})
			return
		}

		if info.Type == "playlist" && len(info.Entries) > 0 {
			mw.PlaylistEntries = info.Entries
			mw.PlaylistChecks = make([]bool, len(info.Entries))
			for i := range mw.PlaylistChecks {
				mw.PlaylistChecks[i] = true
			}

			fyne.Do(func() {
				if strings.TrimSpace(mw.UrlEntry.Text) != myURL {
					return
				}

				if mw.RightPanelCards != nil && mw.PlaylistPanel != nil {
					mw.RightPanelCards.Objects = []fyne.CanvasObject{mw.PlaylistPanel}
					mw.RightPanelCards.Refresh()
					mw.PlaylistList.Refresh()
				}

				mw.setStatus(fmt.Sprintf("Playlist: %d videos", len(info.Entries)))
				mw.Busy.Hide()
				mw.BtnDownload.Enable()
				mw.BtnBest.Enable()
			})
		} else {
			mw.FormatsAll = info.Formats
			mw.CurrentVideoDuration = info.Duration

			fyne.Do(func() {
				if strings.TrimSpace(mw.UrlEntry.Text) != myURL {
					return
				}

				if mw.RightPanelCards != nil && mw.PreviewContainer != nil {
					mw.RightPanelCards.Objects = []fyne.CanvasObject{mw.PreviewContainer}
					mw.RightPanelCards.Refresh()
				}

				mw.applyFilter()
				mw.FormatList.UnselectAll()
				mw.FormatList.Refresh()
				mw.setStatus(fmt.Sprintf("Found formats: %d", len(mw.FormatsView)))
				mw.Busy.Hide()
				if info.Title != "" {
					mw.PreviewTitle.SetText(info.Title)
				} else {
					mw.PreviewTitle.SetText("—")
				}
				mw.BtnOpenFolder.Disable()
			})

			for _, thumbURL := range pickThumbCandidates(info) {
				if res := loadRemoteImageResource(thumbURL); res != nil {
					fyne.Do(func() {
						if strings.TrimSpace(mw.UrlEntry.Text) == myURL {
							mw.PreviewImg.Resource = res
							mw.PreviewImg.Refresh()
						}
					})
					break
				}
			}
		}
	}(url, ctx)
}

func (mw *MainWindow) onFilterChanged(_ string) {
	mw.applyFilter()
	fyne.Do(func() {
		mw.State.SelectedFmt = ""
		mw.BtnDownload.Disable()
		mw.FormatList.UnselectAll()
		mw.FormatList.Refresh()
	})
}

func (mw *MainWindow) applyFilter() {
	selRes := mw.ResSelect.Selected
	selExt := mw.ExtSelect.Selected

	if selRes == "All" && selExt == "All" {
		mw.FormatsView = mw.FormatsAll
		return
	}

	var out []ytdlp.Format
	for _, f := range mw.FormatsAll {
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

		extMatch := (selExt == "All") || (f.Ext == selExt)
		if resMatch && extMatch {
			out = append(out, f)
		}
	}
	mw.FormatsView = out
}

func (mw *MainWindow) onFormatSelected(id widget.ListItemID) {
	if mw.FormatListTapBlock {
		mw.FormatList.UnselectAll()
		return
	}
	if id >= 0 && id < len(mw.FormatsView) {
		mw.State.SelectedFmt = mw.FormatsView[id].FormatID
		mw.BtnDownload.Enable()
		mw.Logger.Dbg("Selected format: " + mw.State.SelectedFmt)
	}
}

func (mw *MainWindow) onOpenFolder() {
	mw.DlMu.Lock()
	target := mw.LastDownloadedFile
	mw.DlMu.Unlock()
	_ = showFileInFolder(target, mw.State.OutputDir)
}

func (mw *MainWindow) onChooseDir() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		mw.State.OutputDir = uri.Path()
		mw.OutDirLabel.SetText(mw.State.OutputDir)
		mw.DlMu.Lock()
		mw.LastDownloadedFile = ""
		mw.DlMu.Unlock()
		mw.BtnOpenFolder.Disable()
	}, mw.Window)
}

func (mw *MainWindow) onCancel() {
	mw.DlMu.Lock()
	if mw.DlCancel != nil {
		mw.Logger.Warn("Cancelling download…")
		mw.DlCancel()
		mw.DlCancel = nil
	}
	mw.DlMu.Unlock()
	mw.setProgressInfo("", "")
}

func (mw *MainWindow) handleDownloadResult(ctx context.Context, resultPath string, err error) {
	mw.DlMu.Lock()
	mw.DlCancel = nil
	if err == nil && resultPath != "" {
		mw.LastDownloadedFile = resultPath
	}
	mw.DlMu.Unlock()
	mw.setDownloading(false)

	if err != nil {
		mw.Logger.Err("Download error: " + err.Error())
		if ctx.Err() != nil {
			mw.setStatus("Cancelled")
		} else {
			mw.setStatus("Download failed")
		}
		return
	}
	mw.setDownloadProgress(100)
	mw.setProgressInfo("", "")
	mw.setStatus("Done ✅")
	fyne.Do(func() { mw.BtnOpenFolder.Enable() })
	playDoneSound()
}

func (mw *MainWindow) startDownloadRoutine(u, dlFormat string) {
	ctx, cancel := context.WithCancel(context.Background())
	mw.DlMu.Lock()
	mw.DlCancel = cancel
	mw.LastDownloadedFile = ""
	mw.DlMu.Unlock()

	selectedItemsStr := ""
	if len(mw.PlaylistEntries) > 0 {
		var selected []string
		for i, chk := range mw.PlaylistChecks {
			if chk {
				selected = append(selected, strconv.Itoa(i+1))
			}
		}
		selectedItemsStr = strings.Join(selected, ",")
		if selectedItemsStr == "" {
			dialog.ShowInformation("No videos selected", "Please select at least one video to download.", mw.Window)
			mw.setDownloading(false)

			mw.DlMu.Lock()
			mw.DlCancel = nil
			mw.DlMu.Unlock()
			return
		}
	}

	go func() {
		allowPl := len(mw.PlaylistEntries) > 0
		useSb := mw.CheckSponsorBlock.Checked
		naming := mw.NamingSelect.Selected

		resultPath, err := mw.Cli.Download(ctx, u, dlFormat, mw.State.OutputDir, mw.FormatSelect.Selected, mw.BrowserSelect.Selected, allowPl, useSb, naming, selectedItemsStr,
			func(p ytdlp.Progress) {
				line, pct := mw.progressLine(p)
				if pct >= 0 {
					mw.setDownloadProgress(pct)
					mw.setProgressInfo(p.Spd, p.Eta)
					if mw.shouldLogProgress(pct) {
						mw.Logger.Dbg(line)
					}
				}
			},
			func(line string) { mw.Logger.Info(line) },
		)
		mw.handleDownloadResult(ctx, resultPath, err)
	}()
}

func (mw *MainWindow) onDownloadBest() {
	u := strings.TrimSpace(mw.UrlEntry.Text)
	if u == "" || mw.State.OutputDir == "" {
		return
	}

	mw.setStatus("Downloading best…")
	mw.setDownloadProgress(0)
	mw.resetProgressThrottle()
	mw.Logger.Dbg("--- DOWNLOAD BEST ---")
	mw.setDownloading(true)

	dlFormat := "bestvideo+bestaudio/best"
	if mw.FormatSelect.Selected == "mp3" {
		dlFormat = "bestaudio/best"
	}
	mw.startDownloadRoutine(u, dlFormat)
}

func (mw *MainWindow) onDownloadSelected() {
	u := strings.TrimSpace(mw.UrlEntry.Text)
	if u == "" || (strings.TrimSpace(mw.State.SelectedFmt) == "" && len(mw.PlaylistEntries) == 0) || mw.State.OutputDir == "" {
		return
	}

	mw.setStatus("Downloading…")
	mw.setDownloadProgress(0)
	mw.resetProgressThrottle()
	mw.Logger.Dbg("--- DOWNLOAD ---")
	mw.setDownloading(true)

	dlFormat := mw.State.SelectedFmt
	if dlFormat == "" {
		dlFormat = "bestvideo+bestaudio/best"
	}

	if mw.FormatSelect.Selected != "mp3" {
		for _, f := range mw.FormatsAll {
			if f.FormatID == mw.State.SelectedFmt {
				if f.VCodec != "" && f.VCodec != "none" && (f.ACodec == "" || f.ACodec == "none") {
					dlFormat += "+bestaudio"
				}
				break
			}
		}
	}

	mw.startDownloadRoutine(u, dlFormat)
}
