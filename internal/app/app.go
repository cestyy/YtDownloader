package app

import (
	fyneapp "fyne.io/fyne/v2/app"

	"YtDownloader/internal/bundled"
	"YtDownloader/internal/ui"
	"YtDownloader/internal/ytdlp"
)

func Run() {
	a := fyneapp.NewWithID("com.cessttyy.ytDownloader")
	applyEmbeddedFont(a)

	tools, err := bundled.EnsureTools("YtDownloader", true)
	if err != nil {
		panic(err)
	}

	cli := &ytdlp.Runner{
		YtDlpPath: tools.YtDlpPath,
		FFmpegDir: tools.BinDir,
	}

	w := ui.ShowMainWindow(a, cli)
	w.ShowAndRun()
}
