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
		widget.NewLabelWithStyle(T("filters"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel(T("resolution")), mw.ResSelect),
			container.NewVBox(widget.NewLabel(T("format")), mw.ExtSelect),
		),
	)

	topRow := container.NewBorder(nil, nil, widget.NewLabel(T("url")), nil, mw.UrlEntry)
	dirRow := container.NewBorder(
		nil, nil,
		widget.NewLabel(T("save_to")),
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
		widget.NewLabel(T("preview")), nil, nil, nil, previewCenter,
	)
	playlistTopBtn := container.NewHBox(mw.BtnSelectAll, mw.BtnUnselectAll, layout.NewSpacer(), mw.SelectedCount)
	mw.PlaylistPanel = container.NewBorder(playlistTopBtn, nil, nil, nil, mw.PlaylistList)

	mw.RightPanelCards = container.NewMax(mw.PreviewContainer)

	rightTop := container.NewVBox(
		widget.NewLabel(T("status")), mw.Status,
		widget.NewSeparator(),
	)
	right := container.NewBorder(rightTop, nil, nil, nil, mw.RightPanelCards)

	mainSplit := container.NewHSplit(left, right)
	mainSplit.Offset = 0.50

	queueScroll := container.NewVScroll(mw.QueueBox)
	queueTop := container.NewHBox(
		widget.NewLabelWithStyle(T("active_queued"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		mw.BtnClearQueue,
	)
	queueLayout := container.NewBorder(queueTop, nil, nil, nil, queueScroll)

	settingsLayout := container.NewBorder(nil, nil, nil, nil,
		container.NewVScroll(mw.buildSettingsTab()),
	)

	logsTop := container.NewVBox(
		widget.NewLabelWithStyle(T("sys_logs"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		mw.Logger.Controls(mw.Window),
		widget.NewSeparator(),
	)
	logsLayout := container.NewBorder(logsTop, nil, nil, nil, container.NewMax(mw.Logger.Widget()))

	mw.DownloadsTab = container.NewTabItemWithIcon(T("downloads_tab"), theme.DownloadIcon(), queueLayout)
	mw.HistoryTab = container.NewTabItemWithIcon(T("history_tab"), theme.HistoryIcon(), mw.buildHistoryTab())

	mw.Tabs = container.NewAppTabs(
		container.NewTabItemWithIcon("Main", theme.HomeIcon(), mainSplit),
		mw.DownloadsTab,
		mw.HistoryTab,
		container.NewTabItemWithIcon(T("settings_tab"), theme.SettingsIcon(), settingsLayout),
		container.NewTabItemWithIcon(T("logs_tab"), theme.DocumentIcon(), logsLayout),
	)
	mw.Tabs.SetTabLocation(container.TabLocationLeading)

	return container.NewMax(mw.Tabs)
}

func (mw *MainWindow) buildSettingsTab() *fyne.Container {
	langSelect := widget.NewSelect([]string{LangEn, LangRu}, func(s string) {
		if CurrentLang != s {
			mw.App.Preferences().SetString("Language", s)
			CurrentLang = s
			mw.UpdateLanguage()
		}
	})
	langSelect.SetSelected(CurrentLang)

	return container.NewVBox(
		settingsSectionHeader(T("appearance")),
		settingsRow(T("lang"), langSelect),
		settingsRow(T("theme"), mw.ThemeSelect),

		widget.NewSeparator(),

		settingsSectionHeader(T("download_sec")),
		mw.CheckSponsorBlock,
		mw.CheckRedownload,
		mw.CheckEmbedMeta,
		settingsRow(T("output_format"), mw.FormatSelect),
		settingsRow(T("naming_template"), mw.NamingSelect),
		settingsRow(T("parallel_dl"), mw.ConcurrentSelect),
		settingsLabel(T("custom_args")),
		mw.CustomArgsEntry,

		widget.NewSeparator(),

		settingsSectionHeader(T("auth_sec")),
		settingsLabel(T("browser_cookie")),
		mw.BrowserSelect,
		settingsLabel(T("file_cookie")),
		container.NewHBox(mw.BtnCookiesSelect, mw.BtnCookiesClear),
		mw.CookiesFileLabel,

		widget.NewSeparator(),

		settingsSectionHeader(T("tools_sec")),
		settingsRow(T("status"), mw.ToolsStatus),
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
