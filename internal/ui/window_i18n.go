package app

func (mw *MainWindow) UpdateLanguage() {
	mw.UrlEntry.SetPlaceHolder(T("paste_url"))
	if mw.Status.Text == "Ready" || mw.Status.Text == "Готов" {
		mw.Status.SetText(T("ready"))
	}

	if mw.CookiesFilePath == "" {
		mw.CookiesFileLabel.SetText(T("no_cookies"))
	}
	mw.BtnCookiesSelect.SetText(T("select_cookies"))
	mw.BtnCookiesClear.SetText(T("clear"))

	mw.CustomArgsEntry.SetPlaceHolder(T("custom_args_eg"))
	mw.BtnDownload.SetText(T("download_selected"))
	mw.BtnBest.SetText(T("best_video"))
	mw.BtnBestAudio.SetText(T("best_audio"))
	mw.BtnOpenFolder.SetText(T("open_folder"))
	mw.BtnChooseDir.SetText(T("select_dir"))

	if mw.ToolsStatus.Text == "Tools: ready" || mw.ToolsStatus.Text == "Утилиты: готовы" {
		mw.ToolsStatus.SetText(T("tools_ready"))
	}
	mw.BtnToolsFolder.SetText(T("tools_folder"))
	mw.BtnToolsUpdate.SetText(T("update_tools"))
	mw.BtnToolsCancel.SetText(T("cancel_update"))

	mw.CheckSponsorBlock.Text = T("sponsorblock")
	mw.CheckSponsorBlock.Refresh()
	mw.CheckRedownload.Text = T("redownload")
	mw.CheckRedownload.Refresh()
	if mw.CheckEmbedMeta != nil {
		mw.CheckEmbedMeta.Text = T("embed_meta")
		mw.CheckEmbedMeta.Refresh()
	}

	mw.BtnSelectAll.SetText(T("select_all"))
	mw.BtnUnselectAll.SetText(T("unselect_all"))
	mw.BtnClearQueue.SetText(T("clear_finished"))

	var currentTabIndex int
	if mw.Tabs != nil && mw.Tabs.Selected() != nil {
		for i, t := range mw.Tabs.Items {
			if t == mw.Tabs.Selected() {
				currentTabIndex = i
				break
			}
		}
	}

	mw.Window.SetContent(mw.buildLayout())

	if mw.Tabs != nil && currentTabIndex >= 0 && currentTabIndex < len(mw.Tabs.Items) {
		mw.Tabs.SelectIndex(currentTabIndex)
	}
}
