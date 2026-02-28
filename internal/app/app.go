package app

import (
	ui "YtDownloader/internal/ui"
	"context"
	_ "embed"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"YtDownloader/internal/bundled"
	"YtDownloader/internal/ytdlp"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

//go:embed logo.png
var appIconBytes []byte

func initFileLogging(appName string) (closer func(), logPath string, err error) {
	cfg, err := os.UserConfigDir()
	if err != nil || cfg == "" {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return nil, "", herr
		}
		cfg = filepath.Join(home, "."+appName)
	}
	logDir := filepath.Join(cfg, appName, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, "", err
	}
	logPath = filepath.Join(logDir, "app.log")

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, "", err
	}
	log.SetOutput(f)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Printf("---- START %s ----", time.Now().Format(time.RFC3339))

	return func() {
		_ = f.Sync()
		_ = f.Close()
	}, logPath, nil
}

func Run() {
	closeLog, logPath, err := initFileLogging("YtDownloader")
	if err == nil && closeLog != nil {
		defer closeLog()
		log.Printf("INFO: log file: %s", logPath)
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC: %v\n%s", r, string(debug.Stack()))
		}
	}()

	a := fyneapp.NewWithID("com.cessttyy.ytDownloader")
	myIcon := fyne.NewStaticResource("logo.png", appIconBytes)
	a.SetIcon(myIcon)

	applyEmbeddedFont(a)

	tools, err := bundled.EnsureToolsFast("YtDownloader", true)
	if err == bundled.ErrToolsMissing {
		boot := a.NewWindow("YtDownloader")
		boot.Resize(fyne.NewSize(440, 160))
		boot.SetFixedSize(true)

		lbl := widget.NewLabel("Downloading tools (yt-dlp + ffmpeg) to C:/Users/<USER>/AppData/Local/<YtDownloader>/bin/")
		progress := widget.NewProgressBar()
		progress.Min, progress.Max = 0, 100
		progress.SetValue(0)

		msg := widget.NewLabel("This is needed only on first run.")
		btnCancel := widget.NewButton("Cancel", func() {})

		boot.SetContent(container.NewVBox(lbl, progress, msg, btnCancel))
		boot.Show()

		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		btnCancel.OnTapped = func() {
			cancel()
			boot.Close()
			a.Quit()
		}

		go func() {
			defer cancel()
			t, derr := bundled.EnsureToolsAutoUpdate(ctx, "YtDownloader", true, func(task string, pct float64) {
				progress.SetValue(pct * 100)
			})
			if derr != nil {
				log.Printf("ERROR: EnsureToolsAutoUpdate: %v", derr)
				fyne.Do(func() {
					dialog.ShowError(derr, boot)
					lbl.SetText("Failed to download tools")
					msg.SetText("Check internet access and try again.")
				})
				return
			}

			cli := &ytdlp.Runner{
				YtDlpPath: t.YtDlpPath,
				FFmpegDir: t.BinDir,
			}

			fyne.Do(func() {
				main := ui.ShowMainWindow(a, cli)
				boot.Close()
				main.Show()
			})
		}()

		a.Run()
		return
	}

	if err != nil {
		log.Printf("ERROR: EnsureToolsFast: %v", err)
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
		if _, err := bundled.EnsureToolsAutoUpdate(ctx, "YtDownloader", false, nil); err != nil {
			log.Printf("WARN: tools auto-update: %v", err)
		} else {
			log.Printf("INFO: tools updated (background)")
		}
	}()

	w.ShowAndRun()
}
