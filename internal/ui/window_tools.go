package app

import (
	"context"
	"time"

	"YtDownloader/internal/bundled"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func (mw *MainWindow) onToolsFolder() {
	t, err := bundled.AppPathsForUI("YtDownloader")
	if err != nil {
		dialog.ShowError(err, mw.Window)
		return
	}
	_ = openFolderDirect(t)
}

func (mw *MainWindow) setToolsBusy(on bool, msg string) {
	fyne.Do(func() {
		if msg != "" {
			mw.ToolsStatus.SetText(msg)
		}
		if on {
			mw.ToolsBusy.Show()
			mw.BtnToolsUpdate.Disable()
			mw.BtnToolsCancel.Enable()
		} else {
			mw.ToolsBusy.Hide()
			mw.BtnToolsUpdate.Enable()
			mw.BtnToolsCancel.Disable()
			mw.ToolsBusy.SetValue(0)
		}
	})
}

func (mw *MainWindow) onToolsUpdate() {
	mw.UpdMu.Lock()
	if mw.UpdRunning || mw.Downloading {
		mw.UpdMu.Unlock()
		return
	}
	mw.UpdRunning = true
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	mw.UpdCancel = cancel
	mw.UpdMu.Unlock()

	mw.setToolsBusy(true, "Tools: updating…")
	mw.Logger.Dbg("--- TOOLS UPDATE ---")

	go func() {
		tools, err := bundled.EnsureToolsAutoUpdate(ctx, "YtDownloader", true, func(task string, pct float64) {
			fyne.Do(func() {
				mw.ToolsStatus.SetText(task)
				mw.ToolsBusy.SetValue(pct * 100)
			})
		})

		mw.UpdMu.Lock()
		mw.UpdRunning = false
		c := mw.UpdCancel
		mw.UpdCancel = nil
		mw.UpdMu.Unlock()
		if c != nil {
			c()
		}

		if err != nil {
			mw.setToolsBusy(false, "Tools: update failed")
			fyne.Do(func() { dialog.ShowError(err, mw.Window) })
			mw.Logger.Warn("Tools update failed: " + err.Error())
			return
		}

		mw.UpdMu.Lock()
		if mw.Downloading {
			mw.UpdMu.Unlock()
			mw.setToolsBusy(false, "Tools: updated (will apply after download)")
			mw.Logger.Info("Tools updated")
			return
		}
		mw.UpdMu.Unlock()

		mw.Cli.YtDlpPath = tools.YtDlpPath
		mw.Cli.FFmpegDir = tools.BinDir

		mw.setToolsBusy(false, "Tools: updated")
		mw.Logger.Info("Tools updated")
	}()
}

func (mw *MainWindow) onToolsCancel() {
	mw.UpdMu.Lock()
	c := mw.UpdCancel
	mw.UpdCancel = nil
	mw.UpdMu.Unlock()
	if c != nil {
		c()
		mw.setToolsBusy(false, "Tools: update cancelled")
		mw.Logger.Warn("Tools update cancelled")
	}
}
