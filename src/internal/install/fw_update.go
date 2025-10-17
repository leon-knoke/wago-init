package install

import "golang.org/x/crypto/ssh"

func UpdateFirmware(client *ssh.Client, logFn func(string), params Parameters, progressFn func(float64)) error {
	logFn("Firmware update process started...")
	return nil
}
