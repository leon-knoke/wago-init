package gui

import (
	"path/filepath"
	"runtime"
	"strings"
	"wago-init/internal/fs"
	"wago-init/internal/install"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

func (mv *mainView) buildContent() {
	mv.setupEntries()
	mv.setupButtons()
	mv.setupSessionsArea()
	mv.setupStartButton()

	ipLabel := widget.NewLabel("IP Address:")
	ipControls := container.NewBorder(nil, nil, ipLabel, mv.deviceDiscoveryBtn, mv.ipEntry)
	settingsSection := container.NewVBox(
		mv.firmwareSettingsBtn,
		mv.containerSettingsBtn,
		mv.awsSettingsBtn,
	)

	searchBtn := widget.NewButton("Search", mv.openConfigFolderDialog)
	entryContainer := container.NewBorder(nil, nil, nil, searchBtn, mv.configPathEntry)

	// ipRow := container.NewBorder(nil, nil, ipLabel, right, ipControls)
	configRow := container.NewBorder(nil, nil, widget.NewLabel("Copy path: "), nil, entryContainer)

	left := container.NewVBox(
		ipControls,
		configRow,
		mv.startBtn,
	)

	right := container.NewHBox(
		widget.NewSeparator(),
		widget.NewLabel("  "),
		widget.NewSeparator(),
		settingsSection,
	)

	top := container.NewBorder(nil, widget.NewSeparator(), nil, right, left)

	content := container.NewBorder(top, nil, nil, nil, mv.sessionsScroll)
	mv.window.SetContent(content)
}

func (mv *mainView) setupEntries() {
	mv.ipEntry = widget.NewEntry()
	mv.ipEntry.SetText(mv.configValues[fs.IpAddress])
	mv.ipEntry.SetPlaceHolder(install.DefaultIp)

	mv.configPathEntry = widget.NewEntry()
	mv.configPathEntry.SetText(mv.configValues[fs.ConfigPath])
	mv.configPathEntry.SetPlaceHolder("Select configuration path")
}

func (mv *mainView) setupButtons() {
	mv.awsSettingsBtn = BuildAWSPromt(&mv.configValues, mv.window)
	mv.containerSettingsBtn = BuildContainerPrompt(&mv.configValues, mv.window)
	mv.firmwareSettingsBtn = BuildFirmwarePrompt(&mv.configValues, mv.window)
	mv.deviceDiscoveryBtn = BuildDeviceDiscoveryPrompt(mv)
}

func (mv *mainView) setupSessionsArea() {
	mv.sessionsBox = container.NewVBox()
	mv.sessionsScroll = container.NewVScroll(mv.sessionsBox)
	mv.sessionsScroll.SetMinSize(fyne.NewSize(400, 300))
}

func (mv *mainView) setupStartButton() {
	mv.startBtn = widget.NewButton("Start", mv.handleStart)
}

func (mv *mainView) openConfigFolderDialog() {
	folderDialog := dialog.NewFolderOpen(func(list fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, mv.window)
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
		mv.configPathEntry.SetText(resolved)
		if mv.configValues == nil {
			mv.configValues = fs.EnvConfig{}
		}
		mv.configValues[fs.ConfigPath] = resolved
	}, mv.window)

	currentPath := strings.TrimSpace(mv.configPathEntry.Text)
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
