package gui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"wago-init/internal/aws"
	"wago-init/internal/fs"
	"wago-init/internal/install"

	"fyne.io/fyne/v2/dialog"
)

func (mv *mainView) handleStart() {
	var once sync.Once
	unlockStart := func() {
		once.Do(func() {
			mv.runOnUI(func() {
				if mv.startBtn != nil {
					mv.startBtn.Enable()
				}
			})
		})
	}

	if mv.startBtn != nil {
		mv.startBtn.Disable()
	}

	fwRevisionRaw := strings.TrimSpace(mv.configValues[fs.FirmwareRevision])
	fwTarget := 0
	var fwWarning string
	if fwRevisionRaw != "" {
		if num, err := strconv.Atoi(fwRevisionRaw); err == nil {
			fwTarget = num
		} else {
			fwWarning = fmt.Sprintf("Warning: firmware revision '%s' is not numeric; skipping automatic comparison", fwRevisionRaw)
		}
	}

	ip := strings.TrimSpace(mv.ipEntry.Text)
	if ip == "" {
		ip = install.DefaultIp
	}

	if mv.hasActiveSessionForIP(ip) {
		unlockStart()
		dialog.ShowError(fmt.Errorf("an installation for %s already exists. Remove it before starting another.", ip), mv.window)
		return
	}

	params := install.Parameters{
		Ip:                ip,
		FirmwareRevision:  fwRevisionRaw,
		PromptPassword:    mv.passwordPrompt,
		PromptNewPassword: mv.newPasswordPrompt,
		ContainerImage:    mv.configValues[fs.ContainerImage],
		ContainerFlags:    install.BuildContainerCommand(mv.configValues[fs.ContainerCommand]),
		NewestFirmware:    fwTarget,
		FirmwarePath:      strings.TrimSpace(mv.configValues[fs.FirmwarePath]),
		ForceFirmware:     strings.TrimSpace(mv.configValues[fs.ForceFirmwareUpdate]) == "true",
	}

	updated := cloneEnvConfig(mv.configValues)
	updated[fs.ConfigPath] = strings.TrimSpace(mv.configPathEntry.Text)
	updated[fs.IpAddress] = ip

	awsRegion := strings.TrimSpace(updated[fs.AWSRegion])
	awsAccountID := strings.TrimSpace(updated[fs.AWSAccountID])
	awsAccessID := strings.TrimSpace(updated[fs.AWSAccessID])
	awsAccessKey := strings.TrimSpace(updated[fs.AWSAccessKey])

	if awsRegion == "" || awsAccessID == "" || awsAccessKey == "" || awsAccountID == "" {
		unlockStart()
		dialog.ShowError(fmt.Errorf("please provide AWS region, account id, access id, and access key before starting"), mv.window)
		return
	}

	session := mv.newInstallSession(ip)
	session.setStartUnlocker(unlockStart)

	originalPasswordPrompt := params.PromptNewPassword
	params.PromptNewPassword = func() (string, bool) {
		value, ok := originalPasswordPrompt()
		if ok {
			session.unlockStart()
		}
		return value, ok
	}

	session.appendLog(fmt.Sprintf("Installation started for %s", ip), "")
	if fwWarning != "" {
		session.appendLog(fwWarning, "")
	}
	session.appendLog("Preparing installation...", "")

	params.Context = session.ctx
	params.ConfigPath = updated[fs.ConfigPath]

	go mv.runInstallationSession(session, params, updated, awsRegion, awsAccountID, awsAccessID, awsAccessKey)
}

func (mv *mainView) runInstallationSession(session *installSession, params install.Parameters, updated fs.EnvConfig, awsRegion, awsAccountID, awsAccessID, awsAccessKey string) {
	if err := fs.SaveConfig(updated); err != nil {
		session.reportFailure(err)
		return
	}

	mv.runOnUI(func() {
		mv.configValues = updated
		mv.configPathEntry.SetText(updated[fs.ConfigPath])
	})

	session.appendLog("Configuration saved", "")

	token, err := aws.FetchLoginPassword(session.ctx, awsRegion, awsAccessID, awsAccessKey)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			session.reportCancellation()
			return
		}
		session.reportFailure(err)
		return
	}

	params.AWSToken = token
	params.AWSEcrUrl = aws.GetEcrUrl(awsAccountID, awsRegion)

	session.appendLog("Authorization with AWS successful", "")

	if session.ctx.Err() != nil {
		session.reportCancellation()
		return
	}

	err = install.Install(
		params,
		session.appendLog,
		session.updateProgress,
	)

	switch {
	case err == nil:
		session.reportSuccess()
	case errors.Is(err, context.Canceled):
		session.reportCancellation()
	default:
		session.reportFailure(err)
	}
}

func (mv *mainView) hasActiveSessionForIP(ip string) bool {
	for _, session := range mv.sessions {
		if session.ip == ip {
			return true
		}
	}
	return false
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
