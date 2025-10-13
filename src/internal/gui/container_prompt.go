package gui

import (
	"strings"
	"wago-init/internal/fs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func BuildContainerPrompt(configValues *fs.EnvConfig, w fyne.Window) *widget.Button {
	containerBtn := widget.NewButton("Container flags", func() {
		values := fs.EnvConfig{}
		if configValues != nil && *configValues != nil {
			values = *configValues
		}

		imageEntry := widget.NewEntry()
		imageEntry.SetText(values[fs.ContainerImage])

		commandEntry := widget.NewMultiLineEntry()
		commandEntry.SetText(fs.DecodeMultilineValue(values[fs.ContainerCommand]))
		commandEntry.Wrapping = fyne.TextWrapWord
		commandEntry.SetMinRowsVisible(24)

		imageForm := widget.NewForm(widget.NewFormItem("Image", imageEntry))
		flagsLabel := widget.NewLabel("Container Flags")
		flagsLabel.Alignment = fyne.TextAlignLeading
		content := container.NewVBox(
			imageForm,
			flagsLabel,
			commandEntry,
		)

		dialogWindow := dialog.NewCustomConfirm(
			"Container Settings",
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

				updated[fs.ContainerImage] = strings.TrimSpace(imageEntry.Text)
				updated[fs.ContainerCommand] = fs.EncodeMultilineValue(strings.TrimSpace(commandEntry.Text))

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
		dialogWindow.Resize(fyne.NewSize(1600, 800))
		dialogWindow.Show()
	})

	return containerBtn
}
