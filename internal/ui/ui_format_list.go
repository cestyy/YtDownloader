package app

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (mw *MainWindow) setupFormatList() {
	mw.FormatList = widget.NewList(
		func() int { return len(mw.FormatsView) },
		func() fyne.CanvasObject {
			l1 := widget.NewLabel("")
			l1.TextStyle = fyne.TextStyle{Bold: true}
			l1.Truncation = fyne.TextTruncateEllipsis
			l2 := widget.NewLabel("")
			l2.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(l1, l2)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			f := mw.FormatsView[i]
			v := o.(*fyne.Container)
			l1 := v.Objects[0].(*widget.Label)
			l2 := v.Objects[1].(*widget.Label)

			res := "Audio"
			if f.Height > 0 {
				res = fmt.Sprintf("%dp", f.Height)
			} else if f.Width > 0 {
				res = fmt.Sprintf("%dp", f.Width)
			}

			sizeBytes := f.Filesize
			if sizeBytes == 0 {
				sizeBytes = f.FilesizeApprox
			}
			l1.SetText(fmt.Sprintf("%s  •  %s  •  Time: %s", res, formatBytes(sizeBytes), formatDuration(mw.CurrentVideoDuration)))

			fps := ""
			if f.FPS > 0 {
				fps = fmt.Sprintf(" | fps: %v", f.FPS)
			}
			l2.SetText(fmt.Sprintf("fmt: %s | ext: %s%s | v: %s a: %s", emptyToDash(f.FormatID), f.Ext, fps, emptyToDash(f.VCodec), emptyToDash(f.ACodec)))
		},
	)
}
