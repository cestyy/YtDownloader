package app

import "fyne.io/fyne/v2"

const (
	LangEn = "🇺🇸 English"
	LangRu = "🇷🇺 Русский"
)

var CurrentLang = LangEn

var translations = map[string]map[string]string{
	LangEn: {
		"url":              "URL:",
		"save_to":          "Save to:",
		"filters":          "Filters:",
		"resolution":       "Resolution (Quality):",
		"format":           "Format (Extension):",
		"preview":          "Preview:",
		"status":           "Status:",
		"downloads_tab":    "Downloads",
		"history_tab":      "History",
		"settings_tab":     "Settings",
		"logs_tab":         "Logs",
		"appearance":       "🎨  Appearance",
		"theme":            "Theme",
		"lang":             "Language / Язык",
		"restart_required": "Please restart the application to apply language changes.",

		"ready":             "Ready",
		"loading":           "Loading…",
		"paste_url":         "Paste YouTube link or another…",
		"no_cookies":        "No cookies.txt selected",
		"select_cookies":    "Select cookies.txt",
		"clear":             "Clear",
		"custom_args_eg":    "e.g. --limit-rate 5M",
		"download_selected": "Download selected",
		"best_video":        "Best Video",
		"best_audio":        "Best Audio",
		"open_folder":       "Open folder",
		"select_dir":        "Select Directory",
		"tools_ready":       "Tools: ready",
		"tools_folder":      "Tools folder",
		"update_tools":      "Update tools",
		"cancel_update":     "Cancel update",
		"sponsorblock":      "Remove Sponsor (SponsorBlock)",
		"redownload":        "Force redownload (if already Ready)",
		"embed_meta":        "Embed Metadata & Thumbnail",
		"select_all":        "Select All",
		"unselect_all":      "Unselect all",
		"clear_finished":    "Clear Finished",
		"active_queued":     "Active & Queued Downloads",
		"sys_logs":          "System Logs & Output",

		"history_folder": "Folder",
		"history_play":   "Play",
		"history_remove": "Remove",
		"history_clear":  "Clear History",
		"history_title":  "Download History",

		"selected_count":     "Selected: %d / %d",
		"playlist_videos":    "Playlist: %d videos",
		"found_formats":      "Found formats: %d",
		"queued_videos":      "Queued: 0 / %d videos",
		"downloading_videos": "Downloading: %d / %d videos",
		"playlist_complete":  "Playlist Complete ✅",
		"playlist_errors":    "Playlist had errors",
		"added_queue":        "Added to Queue",
		"skip_title":         "Skip",
		"skip_msg":           "All selected videos are already downloaded.",
		"speed_label":        "%s / %s   |   Speed: %s   |   ETA: %s",

		"download_sec":    "⬇️ Downloads",
		"output_format":   "Output Format",
		"naming_template": "Naming Template",
		"parallel_dl":     "Parallel Downloads",
		"custom_args":     "Custom ytdlp args",
		"auth_sec":        "🔐 Authentication",
		"browser_cookie":  "Browser Cookies",
		"file_cookie":     "Cookie File",
		"tools_sec":       "⚙️ Tools (yt-dlp & ffmpeg)",
	},
	LangRu: {
		"url":              "Ссылка:",
		"save_to":          "Сохранить в:",
		"filters":          "Фильтры:",
		"resolution":       "Разрешение (Качество):",
		"format":           "Формат (Расширение):",
		"preview":          "Превью:",
		"status":           "Статус:",
		"downloads_tab":    "Загрузки",
		"history_tab":      "История",
		"settings_tab":     "Настройки",
		"logs_tab":         "Логи",
		"appearance":       "🎨  Внешний вид",
		"theme":            "Тема",
		"lang":             "Language / Язык",
		"restart_required": "Пожалуйста, перезапустите приложение для применения языка.",

		"ready":             "Готов",
		"loading":           "Загрузка…",
		"paste_url":         "Вставьте ссылку YouTube или любую другую…",
		"no_cookies":        "Файл cookies.txt не выбран",
		"select_cookies":    "Выбрать cookies.txt",
		"clear":             "Очистить",
		"custom_args_eg":    "напр. --limit-rate 5M",
		"download_selected": "Скачать выбранное",
		"best_video":        "Лучшее видео",
		"best_audio":        "Лучшее аудио",
		"open_folder":       "Открыть папку",
		"select_dir":        "Выбрать папку",
		"tools_ready":       "Утилиты: готовы",
		"tools_folder":      "Папка с утилитами",
		"update_tools":      "Обновить утилиты",
		"cancel_update":     "Отменить",
		"sponsorblock":      "Удалить спонсорские вставки (SponsorBlock)",
		"redownload":        "Скачивать заново (даже если Готов)",
		"embed_meta":        "Вшивать метаданные и превью",
		"select_all":        "Выбрать все",
		"unselect_all":      "Снять выделение",
		"clear_finished":    "Очистить завершенные",
		"active_queued":     "Активные загрузки и очередь",
		"sys_logs":          "Системные логи и вывод",

		"history_folder": "Папка",
		"history_play":   "Файл",
		"history_remove": "Удалить",
		"history_clear":  "Очистить историю",
		"history_title":  "История загрузок",

		"selected_count":     "Выбрано: %d / %d",
		"playlist_videos":    "Плейлист: %d видео",
		"found_formats":      "Найдено форматов: %d",
		"queued_videos":      "В очереди: 0 / %d видео",
		"downloading_videos": "Скачивание: %d / %d видео",
		"playlist_complete":  "Плейлист скачан ✅",
		"playlist_errors":    "Ошибка при скачивании",
		"added_queue":        "Добавлено в очередь",
		"skip_title":         "Пропуск",
		"skip_msg":           "Все выбранные видео уже скачаны.",
		"speed_label":        "%s / %s   |   Скорость: %s   |   Осталось: %s",

		"download_sec":    "⬇️ Загрузки",
		"output_format":   "Формат вывода",
		"naming_template": "Шаблон названия",
		"parallel_dl":     "Параллельные потоки",
		"custom_args":     "Доп. аргументы yt-dlp",
		"auth_sec":        "🔐 Авторизация",
		"browser_cookie":  "Cookies из браузера",
		"file_cookie":     "Файл cookies",
		"tools_sec":       "⚙️ Утилиты (yt-dlp и ffmpeg)",
	},
}

func T(key string) string {
	if val, ok := translations[CurrentLang][key]; ok {
		return val
	}
	if val, ok := translations[LangEn][key]; ok {
		return val
	}
	return key
}

func InitLang(a fyne.App) {
	savedLang := a.Preferences().StringWithFallback("Language", LangEn)
	if savedLang == LangRu {
		CurrentLang = LangRu
	} else {
		CurrentLang = LangEn
	}
}
