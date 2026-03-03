package bundled

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func fileOk(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir() && st.Size() > 0
}

func sha256HexBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func fileSha256Hex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func ensureFile(path string, content []byte) error {
	want := sha256HexBytes(content)

	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		got, err := fileSha256Hex(path)
		if err == nil && strings.EqualFold(got, want) {
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

func downloadFileVerified(ctx context.Context, url, dstPath, wantSHA string, onProgress func(float64)) error {
	if wantSHA == "" {
		return errors.New("empty sha256")
	}

	tmp := dstPath + ".tmp"
	_ = os.Remove(tmp)

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "YtDownloader/1.0")

	client := &http.Client{Timeout: 8 * time.Minute}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d for %s", resp.StatusCode, url)
	}

	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer out.Close()

	pw := &progressWriter{
		total:      resp.ContentLength,
		onProgress: onProgress,
	}

	h := sha256.New()
	mw := io.MultiWriter(out, h, pw)

	if _, err := io.Copy(mw, resp.Body); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, wantSHA) {
		_ = os.Remove(tmp)
		return fmt.Errorf("sha256 mismatch: got %s want %s", got, wantSHA)
	}

	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Remove(dstPath)
	if err := os.Rename(tmp, dstPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func downloadText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "YtDownloader/1.0")

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("http %d for %s", resp.StatusCode, url)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func parseChecksumFileSha256(text, wantName string) (string, error) {
	lines := strings.Split(text, "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		fields := strings.Fields(ln)
		if len(fields) < 2 {
			continue
		}
		sum := strings.ToLower(strings.TrimSpace(fields[0]))
		name := strings.TrimSpace(fields[len(fields)-1])
		name = strings.TrimPrefix(name, "*")
		name = strings.TrimPrefix(name, "./")
		if name == wantName {
			if len(sum) == 64 {
				return sum, nil
			}
		}
	}
	return "", fmt.Errorf("sha256 for %s not found", wantName)
}

func statePath(binDir string) string {
	return filepath.Join(binDir, "update_state.json")
}

func loadState(binDir string) updateState {
	p := statePath(binDir)
	b, err := os.ReadFile(p)
	if err != nil {
		return updateState{}
	}
	var s updateState
	if json.Unmarshal(b, &s) != nil {
		return updateState{}
	}
	return s
}

func saveState(binDir string, s updateState) {
	p := statePath(binDir)
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(p+".tmp", b, 0o644)
	_ = os.Rename(p+".tmp", p)
}

func extractFFmpegZip(zipPath, outDir string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()

	var gotFF, gotFP bool

	for _, f := range zr.File {
		n := strings.ToLower(strings.ReplaceAll(f.Name, "\\", "/"))
		if strings.HasSuffix(n, "/ffmpeg.exe") || strings.HasSuffix(n, "ffmpeg.exe") {
			if err := extractOneZipFile(f, filepath.Join(outDir, "ffmpeg.exe")); err != nil {
				return err
			}
			gotFF = true
		}
		if strings.HasSuffix(n, "/ffprobe.exe") || strings.HasSuffix(n, "ffprobe.exe") {
			if err := extractOneZipFile(f, filepath.Join(outDir, "ffprobe.exe")); err != nil {
				return err
			}
			gotFP = true
		}
	}

	if !gotFF || !gotFP {
		return errors.New("ffmpeg.exe/ffprobe.exe not found in zip")
	}
	return nil
}

