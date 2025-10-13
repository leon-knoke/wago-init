package gui

import (
	"strings"
	"wago-init/internal/fs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func BuildAWSPromt(configValues *fs.EnvConfig, w fyne.Window) *widget.Button {
	awsSettingsBtn := widget.NewButton("AWS settings", func() {
		values := fs.EnvConfig{}
		if configValues != nil && *configValues != nil {
			values = *configValues
		}

		awsRegionEntry := widget.NewEntry()
		awsRegionEntry.SetText(values[fs.AWSRegion])

		accessIDEntry := widget.NewEntry()
		accessIDEntry.SetText(values[fs.AWSAccessID])

		accessKeyEntry := widget.NewPasswordEntry()
		accessKeyEntry.SetText(values[fs.AWSAccessKey])

		form := widget.NewForm(
			widget.NewFormItem("AWS Region", awsRegionEntry),
			widget.NewFormItem("Access ID", accessIDEntry),
			widget.NewFormItem("Access Key", accessKeyEntry),
		)

		d := dialog.NewCustomConfirm(
			"AWS Settings",
			"Save",
			"Cancel",
			form,
			func(ok bool) {
				if !ok {
					return
				}

				updated := make(fs.EnvConfig, len(values)+3)
				for key, value := range values {
					updated[key] = value
				}

				updated[fs.AWSRegion] = strings.TrimSpace(awsRegionEntry.Text)
				updated[fs.AWSAccessID] = strings.TrimSpace(accessIDEntry.Text)
				updated[fs.AWSAccessKey] = strings.TrimSpace(accessKeyEntry.Text)

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
		d.Resize(fyne.NewSize(500, 250))
		d.Show()
	})
	return awsSettingsBtn
}
