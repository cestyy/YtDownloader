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
	dirRow := container.NewBorder(
		nil, nil,
		widget.NewLabel("Save to:"),
		container.NewHBox(mw.BtnChooseDir, mw.BtnOpenFolder),
		mw.OutDirLabel,
	)

	btnRow := container.NewHBox(mw.BtnDownload, mw.BtnBest, mw.BtnBestAudio, layout.NewSpacer())

	leftTop := container.NewVBox(
		topRow, dirRow, btnRow,
		widget.NewSeparator(), filterUI, mw.Busy,
	)

	left := container.NewBorder(leftTop, nil, nil, nil, container.NewMax(mw.FormatList))

	previewCenter := container.NewBorder(mw.PreviewTitle, nil, nil, nil, mw.PreviewImg)
	mw.PreviewContainer = container.NewBorder(
		widget.NewLabel("Preview:"), nil, nil, nil, previewCenter,
	)
	playlistTopBtn := container.NewHBox(mw.BtnSelectAll, mw.BtnUnselectAll, layout.NewSpacer(), mw.SelectedCount)
	mw.PlaylistPanel = container.NewBorder(playlistTopBtn, nil, nil, nil, mw.PlaylistList)

	mw.RightPanelCards = container.NewMax(mw.PreviewContainer)

	rightTop := container.NewVBox(
		widget.NewLabel("Status:"), mw.Status,
		widget.NewSeparator(),
	)
	right := container.NewBorder(rightTop, nil, nil, nil, mw.RightPanelCards)

	mainSplit := container.NewHSplit(left, right)
	mainSplit.Offset = 0.50

	queueScroll := container.NewVScroll(mw.QueueBox)
	queueTop := container.NewHBox(
		widget.NewLabelWithStyle("Active & Queued Downloads", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		mw.BtnClearQueue,
	)
	queueLayout := container.NewBorder(queueTop, nil, nil, nil, queueScroll)

	settingsLayout := container.NewBorder(nil, nil, nil, nil,
		container.NewVScroll(mw.buildSettingsTab()),
	)

	logsTop := container.NewVBox(
		widget.NewLabelWithStyle("System Logs & Output", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		mw.Logger.Controls(mw.Window),
		widget.NewSeparator(),
	)
	logsLayout := container.NewBorder(logsTop, nil, nil, nil, container.NewMax(mw.Logger.Widget()))

	mw.DownloadsTab = container.NewTabItemWithIcon("Downloads", theme.DownloadIcon(), queueLayout)
	mw.HistoryTab = container.NewTabItemWithIcon("History", theme.HistoryIcon(), mw.buildHistoryTab())

	mw.Tabs = container.NewAppTabs(
		container.NewTabItemWithIcon("Main", theme.HomeIcon(), mainSplit),
		mw.DownloadsTab,
		mw.HistoryTab,
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), settingsLayout),
		container.NewTabItemWithIcon("Logs", theme.DocumentIcon(), logsLayout),
	)
	mw.Tabs.SetTabLocation(container.TabLocationLeading)

	return container.NewMax(mw.Tabs)
}

func (mw *MainWindow) buildSettingsTab() *fyne.Container {
	return container.NewVBox(

		settingsSectionHeader("🎨  Appearance"),
		settingsRow("Theme", mw.ThemeSelect),

		widget.NewSeparator(),

		settingsSectionHeader("⬇️  Download"),
		mw.CheckSponsorBlock,
		mw.CheckRedownload,
		mw.CheckEmbedMeta,
		settingsRow("Output format (video/audio)", mw.FormatSelect),
		settingsRow("File naming template", mw.NamingSelect),
		settingsRow("Parallel downloads", mw.ConcurrentSelect),
		settingsLabel("Custom yt-dlp arguments"),
		mw.CustomArgsEntry,

		widget.NewSeparator(),

		settingsSectionHeader("🔑  Authentication"),
		settingsLabel("Browser (for cookie bypass):"),
		mw.BrowserSelect,
		settingsLabel("Or use a cookies.txt file:"),
		container.NewHBox(mw.BtnCookiesSelect, mw.BtnCookiesClear),
		mw.CookiesFileLabel,

		widget.NewSeparator(),

		settingsSectionHeader("🔧  Tools (yt-dlp & ffmpeg)"),
		settingsRow("Status", mw.ToolsStatus),
		mw.ToolsBusy,
		container.NewHBox(mw.BtnToolsUpdate, mw.BtnToolsCancel, mw.BtnToolsFolder),
	)
}

func settingsSectionHeader(title string) *widget.Label {
	return widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

func settingsLabel(text string) *widget.Label {
	return widget.NewLabel(text)
}

func settingsRow(label string, control fyne.CanvasObject) *fyne.Container {
	lbl := widget.NewLabel(label)
	return container.NewBorder(nil, nil, lbl, nil, control)
}
