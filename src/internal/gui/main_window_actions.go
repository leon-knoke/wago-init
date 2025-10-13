package gui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"wago-init/internal/aws"
	"wago-init/internal/fs"
	"wago-init/internal/install"

	"fyne.io/fyne/v2/dialog"
)

func (mv *mainView) handleStart() {
	mv.startBtn.Disable()
	mv.ipEntry.Disable()
	mv.progress.SetValue(0)
	mv.appendOutput("")

	params := install.Parameters{
		Ip:                strings.TrimSpace(mv.ipEntry.Text),
		PromptPassword:    mv.passwordPrompt,
		PromptNewPassword: mv.newPasswordPrompt,
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

	awsAccessID := strings.TrimSpace(updated[fs.AWSAccessID])
	awsAccessKey := strings.TrimSpace(updated[fs.AWSAccessKey])
	awsRegion := strings.TrimSpace(updated[fs.AWSRegion])

	if awsRegion == "" || awsAccessID == "" || awsAccessKey == "" {
		mv.failWithError(fmt.Errorf("please provide AWS region, access id, and access key before starting"))
		return
	}

	token, err := aws.FetchLoginPassword(context.Background(), awsRegion, awsAccessID, awsAccessKey)
	if err != nil {
		mv.failWithError(err)
		return
	}

	mv.appendOutput("Authorization with AWS successful")
	params.AWSToken = token

	err = install.Install(
		params,
		mv.appendOutput,
		mv.updateProgress,
	)

	mv.finishInstallation(err)
}

func (mv *mainView) updateProgress(value float64) {
	mv.runOnUI(func() {
		mv.progress.SetValue(value)
	})
}

func (mv *mainView) appendOutput(line string) {
	timestamp := time.Now().Format("15:04:05")
	formatted := fmt.Sprintf("[%s] %s", timestamp, line)
	mv.runOnUI(func() {
		if mv.outputText == "" {
			mv.outputText = formatted
		} else {
			mv.outputText += "\n" + formatted
		}
		mv.outputUpdating = true
		mv.outputEntry.SetText(mv.outputText)
		mv.outputUpdating = false
		mv.outputScroll.ScrollToBottom()
	})
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
		mv.appendOutput("Error: " + err.Error())
		return
	}

	mv.runOnUI(func() {
		mv.progress.SetValue(1)
		mv.startBtn.Enable()
		mv.ipEntry.Enable()
	})
	mv.appendOutput("Done.")
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
