package install

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

var (
	SerialCommand   = "/etc/config-tools/get_typelabel_value -n UII"
	FirmwareCommand = "/etc/config-tools/get_coupler_details firmware-revision"
)

func CheckSerialNumber(client *ssh.Client, logFn func(string)) error {

	serialOut, err := runSSHCommand(client, SerialCommand, shortSessionTimeout)
	if err != nil {
		return err
	}

	serial := parseSerial(serialOut)
	if serial == "" {
		return errors.New("serial output empty after parsing")
	}
	logFn("Device serial number: " + serial)
	return nil
}

func CheckFirmware(client *ssh.Client, logFn func(string), newestFirmware int) (bool, error) {

	fwUpdateRequired := false

	fwOut, err := runSSHCommand(client, FirmwareCommand, shortSessionTimeout)
	if err != nil {
		return fwUpdateRequired, err
	}
	fwFull, fwBuild := parseFirmwareBuild(fwOut)
	if fwFull == "" {
		return fwUpdateRequired, errors.New("firmware output empty")
	}
	if fwBuild != 0 {
		logFn(fmt.Sprintf("Firmware revision: %d", fwBuild))
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

func parseFirmwareBuild(raw string) (full string, build int) {
	fwBuildRegex := regexp.MustCompile(`\((\d+)\)`)
	full = strings.TrimSpace(raw)
	m := fwBuildRegex.FindStringSubmatch(full)
	if len(m) == 2 {
		num, err := strconv.Atoi(strings.TrimSpace(m[1]))
		if err == nil {
			build = num
		} else {
			build = 0
		}
	}
	return
}
