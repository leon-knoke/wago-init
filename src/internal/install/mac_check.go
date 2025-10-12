package install

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

var allowedOUIs = []string{
	"00:30:de",
	"0:30:de",
}

func CheckMacAddress(installParameters Parameters, logFn func(string)) error {
	ip := installParameters.Ip

	if err := pingOnce(ip, logFn); err != nil {
		logFn("Ping attempt failed, device might be offline: " + err.Error())
	}
	mac, err := lookupMAC(ip)
	if err != nil {
		return fmt.Errorf("failed to resolve MAC for %s: %w", ip, err)
	}
	logFn("Device MAC address: " + mac)

	oui := mac
	if idx := strings.Index(mac, ":"); idx != -1 {
		parts := strings.Split(mac, ":")
		if len(parts) >= 3 {
			oui = strings.ToLower(strings.Join(parts[:3], ":"))
		}
	}
	for _, allowed := range allowedOUIs {
		if oui == allowed {
			return nil
		}
	}
	return errors.New("this device is not supported")
}

func pingOnce(ip string, logFn func(string)) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("ping", "-n", "1", "-w", "1000", ip)
	default: // linux, darwin, others (POSIX-like)
		cmd = exec.Command("ping", "-c", "1", "-W", "1", ip)
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func lookupMAC(ip string) (string, error) {
	var out []byte
	var err error
	switch runtime.GOOS {
	case "windows":
		out, err = exec.Command("arp", "-a", ip).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("arp failed: %s (output: %s)", err, strings.TrimSpace(string(out)))
		}
	default:
		out, err = exec.Command("arp", "-n", ip).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("ip neigh/arp failed: %s (output: %s)", err, strings.TrimSpace(string(out)))
		}
	}

	mac, err := extractMAC(string(out))
	if err != nil {
		return "", err
	}
	return mac, nil
}

func extractMAC(s string) (string, error) {
	re := regexp.MustCompile(`(?i)(?:[0-9a-f]{1,2}:){5}[0-9a-f]{1,2}`)
	m := re.FindString(s)
	if m == "" {
		return "", fmt.Errorf("no mac address found")
	}
	if m[1] == '-' || m[1] == ':' {
		m = "0" + m
	}
	return m, nil
}
