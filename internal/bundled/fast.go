package bundled

import (
	"errors"
	"os"
	"path/filepath"
)

func EnsureToolsFast(appName string, withFFmpeg bool) (*Tools, error) {
	dir, err := appBinDir(appName)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	t := &Tools{
		BinDir:    dir,
		YtDlpPath: filepath.Join(dir, "yt-dlp.exe"),
	}

	if !fileOk(t.YtDlpPath) {
		if len(YTDLP) == 0 {
			return nil, errors.New("yt-dlp not available")
		}
		if err := ensureFile(t.YtDlpPath, YTDLP); err != nil {
			return nil, err
		}
	}

	if withFFmpeg {
		t.FfmpegPath = filepath.Join(dir, "ffmpeg.exe")
		t.FfprobePath = filepath.Join(dir, "ffprobe.exe")

		if !fileOk(t.FfmpegPath) || !fileOk(t.FfprobePath) {
			if len(FFMPEG) == 0 || len(FFPROBE) == 0 {
				return nil, errors.New("ffmpeg not available")
			}
			if err := ensureFile(t.FfmpegPath, FFMPEG); err != nil {
				return nil, err
			}
			if err := ensureFile(t.FfprobePath, FFPROBE); err != nil {
				return nil, err
			}
		}
	}

	return t, nil
}
