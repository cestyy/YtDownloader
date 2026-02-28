package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"YtDownloader/internal/ytdlp"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func sanitizeFileName(name string) string {
	invalid := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range invalid {
		name = strings.ReplaceAll(name, char, "_")
	}
	return strings.TrimSpace(name)
}

func (mw *MainWindow) bindEvents() {
	mw.UrlEntry.OnChanged = mw.onUrlChanged
	mw.ResSelect.OnChanged = mw.onFilterChanged
	mw.ExtSelect.OnChanged = mw.onFilterChanged
	mw.FormatList.OnSelected = mw.onFormatSelected
	mw.BtnOpenFolder.OnTapped = mw.onOpenFolder
	mw.BtnChooseDir.OnTapped = mw.onChooseDir

	mw.BtnBest.OnTapped = mw.onDownloadBest
	mw.BtnDownload.OnTapped = mw.onDownloadSelected
	mw.BtnToolsUpdate.OnTapped = mw.onToolsUpdate
	mw.BtnToolsCancel.OnTapped = mw.onToolsCancel
	mw.BtnToolsFolder.OnTapped = mw.onToolsFolder

	mw.BtnCookiesSelect.OnTapped = mw.onSelectCookies
	mw.BtnCookiesClear.OnTapped = mw.onClearCookies
}

func (mw *MainWindow) updateDownloadsBadge() {
	mw.JobsMu.Lock()
	count := 0
	for _, j := range mw.Jobs {
		if j.Status == "Queued" || j.Status == "Downloading..." || j.Status == "Starting..." {
			count++
		}
	}
	mw.JobsMu.Unlock()

	fyne.Do(func() {
		if mw.DownloadsTab != nil && mw.Tabs != nil {
			if count > 0 {
				mw.DownloadsTab.Text = fmt.Sprintf("Downloads (%d)", count)
			} else {
				mw.DownloadsTab.Text = "Downloads"
			}
			mw.Tabs.Refresh()
		}
	})
}

func (mw *MainWindow) onSelectCookies() {
	dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		mw.CookiesFilePath = uc.URI().Path()
		mw.CookiesFileLabel.SetText(mw.CookiesFilePath)
		mw.BtnCookiesClear.Enable()
	}, mw.Window)
}

func (mw *MainWindow) onClearCookies() {
	mw.CookiesFilePath = ""
	mw.CookiesFileLabel.SetText("No cookies.txt selected")
	mw.BtnCookiesClear.Disable()
}

