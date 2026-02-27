package ytdlp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type Runner struct {
	YtDlpPath string
	FFmpegDir string
}

func hideWindow(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
}

func (r *Runner) baseCmd(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, r.YtDlpPath, args...)
	cmd.Env = append(os.Environ(),
		"PYTHONUTF8=1",
		"PYTHONIOENCODING=utf-8",
	)

	cmd.Cancel = func() error {
		if cmd.Process != nil && runtime.GOOS == "windows" {
			tk := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
			hideWindow(tk)
			_ = tk.Run()
		}
		if cmd.Process != nil {
			return cmd.Process.Kill()
		}
		return nil
	}

	hideWindow(cmd)
	return cmd
}

func (r *Runner) FetchInfo(ctx context.Context, url, browser string) (*VideoInfo, error) {
	args := []string{
		"--no-playlist",
		"--no-warnings",
		"--encoding", "utf-8",
		"--dump-single-json",
	}

	if browser != "" && browser != "none" {
		args = append(args, "--cookies-from-browser", browser)
	}

	args = append(args, url)
	cmd := r.baseCmd(ctx, args...)

	var out bytes.Buffer
	var errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp failed: %w; stderr=%s", err, errb.String())
	}

	var info VideoInfo
	if err := json.Unmarshal(out.Bytes(), &info); err != nil {
		return nil, fmt.Errorf("bad json: %w", err)
	}
	return &info, nil
}

type ProgressHandler func(p Progress)
type LineHandler func(line string)

func (r *Runner) Download(ctx context.Context, url, format, outDir, mergeFormat, browser string, onProgress ProgressHandler, onLine LineHandler) (string, error) {
	if format == "" {
		return "", errors.New("format is empty")
	}

	args := []string{
		"--no-playlist",
		"--encoding", "utf-8",
		"--newline",
		"--progress",
		"--no-color",
		"--no-warnings",
		"-N", "8",
		"--retries", "10",
		"--fragment-retries", "10",
		"--file-access-retries", "5",
		"--http-chunk-size", "10M",
		"--extractor-args", "youtube:player_client=tv,web",
		"-f", format,
		"-P", outDir,
		"-o", "%(title).200B [%(id)s].%(ext)s",
		"--progress-template",
		`download:download:{"p":"%(progress._percent_str)s","eta":"%(progress.eta)s","spd":"%(progress._speed_str)s","dl":"%(progress.downloaded_bytes)s","tot":"%(progress.total_bytes)s"}` + "\n",
	}

	if browser != "" && browser != "none" {
		args = append(args, "--cookies-from-browser", browser)
	}

	if r.FFmpegDir != "" {
		args = append(args, "--ffmpeg-location", r.FFmpegDir)
		if mergeFormat == "mp3" {
			args = append(args, "--extract-audio", "--audio-format", "mp3")
		} else {
			args = append(args, "--merge-output-format", mergeFormat, "--remux-video", mergeFormat)
		}
	}

	args = append(args, url)
	cmd := r.baseCmd(ctx, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	var (
		stateMu       sync.Mutex
		finalFilePath string
		touchedFiles  []string
	)

	handle := func(line string) {
		line = strings.TrimSpace(strings.TrimRight(line, "\r\n"))
		if line == "" {
			return
		}

		stateMu.Lock()
		if strings.HasPrefix(line, "[download] Destination: ") {
			dest := strings.TrimPrefix(line, "[download] Destination: ")
			finalFilePath = dest
			touchedFiles = append(touchedFiles, dest)
		} else if strings.HasPrefix(line, "[Merger] Merging formats into ") {
			dest := strings.TrimPrefix(line, "[Merger] Merging formats into ")
			dest = strings.Trim(dest, `"`)
			finalFilePath = dest
			touchedFiles = append(touchedFiles, dest)
		} else if strings.HasPrefix(line, "[ExtractAudio] Destination: ") {
			dest := strings.TrimPrefix(line, "[ExtractAudio] Destination: ")
			finalFilePath = dest
			touchedFiles = append(touchedFiles, dest)
		} else if strings.Contains(line, "has already been downloaded") && strings.HasPrefix(line, "[download] ") {
			parts := strings.Split(line, " has already been downloaded")
			if len(parts) > 0 {
				finalFilePath = strings.TrimPrefix(parts[0], "[download] ")
			}
		}
		stateMu.Unlock()

		if strings.HasPrefix(line, "download:") {
			raw := strings.TrimPrefix(line, "download:")
			if strings.HasPrefix(raw, "download:") {
				raw = strings.TrimPrefix(raw, "download:")
			}
			var p Progress
			if json.Unmarshal([]byte(raw), &p) == nil && onProgress != nil {
				onProgress(p)
				return
			}
		}

		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			var p Progress
			if json.Unmarshal([]byte(line), &p) == nil && onProgress != nil && strings.TrimSpace(p.Pct) != "" {
				onProgress(p)
				return
			}
		}

		if onLine != nil {
			onLine(line)
		}
	}

	readPipe := func(rdr *bufio.Reader) {
		for {
			line, err := rdr.ReadString('\n')
			if line != "" {
				handle(line)
			}
			if err != nil {
				return
			}
		}
	}

	done := make(chan struct{}, 2)

	go func() {
		readPipe(bufio.NewReaderSize(stdout, 1024*1024))
		done <- struct{}{}
	}()
	go func() {
		readPipe(bufio.NewReaderSize(stderr, 1024*1024))
		done <- struct{}{}
	}()

	<-done
	<-done

	err = cmd.Wait()

	if ctx.Err() != nil {
		for _, f := range touchedFiles {
			_ = os.Remove(f)
			_ = os.Remove(f + ".part")
			_ = os.Remove(f + ".ytdl")
			_ = os.Remove(f + ".temp")
		}
		return "", fmt.Errorf("download cancelled by user")
	}

	if err != nil {
		return finalFilePath, fmt.Errorf("download failed: %w", err)
	}

	return finalFilePath, nil
}
