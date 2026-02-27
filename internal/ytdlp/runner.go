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
	"strings"
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
	hideWindow(cmd)
	return cmd
}

func (r *Runner) FetchInfo(ctx context.Context, url string) (*VideoInfo, error) {
	args := []string{
		"--no-playlist",
		"--no-warnings",
		"--encoding", "utf-8",
		"--dump-single-json",
		url,
	}
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

func (r *Runner) Download(ctx context.Context, url, format, outDir string, onProgress ProgressHandler, onLine LineHandler) (string, error) {
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
		"--extractor-args", "youtube:player_client=ios,android,web",
		"-f", format,
		"-P", outDir,
		"-o", "%(title).200B [%(id)s].%(ext)s",
		"--progress-template",
		`download:download:{"p":"%(progress._percent_str)s","eta":"%(progress.eta)s","spd":"%(progress._speed_str)s","dl":"%(progress.downloaded_bytes)s","tot":"%(progress.total_bytes)s"}` + "\n",
	}

	if r.FFmpegDir != "" {
		args = append(args,
			"--ffmpeg-location", r.FFmpegDir,
			"--merge-output-format", "mp4",
		)
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

	var finalFilePath string

	handle := func(line string) {
		line = strings.TrimSpace(strings.TrimRight(line, "\r\n"))
		if line == "" {
			return
		}

		if strings.HasPrefix(line, "[download] Destination: ") {
			finalFilePath = strings.TrimPrefix(line, "[download] Destination: ")
		} else if strings.HasPrefix(line, "[Merger] Merging formats into ") {
			finalFilePath = strings.TrimPrefix(line, "[Merger] Merging formats into ")
			finalFilePath = strings.Trim(finalFilePath, `"`)
		} else if strings.HasPrefix(line, "[ExtractAudio] Destination: ") {
			finalFilePath = strings.TrimPrefix(line, "[ExtractAudio] Destination: ")
		} else if strings.Contains(line, "has already been downloaded") && strings.HasPrefix(line, "[download] ") {
			parts := strings.Split(line, " has already been downloaded")
			if len(parts) > 0 {
				finalFilePath = strings.TrimPrefix(parts[0], "[download] ")
			}
		}

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

	if err := cmd.Wait(); err != nil {
		return finalFilePath, fmt.Errorf("download failed: %w", err)
	}

	return finalFilePath, nil
}
