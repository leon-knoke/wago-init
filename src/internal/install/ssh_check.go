package install

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	SerialCommand   = "/etc/config-tools/get_typelabel_value -n UII"
	FirmwareCommand = "/etc/config-tools/get_coupler_details firmware-revision"
)

func CheckSSH(installParameters Parameters, logFn func(string)) error {
	ip := installParameters.Ip
	conn, err := InitSshClient(ip)
	if err != nil {
		return err
	}
	defer conn.Close()
	logFn("Connection to device established")

	serialOut, err := runSSHCommand(conn, SerialCommand, shortSessionTimeout)
	if err != nil {
		return err
	}

	serial := parseSerial(serialOut)
	if serial == "" {
		return errors.New("serial output empty after parsing")
	}
	logFn("Device serial number: " + serial)

	fwOut, err := runSSHCommand(conn, FirmwareCommand, shortSessionTimeout)
	if err != nil {
		return err
	}
	fwFull, fwBuild := parseFirmwareBuild(fwOut)
	if fwFull == "" {
		return errors.New("firmware output empty")
	}
	if fwBuild != "" {
		logFn(fmt.Sprintf("Firmware revision: %s", fwBuild))
	} else {
		logFn("Firmware revision: " + fwFull + " (build number not detected)")
	}

	return nil
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
