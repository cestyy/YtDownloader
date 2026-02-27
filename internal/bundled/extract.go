package bundled

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
)

func appBinDir(appName string) (string, error) {
	cache, err := os.UserCacheDir()
	if err == nil && cache != "" {
		return filepath.Join(cache, appName, "bin"), nil
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, appName, "bin"), nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func fileSha256Hex(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return sha256Hex(data), nil
}

func ensureFile(path string, content []byte) error {
	want := sha256Hex(content)

	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		got, err := fileSha256Hex(path)
		if err == nil && got == want {
			return nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

type Tools struct {
	YtDlpPath   string
	FfmpegPath  string
	FfprobePath string
	BinDir      string
}

func EnsureTools(appName string, withFFmpeg bool) (*Tools, error) {
	dir, err := appBinDir(appName)
	if err != nil {
		return nil, err
	}

	t := &Tools{
		BinDir:    dir,
		YtDlpPath: filepath.Join(dir, "yt-dlp.exe"),
	}
	if err := ensureFile(t.YtDlpPath, YTDLP); err != nil {
		return nil, err
	}

	if withFFmpeg {
		t.FfmpegPath = filepath.Join(dir, "ffmpeg.exe")
		t.FfprobePath = filepath.Join(dir, "ffprobe.exe")

		if len(FFMPEG) == 0 || len(FFPROBE) == 0 {
			return nil, errors.New("ffmpeg requested but not embedded")
		}
		if err := ensureFile(t.FfmpegPath, FFMPEG); err != nil {
			return nil, err
		}
		if err := ensureFile(t.FfprobePath, FFPROBE); err != nil {
			return nil, err
		}
	}

	return t, nil
}
