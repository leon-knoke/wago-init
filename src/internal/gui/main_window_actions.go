package gui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
	"wago-init/internal/aws"
	"wago-init/internal/fs"
	"wago-init/internal/install"

	"fyne.io/fyne/v2/dialog"
)

const progressDelayInterval = 12090 * time.Millisecond

func (mv *mainView) handleStart() {
	mv.startBtn.Disable()
	mv.ipEntry.Disable()
	mv.progress.SetValue(0)
	mv.appendOutput("", "")

	fwRevisionRaw := strings.TrimSpace(mv.configValues[fs.FirmwareRevision])
	fwTarget := 0
	if fwRevisionRaw != "" {
		if num, err := strconv.Atoi(fwRevisionRaw); err == nil {
			fwTarget = num
		} else {
			mv.appendOutput(fmt.Sprintf("Warning: firmware revision '%s' is not numeric; skipping automatic comparison", fwRevisionRaw), "")
		}
	}

	params := install.Parameters{
		Ip:                strings.TrimSpace(mv.ipEntry.Text),
		FirmwareRevision:  fwRevisionRaw,
		PromptPassword:    mv.passwordPrompt,
		PromptNewPassword: mv.newPasswordPrompt,
		ContainerImage:    mv.configValues[fs.ContainerImage],
		ContainerFlags:    install.BuildContainerCommand(mv.configValues[fs.ContainerCommand]),
		NewestFirmware:    fwTarget,
		FirmwarePath:      strings.TrimSpace(mv.configValues[fs.FirmwarePath]),
	}

	go mv.runInstallation(params)
}

func (mv *mainView) runInstallation(params install.Parameters) {
	updated := cloneEnvConfig(mv.configValues)
	updated[fs.ConfigPath] = strings.TrimSpace(mv.configPathEntry.Text)
	updated[fs.IpAddress] = strings.TrimSpace(mv.ipEntry.Text)

	if err := fs.SaveConfig(updated); err != nil {
		mv.failWithError(err)
		return
	}

	mv.runOnUI(func() {
		mv.configValues = updated
		mv.configPathEntry.SetText(updated[fs.ConfigPath])
	})

	awsRegion := strings.TrimSpace(updated[fs.AWSRegion])
	awsAccountID := strings.TrimSpace(updated[fs.AWSAccountID])
	awsAccessID := strings.TrimSpace(updated[fs.AWSAccessID])
	awsAccessKey := strings.TrimSpace(updated[fs.AWSAccessKey])

	if awsRegion == "" || awsAccessID == "" || awsAccessKey == "" || awsAccountID == "" {
		mv.failWithError(fmt.Errorf("please provide AWS region, account id, access id, and access key before starting"))
		return
	}

	token, err := aws.FetchLoginPassword(context.Background(), awsRegion, awsAccessID, awsAccessKey)
	if err != nil {
		mv.failWithError(err)
		return
	}
	ecrUrl := aws.GetEcrUrl(awsAccountID, awsRegion)

	mv.appendOutput("Authorization with AWS successful", "")
	params.AWSToken = token
	params.AWSEcrUrl = ecrUrl
	params.ConfigPath = strings.TrimSpace(mv.configValues[fs.ConfigPath])

	err = install.Install(
		params,
		mv.appendOutput,
		mv.updateProgress,
	)

	mv.finishInstallation(err)
}

func (mv *mainView) updateProgress(value float64, targetValue float64) {
	mv.runOnUI(func() {
		mv.progress.SetValue(value)
	})
	if value >= targetValue {
		return
	}
	go func(startValue float64, target float64) {
		current := startValue
		for {
			time.Sleep(progressDelayInterval)
			mv.runOnUI(func() {
				actual := mv.progress.Value
				if actual != current {
					return
				}
				if current >= target {
					mv.progress.SetValue(target)
					return
				}
				current += 0.01
				if current > target {
					current = target
				}
				mv.progress.SetValue(current)
			})
		}
	}(value, targetValue)
}

func (mv *mainView) appendOutput(line string, replaceIdentifier string) {
	formatted := ""
	if line != formatted {
		formatted = fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), line)
	}
	mv.runOnUI(func() {
		if mv.outputText == "" {
			mv.outputText = formatted
		} else {
			mv.outputText += "\n" + formatted
		}
		mv.outputUpdating = true
		mv.refreshOutputEntryLocked()
		mv.outputUpdating = false
		mv.outputScroll.ScrollToBottom()
	})
}

func (mv *mainView) refreshOutputEntryLocked() {
	mv.outputEntry.SetText(mv.outputText)
	rowCount := strings.Count(mv.outputText, "\n")
	mv.outputEntry.CursorRow = rowCount

	lastLine := mv.outputText
	if idx := strings.LastIndex(mv.outputText, "\n"); idx >= 0 {
		if idx+1 < len(mv.outputText) {
			lastLine = mv.outputText[idx+1:]
		} else {
			lastLine = ""
		}
	}
	mv.outputEntry.CursorColumn = utf8.RuneCountInString(lastLine)
	mv.outputEntry.Refresh()
}

func (mv *mainView) failWithError(err error) {
	mv.runOnUI(func() {
		dialog.ShowError(err, mv.window)
		mv.progress.SetValue(0)
		mv.startBtn.Enable()
		mv.ipEntry.Enable()
	})
}

func (mv *mainView) finishInstallation(err error) {
	if err != nil {
		mv.runOnUI(func() {
			mv.progress.SetValue(0)
			mv.startBtn.Enable()
			mv.ipEntry.Enable()
		})
		mv.appendOutput("Error: "+err.Error(), "")
		return
	}

	mv.runOnUI(func() {
		mv.progress.SetValue(1)
		mv.startBtn.Enable()
		mv.ipEntry.Enable()
	})
	mv.appendOutput("Done.", "")
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
