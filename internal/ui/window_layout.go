package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (mw *MainWindow) buildLayout() *fyne.Container {
	filterUI := container.NewVBox(
		widget.NewLabelWithStyle("Filters:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel("Resolution (Quality):"), mw.ResSelect),
			container.NewVBox(widget.NewLabel("Format (Extension):"), mw.ExtSelect),
		),
	)

	topRow := container.NewBorder(nil, nil, widget.NewLabel("URL:"), nil, mw.UrlEntry)
	dirRow := container.NewHBox(
		widget.NewLabel("Save to:"), mw.OutDirLabel,
		layout.NewSpacer(), mw.BtnChooseDir, mw.BtnOpenFolder,
	)
	btnRow := container.NewHBox(mw.BtnDownload, mw.BtnBest, mw.BtnCancel, layout.NewSpacer())

	leftTop := container.NewVBox(
		topRow, dirRow, btnRow,
		widget.NewSeparator(), filterUI, mw.Busy,
	)
	left := container.NewBorder(leftTop, nil, nil, nil, container.NewMax(mw.FormatList))

	mw.PreviewContainer = container.NewVBox(
		widget.NewLabel("Preview:"), mw.PreviewTitle, mw.PreviewImg,
	)
	playlistTopBtn := container.NewHBox(mw.BtnSelectAll, mw.BtnUnselectAll, layout.NewSpacer(), mw.SelectedCount)
	mw.PlaylistPanel = container.NewBorder(playlistTopBtn, nil, nil, nil, mw.PlaylistList)

	mw.RightPanelCards = container.NewMax(mw.PreviewContainer)

	rightTop := container.NewVBox(
		widget.NewLabel("Status:"), mw.Status, mw.DownloadProgress, mw.ProgressInfo,
		widget.NewSeparator(),
	)
	right := container.NewBorder(rightTop, nil, nil, nil, mw.RightPanelCards)

	mainSplit := container.NewHSplit(left, right)
	mainSplit.Offset = 0.50

	settingsView := container.NewVBox(
		widget.NewLabelWithStyle("Application Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Appearance"), mw.ThemeSelect,
		widget.NewSeparator(),
		widget.NewLabel("Downloads"),
		mw.CheckSponsorBlock,
		mw.CheckRedownload,
		widget.NewLabel("File Naming Template"),
		mw.NamingSelect,
		widget.NewSeparator(),
		widget.NewLabel("Output Format (video/audio)"), mw.FormatSelect,
		widget.NewSeparator(),
		widget.NewLabel("Bypass YouTube Bot Check"), mw.BrowserSelect,
		container.NewHBox(mw.BtnCookiesSelect, mw.BtnCookiesClear),
		mw.CookiesFileLabel,
		widget.NewSeparator(),
		widget.NewLabel("Tools (yt-dlp & ffmpeg)"),
		container.NewHBox(mw.ToolsStatus, mw.ToolsBusy),
		container.NewHBox(mw.BtnToolsUpdate, mw.BtnToolsCancel, mw.BtnToolsFolder),
	)

	settingsLayout := container.NewBorder(settingsView, nil, nil, nil, nil)

	logsTop := container.NewVBox(
		widget.NewLabelWithStyle("System Logs & Output", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		mw.Logger.Controls(mw.Window),
		widget.NewSeparator(),
	)

	logsLayout := container.NewBorder(logsTop, nil, nil, nil, container.NewMax(mw.Logger.Widget()))

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Main", theme.HomeIcon(), mainSplit),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), settingsLayout),
		container.NewTabItemWithIcon("Logs", theme.DocumentIcon(), logsLayout),
	)
	tabs.SetTabLocation(container.TabLocationLeading)

	return container.NewMax(tabs)
}