func extractOneZipFile(zf *zip.File, dst string) error {
	rc, err := zf.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	tmp := dst + ".tmp"
	_ = os.Remove(tmp)

	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	_ = os.Remove(dst)
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func ensureLatestYtDlp(ctx context.Context, dst string, onProgress func(float64)) (string, error) {
	sumsURL := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/SHA2-256SUMS"
	exeURL := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"

	sumsText, err := downloadText(ctx, sumsURL)
	if err != nil {
		return "", fmt.Errorf("yt-dlp: download checksums: %w", err)
	}
	sum, err := parseChecksumFileSha256(sumsText, "yt-dlp.exe")
	if err != nil {
		return "", fmt.Errorf("yt-dlp: parse checksums: %w", err)
	}

	if fileOk(dst) {
		if got, gerr := fileSha256Hex(dst); gerr == nil && strings.EqualFold(got, sum) {
			return sum, nil
		}
	}

	if err := downloadFileVerified(ctx, exeURL, dst, sum, onProgress); err != nil {
		return "", fmt.Errorf("yt-dlp: download exe: %w", err)
	}
	return sum, nil
}

func ensureLatestFFmpegWin64(ctx context.Context, binDir string, onProgress func(float64)) (string, error) {
	zipName := "ffmpeg-master-latest-win64-gpl.zip"
	checksURL := "https://github.com/btbn/ffmpeg-builds/releases/latest/download/checksums.sha256"
	zipURL := "https://github.com/btbn/ffmpeg-builds/releases/latest/download/" + zipName

	checksText, err := downloadText(ctx, checksURL)
	if err != nil {
		return "", fmt.Errorf("ffmpeg: download checksums failed. Please download FFmpeg manually to %s: %w", binDir, err)
	}
	zipSHA, err := parseChecksumFileSha256(checksText, zipName)
	if err != nil {
		return "", fmt.Errorf("ffmpeg: failed to parse checksums for %s. The release format might have changed. Please download FFmpeg manually from https://github.com/BtbN/FFmpeg-Builds/releases and extract ffmpeg.exe and ffprobe.exe to %s. Error: %w", zipName, binDir, err)
	}

	ff := filepath.Join(binDir, "ffmpeg.exe")
	fp := filepath.Join(binDir, "ffprobe.exe")
	if fileOk(ff) && fileOk(fp) {
		return zipSHA, nil
	}

	zipPath := filepath.Join(binDir, "ffmpeg.zip")
	if err := downloadFileVerified(ctx, zipURL, zipPath, zipSHA, onProgress); err != nil {
		return "", fmt.Errorf("ffmpeg: download zip: %w", err)
	}
	if err := extractFFmpegZip(zipPath, binDir); err != nil {
		_ = os.Remove(zipPath)
		return "", fmt.Errorf("ffmpeg: extract zip: %w", err)
	}
	_ = os.Remove(zipPath)

	if !fileOk(ff) || !fileOk(fp) {
		return "", fmt.Errorf("ffmpeg: extracted but ffmpeg.exe/ffprobe.exe missing")
	}

	return zipSHA, nil
}

func EnsureToolsAutoUpdate(ctx context.Context, appName string, withFFmpeg bool, onProgress func(task string, pct float64)) (*Tools, error) {
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

	st := loadState(dir)
	now := time.Now().Unix()
	needCheck := st.CheckedAtUnix == 0 || (now-st.CheckedAtUnix) >= int64((7*24*time.Hour).Seconds())

	ff := filepath.Join(dir, "ffmpeg.exe")
	fp := filepath.Join(dir, "ffprobe.exe")

	if needCheck || !fileOk(t.YtDlpPath) || (withFFmpeg && (!fileOk(ff) || !fileOk(fp))) {
		newState := st
		newState.CheckedAtUnix = now

		if sum, err := ensureLatestYtDlp(ctx, t.YtDlpPath, func(pct float64) {
			if onProgress != nil {
				onProgress("Downloadint yt-dlp...", pct)
			}
		}); err == nil {
			newState.YtDlpTag = "latest"
			newState.YtDlpSHA = sum
		} else {
			if !fileOk(t.YtDlpPath) && len(YTDLP) == 0 {
				return nil, err
			}
		}

		if withFFmpeg {
			if zipSHA, err := ensureLatestFFmpegWin64(ctx, dir, func(pct float64) {
				if onProgress != nil {
					onProgress("Downloading FFmpeg...", pct)
				}
			}); err == nil {
				newState.FFTag = "latest"
				newState.FFZipSHA = zipSHA
			} else {
				if (!fileOk(ff) || !fileOk(fp)) && (len(FFMPEG) == 0 || len(FFPROBE) == 0) {
					return nil, err
				}
			}
		}

		saveState(dir, newState)
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
		t.FfmpegPath = ff
		t.FfprobePath = fp

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
