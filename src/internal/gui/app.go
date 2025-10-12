package gui

import (
	"wago-init/internal/install"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func BuildMainWindow() {
	a := app.New()
	w := a.NewWindow("Installer Utility")

	// First row: IP Address label (natural width) + expanding entry (Border layout)
	ipEntry := widget.NewEntry()
	ipEntry.SetPlaceHolder(install.DefaultIp)
	ipLabel := widget.NewLabel("IP Address:")
	ipRow := container.NewBorder(nil, nil, ipLabel, nil, ipEntry)

	// Progress bar
	progress := widget.NewProgressBar()
	progress.SetValue(0)

	// Log output label inside scroll
	outputLabel := widget.NewLabel("")
	outputLabel.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(outputLabel)
	scroll.SetMinSize(fyne.NewSize(400, 300))

	appendOutput := func(line string) {
		if outputLabel.Text == "" {
			outputLabel.SetText(line)
		} else {
			outputLabel.SetText(outputLabel.Text + "\n" + line)
		}
		w.Canvas().Refresh(outputLabel)
		scroll.ScrollToBottom()
	}

	startBtn := widget.NewButton("Start", nil)

	startBtn.OnTapped = func() {
		ip := ipEntry.Text
		startBtn.Disable()
		ipEntry.Disable()
		progress.SetValue(0)

		installParameters := install.Parameters{
			Ip: ip,
		}
		appendOutput("")
		go func(params install.Parameters) {
			err := install.Install(
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
		startBtn,
		progress,
	)

	content := container.NewBorder(top, nil, nil, nil, scroll)

	w.SetContent(content)
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}
