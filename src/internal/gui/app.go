package gui

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"wago-init/internal/aws"
	"wago-init/internal/fs"
	"wago-init/internal/install"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

func BuildMainWindow() {
	a := app.NewWithID("wago-init-app")
	w := a.NewWindow("Wago Init")

	configValues, err := fs.LoadConfig()
	if err != nil {
		fyne.LogError("failed to load configuration", err)
		configValues = fs.EnvConfig{}
	}

	if configValues == nil {
		configValues = fs.EnvConfig{}
	}

	// First row: IP Address label (natural width) + expanding entry (Border layout)
	ipEntry := widget.NewEntry()
	ipEntry.SetText(configValues[fs.IpAddress])
	ipEntry.SetPlaceHolder(install.DefaultIp)
	ipLabel := widget.NewLabel("IP Address:")
	ipRow := container.NewBorder(nil, nil, ipLabel, nil, ipEntry)

	configPathEntry := widget.NewEntry()
	configPathEntry.SetText(configValues[fs.ConfigPath])
	configPathEntry.SetPlaceHolder("Select configuration path")

	searchBtn := widget.NewButton("Search", nil)
	searchBtn.OnTapped = func() {
		folderDialog := dialog.NewFolderOpen(func(list fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if list == nil {
				return
			}

			path := list.Path()
			if runtime.GOOS == "windows" && strings.HasPrefix(path, "/") && len(path) > 2 && path[2] == ':' {
				path = path[1:]
			}
			resolved := filepath.Clean(filepath.FromSlash(path))
			configPathEntry.SetText(resolved)
			configValues[fs.ConfigPath] = resolved
		}, w)

		currentPath := strings.TrimSpace(configPathEntry.Text)
		if currentPath != "" {
			uri := storage.NewFileURI(currentPath)
			if listURI, err := storage.ListerForURI(uri); err == nil {
				folderDialog.SetLocation(listURI)
			} else {
				fyne.LogError("failed to set initial folder", err)
			}
		}

		folderDialog.SetTitleText("Select configuration directory")
		folderDialog.Show()
	}

	awsSettingsBtn := BuildAWSPromt(&configValues, w)

	configRow := container.NewBorder(nil, nil, widget.NewLabel("Config path"), awsSettingsBtn,
		container.NewBorder(nil, nil, nil, searchBtn, configPathEntry),
	)

	// Progress bar
	progress := widget.NewProgressBar()
	progress.SetValue(0)

	// Log output label inside scroll
	outputLabel := widget.NewLabel("")
	outputLabel.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(outputLabel)
	scroll.SetMinSize(fyne.NewSize(400, 300))

	appendOutput := func(line string) {
		fyne.Do(func() {
			if outputLabel.Text == "" {
				outputLabel.SetText(line)
			} else {
				outputLabel.SetText(outputLabel.Text + "\n" + line)
			}
			w.Canvas().Refresh(outputLabel)
			scroll.ScrollToBottom()
		})
	}

	passwordPrompt := passwordPromtFunc(w)
	passwordNewPrompt := newPasswordPromtFunc(w)

	startBtn := widget.NewButton("Start", nil)

	startBtn.OnTapped = func() {
		ip := strings.TrimSpace(ipEntry.Text)
		startBtn.Disable()
		ipEntry.Disable()
		progress.SetValue(0)

		installParameters := install.Parameters{
			Ip:                ip,
			PromptPassword:    passwordPrompt,
			PromptNewPassword: passwordNewPrompt,
		}
		appendOutput("")
		go func(params install.Parameters) {
			updated := cloneEnvConfig(configValues)
			updated[fs.ConfigPath] = strings.TrimSpace(configPathEntry.Text)
			updated[fs.IpAddress] = strings.TrimSpace(ipEntry.Text)

			if err := fs.SaveConfig(updated); err != nil {
				fyne.Do(func() {
					dialog.ShowError(err, w)
					progress.SetValue(0)
					startBtn.Enable()
					ipEntry.Enable()
				})
				return
			}

			fyne.Do(func() {
				configValues = updated
				configPathEntry.SetText(updated[fs.ConfigPath])
			})

			awsAccessID := strings.TrimSpace(updated[fs.AWSAccessID])
			awsAccessKey := strings.TrimSpace(updated[fs.AWSAccessKey])
			awsRegion := strings.TrimSpace(updated[fs.AWSRegion])

			if awsRegion == "" || awsAccessID == "" || awsAccessKey == "" {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("please provide AWS region, access id, and access key before starting"), w)
					progress.SetValue(0)
					startBtn.Enable()
					ipEntry.Enable()
				})
				return
			}

			token, err := aws.FetchLoginPassword(context.Background(), awsRegion, awsAccessID, awsAccessKey)
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(err, w)
					progress.SetValue(0)
					startBtn.Enable()
					ipEntry.Enable()
				})
				return
			}
			appendOutput("Authorization with AWS successful")
			params.AWSToken = token
			err = install.Install(
				params,
				func(msg string) { fyne.Do(func() { appendOutput(msg) }) },
				func(p float64) { fyne.Do(func() { progress.SetValue(p) }) },
			)
			fyne.Do(func() {
				if err != nil {
					progress.SetValue(0)
					appendOutput("Error: " + err.Error())
				} else {
					progress.SetValue(1)
					appendOutput("Done.")
				}
				startBtn.Enable()
				ipEntry.Enable()
			})
		}(installParameters)
	}

	top := container.NewVBox(
		ipRow,
		configRow,
		startBtn,
		progress,
	)

	content := container.NewBorder(top, nil, nil, nil, scroll)

	w.SetContent(content)
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}

func cloneEnvConfig(src fs.EnvConfig) fs.EnvConfig {
	if src == nil {
		return fs.EnvConfig{}
	}
	dst := make(fs.EnvConfig, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
