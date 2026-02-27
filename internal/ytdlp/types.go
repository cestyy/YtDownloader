package ytdlp

type Thumbnail struct {
	URL    string `json:"url"`
	Ext    string `json:"ext"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type VideoInfo struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	WebpageURL string      `json:"webpage_url"`
	Thumbnail  string      `json:"thumbnail"`
	Thumbnails []Thumbnail `json:"thumbnails"`
	Duration   float64     `json:"duration"`
	Formats    []Format    `json:"formats"`
}

type Format struct {
	FormatID       string  `json:"format_id"`
	Ext            string  `json:"ext"`
	Protocol       string  `json:"protocol"`
	VCodec         string  `json:"vcodec"`
	ACodec         string  `json:"acodec"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	FPS            float64 `json:"fps"`
	Filesize       int64   `json:"filesize"`
	FilesizeApprox int64   `json:"filesize_approx"`
	TBR            float64 `json:"tbr"`
	ABR            float64 `json:"abr"`
	VBR            float64 `json:"vbr"`
}

type Progress struct {
	Pct string `json:"p"`
	Eta string `json:"eta"`
	Spd string `json:"spd"`
	Dl  string `json:"dl"`
	Tot string `json:"tot"`
}
