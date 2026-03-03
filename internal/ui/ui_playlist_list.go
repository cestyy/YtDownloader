package app

import (
	"context"
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	thumbCache  sync.Map
	thumbSem    = make(chan struct{}, 5)
	thumbCtx    context.Context
	thumbCtxMu  sync.Mutex
	thumbCancel context.CancelFunc
)

func (mw *MainWindow) setupPlaylistList() {
	thumbCtxMu.Lock()
	if thumbCancel != nil {
		thumbCancel()
	}
	thumbCtx, thumbCancel = context.WithCancel(context.Background())
	thumbCtxMu.Unlock()

	mw.PlaylistList = widget.NewList(
		func() int { return len(mw.PlaylistEntries) },
		func() fyne.CanvasObject {
			chk := widget.NewCheck("", nil)

			img := canvas.NewImageFromResource(theme.FileImageIcon())
			img.SetMinSize(fyne.NewSize(80, 45))
			img.FillMode = canvas.ImageFillContain

			title := widget.NewLabel("Title")
			title.TextStyle.Bold = true
			title.Truncation = fyne.TextTruncateEllipsis

			author := widget.NewLabel("Author")
			author.TextStyle = fyne.TextStyle{Italic: true}

			statusLbl := widget.NewLabel("")
			statusLbl.TextStyle = fyne.TextStyle{Bold: true}

			bottomRow := container.NewHBox(author, layout.NewSpacer(), statusLbl)
			vbox := container.NewVBox(title, bottomRow)

			return container.NewBorder(nil, nil, container.NewHBox(chk, img), nil, vbox)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			entry := mw.PlaylistEntries[i]
			c := o.(*fyne.Container)

			leftHbox := c.Objects[1].(*fyne.Container)
			chk := leftHbox.Objects[0].(*widget.Check)
			img := leftHbox.Objects[1].(*canvas.Image)

			vbox := c.Objects[0].(*fyne.Container)
			title := vbox.Objects[0].(*widget.Label)

			bottomRow := vbox.Objects[1].(*fyne.Container)
			author := bottomRow.Objects[0].(*widget.Label)
			statusLbl := bottomRow.Objects[2].(*widget.Label)

			title.SetText(fmt.Sprintf("%d. %s", i+1, entry.Title))
			author.SetText(entry.Uploader)

			statusLbl.SetText(mw.PlaylistStatuses[i])

			chk.Checked = mw.PlaylistChecks[i]
			chk.OnChanged = func(b bool) {
				mw.PlaylistChecks[i] = b
				mw.updateSelectedCount()
			}
			chk.Refresh()

			img.Resource = theme.FileImageIcon()

			cached, ok := thumbCache.Load(entry.ID)
			if ok && cached != "loading" {
				if res, isRes := cached.(fyne.Resource); isRes {
					img.Resource = res
				}
			} else if !ok {
				thumbCache.Store(entry.ID, "loading")
				thumbURL := fmt.Sprintf("https://img.youtube.com/vi/%s/mqdefault.jpg", entry.ID)

				go func(id, url string, index widget.ListItemID) {
					thumbCtxMu.Lock()
					ctx := thumbCtx
					thumbCtxMu.Unlock()

					select {
					case <-ctx.Done():
						thumbCache.Delete(id)
						return
					case thumbSem <- struct{}{}:
					}
					defer func() { <-thumbSem }()

					res := loadRemoteImageResource(url)
					if res != nil {
						thumbCache.Store(id, res)
						fyne.Do(func() {
							if mw.PlaylistList != nil {
								mw.PlaylistList.RefreshItem(index)
							}
						})
					} else {
						thumbCache.Delete(id)
					}
				}(entry.ID, thumbURL, i)
			}

			img.Refresh()
		},
	)
}
