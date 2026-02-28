package app

import (
	"bytes"
	"io"
	"net/http"
	neturl "net/url"
	"path"
	"strings"
	"time"

	"YtDownloader/internal/ytdlp"

	"fyne.io/fyne/v2"
)

func pickThumbCandidates(info *ytdlp.VideoInfo) []string {
	type cand struct {
		url string
		px  int
	}

	isLikelyImage := func(u string) bool {
		u = strings.ToLower(u)
		return strings.Contains(u, ".jpg") || strings.Contains(u, ".jpeg") ||
			strings.Contains(u, ".png") || strings.Contains(u, ".webp")
	}

	addBoth := func(out *[]cand, u string, px int) {
		u = strings.TrimSpace(u)
		if u == "" {
			return
		}
		*out = append(*out, cand{url: u, px: px})
		lu := strings.ToLower(u)
		if strings.Contains(lu, ".webp") {
			*out = append(*out, cand{url: strings.ReplaceAll(u, ".webp", ".jpg"), px: px - 1})
			*out = append(*out, cand{url: strings.ReplaceAll(u, ".webp", ".jpeg"), px: px - 2})
		}
	}

	list := make([]cand, 0, len(info.Thumbnails)+4)
	for _, t := range info.Thumbnails {
		u := strings.TrimSpace(t.URL)
		if u == "" {
			continue
		}
		ext := strings.ToLower(strings.TrimSpace(t.Ext))
		if ext == "jpg" || ext == "jpeg" || ext == "png" || ext == "webp" || isLikelyImage(u) {
			px := 0
			if t.Width > 0 && t.Height > 0 {
				px = t.Width * t.Height
			}
			addBoth(&list, u, px)
		}
	}

	if u := strings.TrimSpace(info.Thumbnail); u != "" && isLikelyImage(u) {
		addBoth(&list, u, 0)
	}

	for i := 0; i < len(list); i++ {
		best := i
		for j := i + 1; j < len(list); j++ {
			if list[j].px > list[best].px {
				best = j
			}
		}
		list[i], list[best] = list[best], list[i]
	}

	seen := make(map[string]struct{}, len(list))
	out := make([]string, 0, len(list))
	for _, c := range list {
		if c.url == "" {
			continue
		}
		if _, ok := seen[c.url]; ok {
			continue
		}
		seen[c.url] = struct{}{}
		out = append(out, c.url)
	}
	return out
}

func loadRemoteImageResource(url string) fyne.Resource {
	client := &http.Client{Timeout: 12 * time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil || len(b) == 0 {
		return nil
	}

	name := "thumb"
	ext := ".jpg"
	if u, err := neturl.Parse(url); err == nil {
		if e := path.Ext(u.Path); e != "" {
			ext = e
		}
	}
	name += ext

	return fyne.NewStaticResource(name, bytes.Clone(b))
}
