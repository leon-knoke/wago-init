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
	firmwareRemoteDir         = "/home/update"
	firmwareReconnectTimeout  = 6 * time.Minute
	firmwareReconnectInterval = 10 * time.Second
	firmwareInitialWait       = 10 * time.Second
	firmwareUnzipTimeout      = 2 * time.Minute
	firmwareActivateTimeout   = 5 * time.Minute
	firmwareStartTimeout      = 15 * time.Minute
	firmwareFinishTimeout     = 5 * time.Minute
	firmwareLogPollInterval   = 10 * time.Second
)

func UpdateFirmware(client *ssh.Client, logFn func(string), params *Parameters, progressFn func(float64)) (*ssh.Client, error) {
	if params == nil {
		return client, errors.New("firmware parameters are nil")
	}
	if client == nil {
		return client, errors.New("ssh client is nil")
	}
	if logFn == nil {
		logFn = func(string) {}
	}

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

	progressFn(0.1)

	logFn("Uploading firmware package to device")
	if err := CopyPathToDevice(client, localPath, firmwareRemoteDir, logFn); err != nil {
		return client, fmt.Errorf("upload firmware: %w", err)
	}

	progressFn(0.2)

	remoteFileName := filepath.Base(localPath)
	remoteFilePath := path.Join(firmwareRemoteDir, remoteFileName)

	unzipCmd := fmt.Sprintf("cd %s && unzip -o %s", shellQuote(firmwareRemoteDir), shellQuote(remoteFileName))
	logFn("Extracting firmware package on device")
	if err := runSSHCommandStreaming(client, unzipCmd, firmwareUnzipTimeout, logFn); err != nil {
		return client, fmt.Errorf("unzip firmware: %w", err)
	}

	progressFn(0.24)

	if _, err := runSSHCommand(client, fmt.Sprintf("rm -f %s", shellQuote(remoteFilePath)), shortSessionTimeout); err != nil {
		return client, fmt.Errorf("cleanup firmware archive: %w", err)
	}

	progressFn(0.25)

	logFn("Activating firmware daemon")
	if err := runSSHCommandStreaming(client, "/etc/config-tools/fwupdate activate [--keep-application]", firmwareActivateTimeout, logFn); err != nil {
		runSSHCommandStreaming(client, "/etc/config-tools/fwupdate cancel", firmwareActivateTimeout, logFn)
		return client, fmt.Errorf("fwupdate activate: %w", err)
	}

	time.Sleep(firmwareInitialWait)

	progressFn(0.27)

	startCmd := fmt.Sprintf("/etc/config-tools/fwupdate start --path %s", firmwareRemoteDir)
	startErr := runSSHCommandStreaming(client, startCmd, firmwareStartTimeout, logFn)
	if startErr != nil {
		runSSHCommandStreaming(client, "/etc/config-tools/fwupdate cancel", firmwareActivateTimeout, logFn)
		return client, fmt.Errorf("fwupdate start: %w", startErr)
	}
	logFn("Firmware update initiated, monitoring device status...")
	if err := monitorFirmwareProgress(client, logFn); err != nil {
		return client, err
	}

	logFn("Device connection lost, waiting for reboot to complete...")

	progressFn(0.40)

	_ = client.Close()

	time.Sleep(firmwareInitialWait)

	logFn("Waiting for device to come back online after reboot...")
	newClient, newPassword, err := reconnectAfterFirmware(params, logFn)
	if err != nil {
		return client, err
	}
	params.CurrentPassword = newPassword

	progressFn(0.59)

	finishCmd := "/etc/config-tools/fwupdate finish"
	logFn("Finalising firmware update")
	if err := runSSHCommandStreaming(newClient, finishCmd, firmwareFinishTimeout, logFn); err != nil {
		return newClient, fmt.Errorf("fwupdate finish: %w", err)
	}

	stillRequired, err := CheckFirmware(newClient, logFn, params.NewestFirmware)
	if err != nil {
		return newClient, err
	}
	if stillRequired {
		return newClient, errors.New("firmware update did not reach target revision")
	}

	logFn("Firmware update completed successfully")
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

func monitorFirmwareProgress(client *ssh.Client, logFn func(string)) error {
	for {
		output, err := runSSHCommand(client, "/etc/config-tools/fwupdate status", longSessionTimeout)
		if err != nil {
			return nil
		}
		if strings.Contains(strings.ToLower(output), "status=error") {
			return fmt.Errorf("firmware update reported error: %s", output)
		}

		time.Sleep(firmwareLogPollInterval)
	}
}

func reconnectAfterFirmware(params *Parameters, logFn func(string)) (*ssh.Client, string, error) {
	deadline := time.Now().Add(firmwareReconnectTimeout)
	password := params.CurrentPassword
	addr := net.JoinHostPort(params.Ip, "22")

	for time.Now().Before(deadline) {
		if password != "" {
			client, err := dialSSH(addr, password)
			if err == nil {
				logFn("Reconnected to device using stored credentials")
				return client, password, nil
			}
			if isAuthError(err) {
				logFn("Stored password rejected, requesting password from user")
				password = ""
			} else {
				logFn(fmt.Sprintf("Reconnect attempt failed: %v", err))
			}
		}

		client, pwd, err := InitSshClient(params.Ip, params.PromptPassword)
		if err == nil {
			logFn("Reconnected to device after reboot")
			return client, pwd, nil
		}
		logFn(fmt.Sprintf("Reconnect attempt failed: %v", err))

		time.Sleep(firmwareReconnectInterval)
	}

	return nil, password, errors.New("timed out waiting for device to reboot")
}
