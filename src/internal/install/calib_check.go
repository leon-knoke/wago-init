package install

import (
	"errors"
	"strings"

	"golang.org/x/crypto/ssh"
)

const (
	calibCommand       = "cat /etc/calib"
	removeCalibCommand = "rm /etc/calib"
	initCalibCommand   = "/etc/init.d/calib start"
)

func ValidateCalibrationData(client *ssh.Client) error {
	if checkCalibrationData(client) == nil {
		return nil
	}
	if err := reinitCalibrationData(client); err != nil {
		return err
	}
	return checkCalibrationData(client)
}

func checkCalibrationData(client *ssh.Client) error {
	output, err := runSSHCommand(client, calibCommand, shortSessionTimeout)
	if err != nil {
		return err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 5 {
		return errors.New("Device is missing calibration data. Please return this device to retailer")
	}

	return nil
}

func reinitCalibrationData(client *ssh.Client) error {
	_, err := runSSHCommand(client, removeCalibCommand, shortSessionTimeout)
	if err != nil {
		return err
	}
	_, err = runSSHCommand(client, initCalibCommand, shortSessionTimeout)
	if err != nil {
		return err
	}
	return nil
}
