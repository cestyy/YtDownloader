# Third-Party Notices

This repository contains source code for YtDownloader. The application may download and use third-party software at runtime.

## yt-dlp
- Project: https://github.com/yt-dlp/yt-dlp
- License: The Unlicense (project code). Upstream notes that some release artifacts (e.g., Windows executable builds) may include components under other licenses (including GPLv3+). See upstream licensing notes and THIRD_PARTY_LICENSES.txt in yt-dlp releases.
- Usage: downloaded as an external binary (yt-dlp.exe) into the user’s local app data directory.

## FFmpeg
- Project: https://ffmpeg.org/
- License: FFmpeg is licensed under LGPL or GPL depending on build configuration and enabled components. See FFmpeg legal page: https://www.ffmpeg.org/legal.html
- Usage: downloaded as external binaries (ffmpeg.exe, ffprobe.exe) into the user’s local app data directory.
- Windows builds source: https://github.com/BtbN/FFmpeg-Builds (the app downloads a “win64-gpl-shared” build variant when using that source).

## Fyne
- Project: https://github.com/fyne-io/fyne
- License: BSD 3-Clause
- Usage: GUI framework.

Notes:
- Third-party binaries are not shipped in this repository by default; they are downloaded to the user’s local application data directory at runtime (unless a custom build embeds them).