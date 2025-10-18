package install

import (
	"archive/zip"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	firmwareRemoteDir            = "/home/update"
	firmwareStartCommand         = "/etc/config-tools/fwupdate start --path"
	firmwareActivateCommand      = "/etc/config-tools/fwupdate activate [--keep-application]"
	firmwareCancelCommand        = "/etc/config-tools/fwupdate cancel"
	firmwareStatusCommand        = "/etc/config-tools/fwupdate status"
	firmwareFinishCommand        = "/etc/config-tools/fwupdate finish"
	firmwareReconnectTimeout     = 6 * time.Minute
	firmwareReconnectInterval    = 10 * time.Second
	firmwareUnzipTimeout         = 2 * time.Minute
	firmwareActivateTimeout      = 5 * time.Minute
	firmwareStartTimeout         = 15 * time.Minute
	firmwareFinishTimeout        = 5 * time.Minute
	firmwareLogPollIntervalLong  = 10 * time.Second
	firmwareLogPollIntervalShort = 5 * time.Second
)

func UpdateFirmware(client *ssh.Client, logFn func(string, string), params *Parameters, progressFn func(float64, float64)) (*ssh.Client, error) {

	progressFn(0.01, 0.07)

	localPath := strings.TrimSpace(params.FirmwarePath)
	if localPath == "" {
		return client, errors.New("firmware path is not configured")
	}
	if err := validateFirmwareFile(localPath); err != nil {
		return client, err
	}

	if _, err := runSSHCommand(client, "rm -rf /home/update/* && mkdir -p /home/update", longSessionTimeout); err != nil {
		return client, fmt.Errorf("prepare remote firmware directory: %w", err)
	}

	logFn("Uploading firmware package to device", "")
	if err := CopyPathToDevice(client, localPath, firmwareRemoteDir, logFn); err != nil {
		return client, fmt.Errorf("upload firmware: %w", err)
	}

	progressFn(0.08, 0.08)

	remoteFileName := filepath.Base(localPath)
	remoteFilePath := path.Join(firmwareRemoteDir, remoteFileName)

	unzipCmd := fmt.Sprintf("cd %s && unzip -o %s", shellQuote(firmwareRemoteDir), shellQuote(remoteFileName))
	logFn("Extracting firmware package on device", "")
	if err := runSSHCommandStreaming(client, unzipCmd, firmwareUnzipTimeout, logFn); err != nil {
		return client, fmt.Errorf("unzip firmware: %w", err)
	}

	if _, err := runSSHCommand(client, fmt.Sprintf("rm -f %s", shellQuote(remoteFilePath)), shortSessionTimeout); err != nil {
		return client, fmt.Errorf("cleanup firmware archive: %w", err)
	}

	progressFn(0.09, 0.09)

	logFn("Activating firmware daemon", "")
	if err := runSSHCommandStreaming(client, firmwareActivateCommand, firmwareActivateTimeout, logFn); err != nil {
		runSSHCommandStreaming(client, firmwareCancelCommand, firmwareActivateTimeout, logFn)
		return client, fmt.Errorf("fwupdate activate: %w", err)
	}

	if err := monitorFirmwareInitialization(client); err != nil {
		return client, err
	}

	progressFn(0.10, 0.30)

	startCmd := fmt.Sprintf("%s %s", firmwareStartCommand, firmwareRemoteDir)
	startErr := runSSHCommandStreaming(client, startCmd, firmwareStartTimeout, logFn)
	if startErr != nil {
		runSSHCommandStreaming(client, firmwareCancelCommand, firmwareActivateTimeout, logFn)
		return client, fmt.Errorf("fwupdate start: %w", startErr)
	}
	logFn("Firmware update initiated, monitoring device status...", "")
	if err := monitorFirmwareProgress(client, logFn, progressFn); err != nil {
		return client, err
	}

	logFn("Device connection lost, waiting for reboot to complete...", "")

	progressFn(0.31, 0.41)

	_ = client.Close()

	logFn("Waiting for device to come back online after reboot...", "")
	newClient, newPassword, err := reconnectAfterFirmware(params, logFn)
	if err != nil {
		return client, err
	}
	params.CurrentPassword = newPassword

	progressFn(0.59, 0.59)

	if err := monitorFirmwareFinalization(newClient, logFn); err != nil {
		return client, err
	}

	logFn("Finalising firmware update", "")
	if err := runSSHCommandStreaming(newClient, firmwareFinishCommand, firmwareFinishTimeout, logFn); err != nil {
		return newClient, fmt.Errorf("fwupdate finish: %w", err)
	}

	stillRequired, err := CheckFirmware(newClient, logFn, params.NewestFirmware)
	if err != nil {
		return newClient, err
	}
	if stillRequired {
		logFn("Firmware update did not complete successfully; firmware update is still required", "")
	} else {
		logFn("Firmware update completed successfully", "")
	}

	return newClient, nil
}

