package install

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
)

var (
	SerialCommand   = "/etc/config-tools/get_typelabel_value -n UII"
	FirmwareCommand = "/etc/config-tools/get_coupler_details firmware-revision"
)

func CheckSSH(client *ssh.Client, logFn func(string), newestFirmware string) (bool, error) {
	logFn("Connection to device established")

	fwUpdateRequired := false

	serialOut, err := runSSHCommand(client, SerialCommand, shortSessionTimeout)
	if err != nil {
		return fwUpdateRequired, err
	}

	serial := parseSerial(serialOut)
	if serial == "" {
		return fwUpdateRequired, errors.New("serial output empty after parsing")
	}
	logFn("Device serial number: " + serial)

	fwOut, err := runSSHCommand(client, FirmwareCommand, shortSessionTimeout)
	if err != nil {
		return fwUpdateRequired, err
	}
	fwFull, fwBuild := parseFirmwareBuild(fwOut)
	if fwFull == "" {
		return fwUpdateRequired, errors.New("firmware output empty")
	}
	if fwBuild != "" {
		logFn(fmt.Sprintf("Firmware revision: %s", fwBuild))
		if fwBuild < newestFirmware {
			fwUpdateRequired = true
		}
	} else {
		logFn("Firmware revision: " + fwFull + " (build number not detected)")
	}

	return fwUpdateRequired, nil
}

func parseSerial(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "UII=") {
		return strings.TrimSpace(raw[4:])
	}
	return raw
}

func parseFirmwareBuild(raw string) (full string, build string) {
	fwBuildRegex := regexp.MustCompile(`\((\d+)\)`)
	full = strings.TrimSpace(raw)
	m := fwBuildRegex.FindStringSubmatch(full)
	if len(m) == 2 {
		build = m[1]
	}
	return
}
