package gui

import (
	"path/filepath"
	"runtime"
	"strings"
	"wago-init/internal/fs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

func BuildFirmwarePrompt(configValues *fs.EnvConfig, w fyne.Window) *widget.Button {
	firmwareBtn := widget.NewButton("Firmware settings", func() {
		values := fs.EnvConfig{}
		if configValues != nil && *configValues != nil {
			values = *configValues
		}

		revisionEntry := widget.NewEntry()
		revisionEntry.SetText(values[fs.FirmwareRevision])
		revisionEntry.SetPlaceHolder("e.g., 28")

		fileEntry := widget.NewEntry()
		fileEntry.SetText(values[fs.FirmwarePath])
		fileEntry.SetPlaceHolder("Select firmware update file (.wup)")

		browseBtn := widget.NewButton("Browse", nil)
		browseBtn.OnTapped = func() {
			fileDialog := dialog.NewFileOpen(func(read fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				if read == nil {
					return
				}
				defer read.Close()

				path := read.URI().Path()
				if runtime.GOOS == "windows" && strings.HasPrefix(path, "/") && len(path) > 2 && path[2] == ':' {
					path = path[1:]
				}
				resolved := filepath.Clean(filepath.FromSlash(path))
				fileEntry.SetText(resolved)
				if values == nil {
					values = fs.EnvConfig{}
				}
				values[fs.FirmwarePath] = resolved
			}, w)
			fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".wup"}))

			currentPath := strings.TrimSpace(fileEntry.Text)
			if currentPath != "" {
				dir := filepath.Dir(currentPath)
				uri := storage.NewFileURI(dir)
				if listURI, err := storage.ListerForURI(uri); err == nil {
					fileDialog.SetLocation(listURI)
				} else {
					fyne.LogError("failed to set initial firmware file location", err)
				}
			}

			fileDialog.Show()
		}

		fileRow := container.NewBorder(nil, nil, nil, browseBtn, fileEntry)

		content := container.NewVBox(
			widget.NewForm(widget.NewFormItem("Firmware Revision", revisionEntry)),
			widget.NewLabel("Firmware Update File"),
			fileRow,
		)

		dialogWindow := dialog.NewCustomConfirm(
			"Firmware Settings",
			"Save",
			"Cancel",
			content,
			func(ok bool) {
				if !ok {
					return
				}

				updated := make(fs.EnvConfig, len(values)+2)
				for key, value := range values {
					updated[key] = value
				}

				updated[fs.FirmwareRevision] = strings.TrimSpace(revisionEntry.Text)
				updated[fs.FirmwarePath] = strings.TrimSpace(fileEntry.Text)

				if err := fs.SaveConfig(updated); err != nil {
					dialog.ShowError(err, w)
					return
				}

				if configValues != nil {
					*configValues = updated
				}
			},
			w,
		)

		dialogWindow.Resize(fyne.NewSize(500, 260))
		dialogWindow.Show()
	})

	return firmwareBtn
}
