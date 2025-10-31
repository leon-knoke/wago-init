package gui

import (
	"wago-init/internal/fs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// mainView tracks the state of the primary application window.
type mainView struct {
	app                  fyne.App
	window               fyne.Window
	configValues         fs.EnvConfig
	ipEntry              *widget.Entry
	configPathEntry      *widget.Entry
	startBtn             *widget.Button
	passwordPrompt       func() (string, bool)
	newPasswordPrompt    func() (string, bool)
	containerSettingsBtn *widget.Button
	awsSettingsBtn       *widget.Button
	firmwareSettingsBtn  *widget.Button
	deviceDiscoveryBtn   *widget.Button
	sessions             []*installSession
	sessionsBox          *fyne.Container
	sessionsScroll       *container.Scroll
	deviceDiscoveryCache []discoveredDevice
}

func BuildMainWindow() {
	application := app.NewWithID("wago-init-app")
	window := application.NewWindow("Wago Init")

	configValues := loadInitialConfig()
	view := newMainView(application, window, configValues)

	view.buildContent()

	window.Resize(fyne.NewSize(1250, 800))
	window.ShowAndRun()
}

func newMainView(application fyne.App, window fyne.Window, configValues fs.EnvConfig) *mainView {
	if configValues == nil {
		configValues = fs.EnvConfig{}
	}

	return &mainView{
		app:               application,
		window:            window,
		configValues:      configValues,
		passwordPrompt:    passwordPromtFunc(window),
		newPasswordPrompt: newPasswordPromtFunc(window),
	}
}

func loadInitialConfig() fs.EnvConfig {
	configValues, err := fs.LoadConfig()
	if err != nil {
		fyne.LogError("failed to load configuration", err)
		configValues = fs.EnvConfig{}
	}

	if configValues == nil {
		configValues = fs.EnvConfig{}
	}

	return configValues
}

func (mv *mainView) runOnUI(fn func()) {
	fyne.Do(fn)
}

func GetClipboard(window fyne.Window) fyne.Clipboard {
	if window == nil {
		return nil
	}
	if app := fyne.CurrentApp(); app != nil {
		return app.Clipboard()
	}
	return nil
}
