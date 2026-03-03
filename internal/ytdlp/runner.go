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

type PlaylistEntry struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Uploader string `json:"uploader"`
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

func (r *Runner) FetchInfo(ctx context.Context, url, browser, cookiesFile string) (*VideoInfo, error) {
	args := []string{
		"--no-warnings",
		"--encoding", "utf-8",
		"--dump-single-json",
	}

	if strings.Contains(url, "list=") {
		args = append(args, "--flat-playlist")
	} else {
		args = append(args, "--no-playlist")
	}

	if cookiesFile != "" {
		args = append(args, "--cookies", cookiesFile)
	} else if browser != "" && browser != "none" {
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

type DownloadOptions struct {
	URL             string
	Format          string
	OutDir          string
	MergeFormat     string
	Browser         string
	CookiesFile     string
	AllowPlaylist   bool
	UseSponsorBlock bool
	NameTemplate    string
	SelectedItems   string
	CustomArgs      string
	EmbedMeta       bool
	OnStart         func(touchedFiles []string)
	OnProgress      ProgressHandler
	OnLine          LineHandler
}

func (r *Runner) Download(ctx context.Context, opts DownloadOptions) (string, error) {
	if opts.Format == "" {
		return "", errors.New("format is empty")
	}

	args := []string{
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
		"-f", opts.Format,
		"-P", opts.OutDir,
		"--progress-template",
		`download:download:{"p":"%(progress._percent_str)s","eta":"%(progress.eta)s","spd":"%(progress._speed_str)s","dl":"%(progress.downloaded_bytes)s","tot":"%(progress.total_bytes)s"}` + "\n",
	}

	if !opts.AllowPlaylist {
		args = append(args, "--no-playlist")
	} else {
		args = append(args, "--yes-playlist")
	}

	if opts.UseSponsorBlock {
		args = append(args, "--sponsorblock-remove", "sponsor,intro,outro")
	}

	if opts.EmbedMeta {
		args = append(args, "--embed-metadata", "--embed-thumbnail")
	}

	fileNameTemplate := "%(title).200B [%(id)s].%(ext)s"
	if opts.NameTemplate == "Author - Title" {
		fileNameTemplate = "%(uploader)s - %(title).200B.%(ext)s"
	} else if opts.NameTemplate == "Title (Year)" {
		fileNameTemplate = "%(title).200B (%(upload_date>%Y)s).%(ext)s"
	}
	args = append(args, "-o", fileNameTemplate)

	if opts.SelectedItems != "" {
		args = append(args, "--playlist-items", opts.SelectedItems)
	}

	if opts.CookiesFile != "" {
		args = append(args, "--cookies", opts.CookiesFile)
	} else if opts.Browser != "" && opts.Browser != "none" {
		args = append(args, "--cookies-from-browser", opts.Browser)
	}

	if r.FFmpegDir != "" {
		args = append(args, "--ffmpeg-location", r.FFmpegDir)
		if opts.MergeFormat == "mp3" {
			args = append(args, "--extract-audio", "--audio-format", "mp3")
		} else {
			args = append(args, "--merge-output-format", opts.MergeFormat, "--remux-video", opts.MergeFormat)
		}
	}

	if opts.CustomArgs != "" {
		args = append(args, strings.Fields(opts.CustomArgs)...)
	}

	args = append(args, opts.URL)
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
			if opts.OnStart != nil {
				opts.OnStart(touchedFiles)
			}
		} else if strings.HasPrefix(line, "[Merger] Merging formats into ") {
			dest := strings.TrimPrefix(line, "[Merger] Merging formats into ")
			dest = strings.Trim(dest, `"`)
			finalFilePath = dest
			touchedFiles = append(touchedFiles, dest)
			if opts.OnStart != nil {
				opts.OnStart(touchedFiles)
			}
		} else if strings.HasPrefix(line, "[ExtractAudio] Destination: ") {
			dest := strings.TrimPrefix(line, "[ExtractAudio] Destination: ")
			finalFilePath = dest
			touchedFiles = append(touchedFiles, dest)
			if opts.OnStart != nil {
				opts.OnStart(touchedFiles)
			}
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
			if json.Unmarshal([]byte(raw), &p) == nil && opts.OnProgress != nil {
				opts.OnProgress(p)
				return
			}
		}

		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			var p Progress
			if json.Unmarshal([]byte(line), &p) == nil && opts.OnProgress != nil && strings.TrimSpace(p.Pct) != "" {
				opts.OnProgress(p)
				return
			}
		}

		if opts.OnLine != nil {
			opts.OnLine(line)
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
		return "", fmt.Errorf("download cancelled by user")
	}

	if err != nil {
		return finalFilePath, fmt.Errorf("download failed: %w", err)
	}

	return finalFilePath, nil
}