func validateFirmwareFile(localPath string) error {
	if strings.ToLower(filepath.Ext(localPath)) != ".wup" {
		return fmt.Errorf("firmware file must have .wup extension: %s", localPath)
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("stat firmware file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("firmware path points to a directory: %s", localPath)
	}

	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open firmware file: %w", err)
	}
	defer file.Close()

	if _, err := zip.NewReader(file, info.Size()); err != nil {
		return fmt.Errorf("firmware file is not a valid zip archive: %w", err)
	}

	return nil
}

func monitorFirmwareInitialization(client *ssh.Client) error {
	for {
		output, err := runSSHCommand(client, firmwareStatusCommand, longSessionTimeout)
		if err != nil {
			return err
		}
		if strings.Contains(strings.ToLower(output), "status=prepared") {
			return nil
		}
		time.Sleep(firmwareLogPollIntervalShort)
	}
}

func monitorFirmwareProgress(client *ssh.Client, logFn func(string, string), progressFn func(float64, float64)) error {
	var lastStatusLine string

	for {
		output, err := runSSHCommand(client, firmwareStatusCommand, longSessionTimeout)
		if err != nil {
			logFn("Stopped receiving firmware status updates; device is likely rebooting.", "")
			return nil
		}

		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}

			lower := strings.ToLower(trimmed)
			if trimmed != lastStatusLine {
				logFn(trimmed, "")
				lastStatusLine = trimmed
			}

			if strings.Contains(lower, "status=error") {
				return fmt.Errorf("firmware update reported error: %s", trimmed)
			}
		}

		time.Sleep(firmwareLogPollIntervalShort)
	}
}

func monitorFirmwareFinalization(client *ssh.Client, logFn func(string, string)) error {
	const maxTransientErrors = 6

	var (
		errorCount     int
		lastStatusLine string
	)

	for {
		time.Sleep(firmwareLogPollIntervalShort)

		output, err := runSSHCommand(client, firmwareStatusCommand, longSessionTimeout)
		if err != nil {
			errorCount++
			if errorCount > maxTransientErrors {
				return fmt.Errorf("monitor firmware finalization: %w", err)
			}
			logFn(fmt.Sprintf("Lost connection while checking firmware status (%d/%d); retrying...", errorCount, maxTransientErrors), "")
			time.Sleep(firmwareLogPollIntervalShort)
		}

		errorCount = 0

		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}

			lower := strings.ToLower(trimmed)
			if trimmed != lastStatusLine {
				logFn("Firmware status: "+trimmed, "")
				lastStatusLine = trimmed
			}

			if strings.Contains(lower, "status=error") {
				return fmt.Errorf("firmware update finalization reported error: %s", trimmed)
			}

			if strings.Contains(lower, "status=unconfirmed") || strings.Contains(lower, "status=idle") || strings.Contains(lower, "status=finished") {
				return nil
			}
		}
	}
}

func reconnectAfterFirmware(params *Parameters, logFn func(string, string)) (*ssh.Client, string, error) {
	deadline := time.Now().Add(firmwareReconnectTimeout)
	password := params.CurrentPassword
	addr := net.JoinHostPort(params.Ip, "22")

	for time.Now().Before(deadline) {
		if password != "" {
			client, err := dialSSH(addr, password)
			if err == nil {
				logFn("Reconnected to device using stored credentials", "")
				return client, password, nil
			}
			if isAuthError(err) {
				logFn("Stored password rejected, requesting password from user", "")
				password = ""
			}
		}

		client, pwd, err := InitSshClient(params.Ip, params.PromptPassword)
		if err == nil {
			logFn("Reconnected to device after reboot", "")
			return client, pwd, nil
		}

		time.Sleep(firmwareReconnectInterval)
	}

	return nil, password, errors.New("timed out waiting for device to reboot")
}
