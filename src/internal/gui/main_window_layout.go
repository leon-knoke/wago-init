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
	mv.setupOutputArea()
	mv.setupStartButton()

	ipRow := mv.buildIPRow()
	configRow := mv.buildConfigRow()

	buttonSize := mv.startBtn.MinSize()
	buttonWrapper := container.NewGridWrap(fyne.NewSize(120, buttonSize.Height), mv.startBtn)
	progressWrapper := container.NewMax(mv.progress)
	startStack := container.NewStack(progressWrapper, buttonWrapper)
	mv.progress.Hide()

	top := container.NewVBox(
		ipRow,
		configRow,
		startStack,
	)

	content := container.NewBorder(top, nil, nil, nil, mv.outputScroll)
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

func (mv *mainView) setupOutputArea() {
	mv.progress = widget.NewProgressBar()
	mv.progress.SetValue(0)

	mv.outputEntry = widget.NewMultiLineEntry()
	mv.outputEntry.SetMinRowsVisible(14)
	mv.outputEntry.Wrapping = fyne.TextWrapWord
	mv.outputEntry.TextStyle = fyne.TextStyle{Monospace: true}
	mv.outputEntry.OnChanged = func(s string) {
		if mv.outputUpdating {
			return
		}
		mv.outputUpdating = true
		mv.refreshOutputEntryLocked()
		mv.outputUpdating = false
	}

	mv.outputScroll = container.NewVScroll(mv.outputEntry)
	mv.outputScroll.SetMinSize(fyne.NewSize(400, 300))
}

func (mv *mainView) setupStartButton() {
	mv.startBtn = widget.NewButton("Start", mv.handleStart)
}

func (mv *mainView) buildIPRow() fyne.CanvasObject {
	ipLabel := widget.NewLabel("IP Address  ")
	ipControls := container.NewBorder(nil, nil, nil, container.NewHBox(widget.NewLabel(" "), mv.deviceDiscoveryBtn), mv.ipEntry)
	right := container.NewHBox(
		widget.NewLabel("  "),
		mv.containerSettingsBtn,
		widget.NewLabel("  "),
		mv.awsSettingsBtn,
		widget.NewLabel("  "),
		mv.firmwareSettingsBtn,
	)
	return container.NewBorder(nil, nil, ipLabel, right, ipControls)
}

func (mv *mainView) buildConfigRow() fyne.CanvasObject {
	searchBtn := widget.NewButton("Search", mv.openConfigFolderDialog)
	entryContainer := container.NewBorder(nil, nil, nil, searchBtn, mv.configPathEntry)
	return container.NewBorder(nil, nil, widget.NewLabel("Config path"), nil, entryContainer)
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
