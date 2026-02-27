package bundled

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrToolsMissing = errors.New("tools missing: need download")

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

	ok := fileOk(t.YtDlpPath)

	if withFFmpeg {
		t.FfmpegPath = filepath.Join(dir, "ffmpeg.exe")
		t.FfprobePath = filepath.Join(dir, "ffprobe.exe")
		ok = ok && fileOk(t.FfmpegPath) && fileOk(t.FfprobePath)
	}

	if ok {
		return t, nil
	}

	if len(YTDLP) > 0 && !fileOk(t.YtDlpPath) {
		if err := ensureFile(t.YtDlpPath, YTDLP); err != nil {
			return nil, err
		}
		if withFFmpeg {
			if len(FFMPEG) > 0 && !fileOk(t.FfmpegPath) {
				_ = ensureFile(t.FfmpegPath, FFMPEG)
			}
			if len(FFPROBE) > 0 && !fileOk(t.FfprobePath) {
				_ = ensureFile(t.FfprobePath, FFPROBE)
			}
		}
		if fileOk(t.YtDlpPath) && (!withFFmpeg || (fileOk(t.FfmpegPath) && fileOk(t.FfprobePath))) {
			return t, nil
		}
	}

	return t, ErrToolsMissing
}
