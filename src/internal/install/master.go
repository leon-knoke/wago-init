package install

import (
	"errors"
	"fmt"
	"strings"
)

func Install(installParameters Parameters, logFn func(string, string), progressFn func(float64, float64)) error {

	params, err := validateParameters(installParameters)
	if err != nil {
		return err
	}
	if err := checkCancellation(params.Context); err != nil {
		return err
	}
	logFn("Starting process for IP: "+params.Ip, "")

	err = CheckMacAddress(params, logFn)
	if err != nil {
		return err
	}
	if err := checkCancellation(params.Context); err != nil {
		return err
	}

	ip := params.Ip
	client, password, err := InitSshClient(ip, params.PromptPassword)
	if err != nil {
		return err
	}
	params.CurrentPassword = password
	logFn("Connection to device established", "")
	defer client.Close()

	if err := checkCancellation(params.Context); err != nil {
		return err
	}

	err = CheckSerialNumber(client, logFn)
	if err != nil {
		return err
	}
	if err := checkCancellation(params.Context); err != nil {
		return err
	}

	err = CheckCalibrationData(client)
	if err != nil {
		return err
	}

	logFn("Asking for new user password", "")
	newPassword, ok := params.PromptNewPassword()
	if !ok {
		return errors.New("new password prompt cancelled by user")
	}
	logFn("Received new password from user", "")
	if err := checkCancellation(params.Context); err != nil {
		return err
	}

	err = ChangeUserPasswords(client, logFn, newPassword)
	if err != nil {
		return err
	}

	progressFn(0.01, 0.01)

	fwUpdateRequired, err := CheckFirmware(client, logFn, params.NewestFirmware)
	if err != nil {
		return err
	}
	if fwUpdateRequired || params.ForceFirmware {
		logFn("Pending firmware update. Starting...", "")
		client, err = UpdateFirmware(client, logFn, &params, progressFn)
		if err != nil {
			return err
		}
	}
	if err := checkCancellation(params.Context); err != nil {
		return err
	}
	progressFn(0.6, 0.64)

	err = ConfigureServices(client, logFn)
	if err != nil {
		return err
	}
	if err := checkCancellation(params.Context); err != nil {
		return err
	}
	progressFn(0.65, 0.99)

	err = CreateContainer(client, logFn, params)
	if err != nil {
		return err
	}
	if err := checkCancellation(params.Context); err != nil {
		return err
	}

	if err := CopyPathToDevice(client, params.Context, params.ConfigPath, "/root", logFn); err != nil {
		return err
	}

	logFn("Installation complete.", "")
	progressFn(1, 1)

	return nil
}

func validateParameters(params Parameters) (Parameters, error) {
	if params.Ip == "" {
		params.Ip = DefaultIp
	}

	parts := strings.Split(params.Ip, ".")
	if len(parts) != 4 {
		return params, errors.New("invalid number of octets in IP address")
	}

	for _, part := range parts {
		var num int
		_, err := fmt.Sscanf(part, "%d", &num)
		if err != nil || num < 0 || num > 255 {
			return params, errors.New("invalid octet in IP address")
		}
	}

	return params, nil
}
