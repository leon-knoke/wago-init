package install

import (
	"errors"
	"fmt"
	"strings"
)

func Install(installParameters Parameters, logFn func(string), progressFn func(float64)) error {
	progressFn(0)
	params, err := validateParameters(installParameters)
	if err != nil {
		return err
	}
	logFn("Starting process for IP: " + params.Ip)

	err = CheckMacAddress(params, logFn)
	if err != nil {
		return err
	}

	ip := params.Ip
	client, password, err := InitSshClient(ip, params.PromptPassword)
	if err != nil {
		return err
	}
	params.CurrentPassword = password
	logFn("Connection to device established")

	err = CheckSerialNumber(client, logFn)
	if err != nil {
		return err
	}

	progressFn(0.05)

	fwUpdateRequired, err := CheckFirmware(client, logFn, params.NewestFirmware)
	if err != nil {
		return err
	}
	if fwUpdateRequired {
		logFn("Firmware update required. Starting update...")
		client, err = UpdateFirmware(client, logFn, &params, progressFn)
		if err != nil {
			return err
		}
	}
	progressFn(0.6)

	newPassword, ok := params.PromptNewPassword()
	if !ok {
		return errors.New("new password prompt cancelled by user")
	}

	err = ChangeUserPasswords(client, logFn, newPassword)
	if err != nil {
		return err
	}
	progressFn(0.65)

	err = ConfigureServices(client, logFn)
	if err != nil {
		return err
	}
	progressFn(0.7)

	err = CreateContainer(client, logFn, params)
	if err != nil {
		return err
	}
	progressFn(0.95)

	CopyPathToDevice(client, params.ConfigPath, "/root", logFn)

	logFn("Installation complete.")
	progressFn(1)

	client.Close()
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
