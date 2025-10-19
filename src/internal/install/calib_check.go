package install

import (
	"errors"
	"strings"

	"golang.org/x/crypto/ssh"
)

const calibCommand = "cat /etc/calib"

func CheckCalibrationData(client *ssh.Client) error {
	// expected output:

	// PT1 PT2 AI1 AI2 AO1 AO2
	// 9610 1000 40598 3000
	// 9625 1000 40624 3000
	// 14144 2506 41875 7494
	// 14142 2506 41864 7494
	// 1059 350 9019 3000
	// 1059 350 9022 3000

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
