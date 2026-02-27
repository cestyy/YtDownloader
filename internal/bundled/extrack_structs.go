package bundled

type Tools struct {
	YtDlpPath   string
	FfmpegPath  string
	FfprobePath string
	BinDir      string
}

type updateState struct {
	CheckedAtUnix int64  `json:"checked_at_unix"`
	YtDlpTag      string `json:"yt_dlp_tag"`
	YtDlpSHA      string `json:"yt_dlp_sha"`
	FFTag         string `json:"ff_tag"`
	FFZipSHA      string `json:"ff_zip_sha"`
}

type progressWriter struct {
	total      int64
	downloaded int64
	onProgress func(float64)
	lastPct    int
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.downloaded += int64(n)
	if pw.total > 0 && pw.onProgress != nil {
		pct := int(float64(pw.downloaded) / float64(pw.total) * 100)
		if pct > pw.lastPct {
			pw.lastPct = pct
			pw.onProgress(float64(pw.downloaded) / float64(pw.total))
		}
	}
	return n, nil
}