func (mw *MainWindow) setStatus(s string) {
	fyne.Do(func() { mw.Status.SetText(s) })
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
		mw.BtnBest.Disable()
		mw.State.SelectedFmt = ""
		mw.FormatsAll = nil
		mw.FormatsView = nil
		mw.CurrentVideoDuration = 0

		mw.PlaylistEntries = nil
		mw.PlaylistChecks = nil
		mw.PlaylistStatuses = nil
		mw.PlaylistTitle = ""
		mw.updateSelectedCount()

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
		mw.PreviewImg.Hide()
		mw.PreviewImg.Refresh()

		mw.Busy.Hide()
		mw.setStatus("Ready")
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
		mw.BtnBest.Disable()
		mw.State.SelectedFmt = ""
		mw.setStatus("Loading…")
		mw.PreviewImg.Resource = nil
		mw.PreviewImg.Hide()
		mw.PreviewImg.Refresh()
	})

	mw.Logger.Dbg("--- PROCESS URL --- " + url)

	go func(myURL string, myCtx context.Context) {
		info, err := mw.Cli.FetchInfo(myCtx, myURL, mw.BrowserSelect.Selected, mw.CookiesFilePath)
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
			mw.PlaylistTitle = info.Title
			mw.PlaylistEntries = info.Entries
			mw.PlaylistChecks = make([]bool, len(info.Entries))
			mw.PlaylistStatuses = make([]string, len(info.Entries))
			for i := range mw.PlaylistChecks {
				mw.PlaylistChecks[i] = true
			}

			mw.updateSelectedCount()

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
				mw.BtnBest.Enable()
			})

			for _, thumbURL := range pickThumbCandidates(info) {
				if res := loadRemoteImageResource(thumbURL); res != nil {
					fyne.Do(func() {
						if strings.TrimSpace(mw.UrlEntry.Text) == myURL {
							mw.PreviewImg.Resource = res
							mw.PreviewImg.Show()
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
	}, mw.Window)
}

func (mw *MainWindow) enqueueDownload(u, dlFormat string) {
	allowPl := len(mw.PlaylistEntries) > 0
	var entriesCopy []ytdlp.PlaylistEntry
	var checksCopy []bool
	var selectedItemsStr string

	title := mw.PreviewTitle.Text
	if title == "—" || title == "" {
		title = u
	}

	forceRedownload := mw.CheckRedownload.Checked

	if allowPl {
		title = mw.PlaylistTitle
		if title == "" {
			title = "Playlist"
		}

		entriesCopy = make([]ytdlp.PlaylistEntry, len(mw.PlaylistEntries))
		copy(entriesCopy, mw.PlaylistEntries)

		checksCopy = make([]bool, len(mw.PlaylistChecks))
		copy(checksCopy, mw.PlaylistChecks)

		var selected []string
		for i, chk := range mw.PlaylistChecks {
			if chk {
				if !forceRedownload && mw.PlaylistStatuses[i] == "Ready ✅" {
					checksCopy[i] = false
					continue
				}
				selected = append(selected, strconv.Itoa(i+1))
			}
		}
		selectedItemsStr = strings.Join(selected, ",")
		if selectedItemsStr == "" {
			dialog.ShowInformation("Skip", "All selected videos are already downloaded.", mw.Window)
			return
		}
		title += fmt.Sprintf(" (%d videos)", len(selected))

		fyne.Do(func() {
			for i, chk := range checksCopy {
				if chk {
					mw.PlaylistStatuses[i] = "Queued..."
				}
			}
			mw.PlaylistList.Refresh()
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	job := &DownloadJob{
		Title:            title,
		URL:              u,
		DlFormat:         dlFormat,
		OutputDir:        mw.State.OutputDir,
		FormatSelect:     mw.FormatSelect.Selected,
		BrowserSelect:    mw.BrowserSelect.Selected,
		CookiesFilePath:  mw.CookiesFilePath,
		AllowPl:          allowPl,
		UseSb:            mw.CheckSponsorBlock.Checked,
		Naming:           mw.NamingSelect.Selected,
		SelectedItemsStr: selectedItemsStr,
		PlaylistEntries:  entriesCopy,
		PlaylistChecks:   checksCopy,
		Status:           "Queued",
		Thumbnail:        mw.PreviewImg.Resource,
		Ctx:              ctx,
		Cancel:           cancel,
	}

	mw.JobsMu.Lock()
	mw.Jobs = append(mw.Jobs, job)
	mw.JobsMu.Unlock()

	job.UI = mw.buildJobUI(job)
	mw.QueueBox.Add(job.UI.Root)
	mw.QueueBox.Refresh()

	mw.App.SendNotification(fyne.NewNotification("Added to Queue", title))

	mw.updateDownloadsBadge()

	go mw.processJob(job)
}

func (mw *MainWindow) processJob(job *DownloadJob) {
	mw.DlSemaphore <- struct{}{}
	defer func() { <-mw.DlSemaphore }()

	mw.JobsMu.Lock()
	if job.Status == "Cancelled" {
		mw.JobsMu.Unlock()
		return
	}
	job.Status = "Downloading..."
	mw.JobsMu.Unlock()

	mw.UpdMu.Lock()
	mw.Downloading = true
	mw.UpdMu.Unlock()

	mw.updateDownloadsBadge()

	fyne.Do(func() {
		job.UI.StatusLbl.SetText("Starting...")
		if mw.UrlEntry.Text == job.URL {
			mw.Status.SetText("Downloading: " + job.Title)
		}
	})

	targetDir := job.OutputDir
	if job.AllowPl {
		safeName := sanitizeFileName(job.Title)
		if idx := strings.LastIndex(safeName, " ("); idx != -1 {
			safeName = safeName[:idx]
		}
		if safeName == "" {
			safeName = "Playlist"
		}
		targetDir = filepath.Join(job.OutputDir, safeName)
		os.MkdirAll(targetDir, 0755)
	}

	currentProcessingIndex := -1
	totalSelected := 0
	for _, chk := range job.PlaylistChecks {
		if chk {
			totalSelected++
		}
	}
	finishedCount := 0

	resultPath, err := mw.Cli.Download(job.Ctx, job.URL, job.DlFormat, targetDir, job.FormatSelect, job.BrowserSelect, job.CookiesFilePath, job.AllowPl, job.UseSb, job.Naming, job.SelectedItemsStr,
		func(p ytdlp.Progress) {
			pct := parsePercent(p.Pct)
			job.Progress = pct
			job.Speed = p.Spd
			job.ETA = p.Eta

			if mw.shouldLogProgress(pct) {
				mw.Logger.Dbg(fmt.Sprintf("[download] %s %s eta:%v", p.Pct, p.Spd, p.Eta))
			}

			fyne.Do(func() {
				etaSec, _ := strconv.ParseFloat(job.ETA, 64)
				etaStr := formatDuration(etaSec)

				if job.AllowPl {
					if currentProcessingIndex != -1 && currentProcessingIndex < len(job.UI.ChildProgs) {
						if job.UI.ChildProgs[currentProcessingIndex] != nil {
							job.UI.ChildProgs[currentProcessingIndex].SetValue(pct)
							job.UI.ChildStats[currentProcessingIndex].SetText(fmt.Sprintf("%s | ETA: %s", emptyToDash(job.Speed), etaStr))
						}
					}
					overallPct := (float64(finishedCount*100) + pct) / float64(totalSelected)
					job.UI.ProgBar.SetValue(overallPct)
					job.UI.StatusLbl.SetText(fmt.Sprintf("Downloading: %d / %d videos", finishedCount+1, totalSelected))
				} else {
					job.UI.ProgBar.SetValue(pct)
					job.UI.StatusLbl.SetText(fmt.Sprintf("Speed: %s | ETA: %s", emptyToDash(job.Speed), etaStr))
				}
			})
		},
		func(line string) {
			mw.Logger.Info(line)
			if job.AllowPl {
				for i, entry := range job.PlaylistEntries {
					if job.PlaylistChecks[i] && strings.Contains(line, entry.ID) {
						if currentProcessingIndex != i {
							prev := currentProcessingIndex
							currentProcessingIndex = i

							fyne.Do(func() {
								if prev != -1 && prev < len(job.UI.ChildProgs) {
									if job.UI.ChildProgs[prev] != nil {
										job.UI.ChildProgs[prev].SetValue(100)
										job.UI.ChildStats[prev].SetText("Ready ✅")
									}
									finishedCount++
								}
								if i < len(job.UI.ChildStats) && job.UI.ChildStats[i] != nil {
									job.UI.ChildStats[i].SetText("Downloading...")
								}
							})
						}
						break
					}
				}
			}
		},
	)

	if err == nil && job.AllowPl && currentProcessingIndex != -1 {
		fyne.Do(func() {
			if currentProcessingIndex < len(job.UI.ChildProgs) && job.UI.ChildProgs[currentProcessingIndex] != nil {
				job.UI.ChildProgs[currentProcessingIndex].SetValue(100)
				job.UI.ChildStats[currentProcessingIndex].SetText("Ready ✅")
			}
			job.UI.ProgBar.SetValue(100)
		})
	}

	if err != nil {
		if job.Ctx.Err() != nil {
			mw.JobsMu.Lock()
			job.Status = "Cancelled"
			mw.JobsMu.Unlock()
			fyne.Do(func() { job.UI.StatusLbl.SetText("Cancelled") })
		} else {
			mw.JobsMu.Lock()
			job.Status = "Error"
			mw.JobsMu.Unlock()
			fyne.Do(func() { job.UI.StatusLbl.SetText("Error") })
			mw.App.SendNotification(fyne.NewNotification("Download Failed", job.Title))
		}
	} else {
		mw.JobsMu.Lock()
		job.Status = "Done ✅"
		mw.JobsMu.Unlock()
		job.Progress = 100
		fyne.Do(func() {
			job.UI.StatusLbl.SetText("Done ✅")
			job.UI.ProgBar.SetValue(100)
			job.UI.BtnCancel.Disable()
		})
		mw.App.SendNotification(fyne.NewNotification("Download Complete ✅", job.Title))
		playDoneSound()

		mw.DlMu.Lock()
		if resultPath != "" {
			mw.LastDownloadedFile = resultPath
		}
		mw.DlMu.Unlock()
	}

	mw.updateDownloadsBadge()

	mw.JobsMu.Lock()
	anyActive := false
	for _, j := range mw.Jobs {
		if j.Status == "Downloading..." || j.Status == "Queued" || j.Status == "Starting..." {
			anyActive = true
			break
		}
	}
	mw.JobsMu.Unlock()

	if !anyActive {
		mw.UpdMu.Lock()
		mw.Downloading = false
		mw.UpdMu.Unlock()
	}
}

func (mw *MainWindow) onDownloadBest() {
	u := strings.TrimSpace(mw.UrlEntry.Text)
	if u == "" || mw.State.OutputDir == "" {
		return
	}

	mw.resetProgressThrottle()
	mw.Logger.Dbg("--- ENQUEUE BEST ---")

	dlFormat := "bestvideo+bestaudio/best"
	if mw.FormatSelect.Selected == "mp3" {
		dlFormat = "bestaudio/best"
	}
	mw.enqueueDownload(u, dlFormat)
}

func (mw *MainWindow) onDownloadSelected() {
	u := strings.TrimSpace(mw.UrlEntry.Text)
	if u == "" || (strings.TrimSpace(mw.State.SelectedFmt) == "" && len(mw.PlaylistEntries) == 0) || mw.State.OutputDir == "" {
		return
	}

	mw.resetProgressThrottle()
	mw.Logger.Dbg("--- ENQUEUE DOWNLOAD ---")

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

	mw.enqueueDownload(u, dlFormat)
}
