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
	progressFn(0.04)

	ip := params.Ip
	client, err := InitSshClient(ip, params.PromptPassword)
	if err != nil {
		return err
	}

	err = CheckSSH(client, logFn)
	if err != nil {
		return err
	}
	progressFn(0.1)

	newPassword, ok := params.PromptNewPassword()
	if !ok {
		return errors.New("new password prompt cancelled by user")
	}

	err = ChangeUserPasswords(client, logFn, newPassword)
	if err != nil {
		return err
	}
	progressFn(0.15)

	err = ConfigureServices(client, logFn)
	if err != nil {
		return err
	}
	progressFn(0.2)

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
