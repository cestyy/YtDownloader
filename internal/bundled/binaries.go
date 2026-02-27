package bundled

import _ "embed"

//go:embed assets/bin/windows/amd64/yt-dlp.exe
var YTDLP []byte

//go:embed assets/bin/windows/amd64/ffmpeg.exe
var FFMPEG []byte

//go:embed assets/bin/windows/amd64/ffprobe.exe
var FFPROBE []byte
