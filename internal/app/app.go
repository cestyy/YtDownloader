package app

import (
	"context"
	"time"

	"YtDownloader/internal/bundled"
	"YtDownloader/internal/ui"
	"YtDownloader/internal/ytdlp"

	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
)

func Run() {
	a := fyneapp.NewWithID("com.cessttyy.ytDownloader")
	applyEmbeddedFont(a)

	tools, err := bundled.EnsureToolsFast("YtDownloader", true)
	if err != nil {
		w := a.NewWindow("YtDownloader")
		dialog.ShowError(err, w)
		w.ShowAndRun()
		return
	}

	cli := &ytdlp.Runner{
		YtDlpPath: tools.YtDlpPath,
		FFmpegDir: tools.BinDir,
	}

	w := ui.ShowMainWindow(a, cli)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		if _, err := bundled.EnsureToolsAutoUpdate(ctx, "YtDownloader", true); err != nil {
			return
		}
	}()

	w.ShowAndRun()
}
