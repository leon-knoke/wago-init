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
	// normalized OUI (lowercase, two hex digits per octet)
	"00:30:de",
}

func CheckMacAddress(installParameters Parameters, logFn func(string, string)) error {
	ip := installParameters.Ip

	if err := pingOnce(ip); err != nil {
		logFn("Ping attempt failed, device might be offline: "+err.Error(), "")
	}

	mac, allowed, err := DiscoverDeviceMAC(ip)
	if err != nil {
		return fmt.Errorf("failed to resolve MAC for %s: %w", ip, err)
	}
	if !allowed {
		return errors.New("this device is not supported")
	}

	logFn("Device MAC address: "+mac, "")
	return nil
}

func DiscoverDeviceMAC(ip string) (string, bool, error) {
	pingErr := pingOnce(ip)

	mac, err := lookupMAC(ip)
	if err != nil {
		if pingErr != nil {
			return "", false, fmt.Errorf("ping failed: %w; lookup failed: %v", pingErr, err)
		}
		return "", false, err
	}

	return mac, isAllowedOUI(mac), nil
}

func isAllowedOUI(mac string) bool {
	parts := strings.Split(mac, ":")
	if len(parts) < 3 {
		return false
	}
	oui := strings.ToLower(strings.Join(parts[:3], ":"))
	for _, allowed := range allowedOUIs {
		if oui == allowed {
			return true
		}
	}
	return false
}

func pingOnce(ip string) error {
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
	case "linux":
		// prefer `ip neigh show <ip>` on modern linux systems instead of the deprecated `arp`
		out, err = exec.Command("ip", "neigh", "show", ip).CombinedOutput()
		if err != nil {
			out, err = exec.Command("arp", "-n", ip).CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("ip neigh/arp failed: %s (output: %s)", err, strings.TrimSpace(string(out)))
			}
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
	re := regexp.MustCompile(`(?i)(?:[0-9a-f]{1,2}(?:[:\-])){5}[0-9a-f]{1,2}`)
	m := re.FindString(s)
	if m == "" {
		return "", fmt.Errorf("no mac address found")
	}
	normalized, err := normalizeMAC(m)
	if err != nil {
		return "", err
	}
	return normalized, nil
}

func normalizeMAC(raw string) (string, error) {
	r := strings.ReplaceAll(raw, "-", ":")
	r = strings.ToLower(r)
	parts := strings.Split(r, ":")
	if len(parts) != 6 {
		return "", fmt.Errorf("unexpected mac format: %s", raw)
	}
	for i, p := range parts {
		if len(p) == 1 {
			parts[i] = "0" + p
		} else if len(p) == 2 {
			// ok
		} else {
			return "", fmt.Errorf("unexpected mac octet: %s", p)
		}
	}
	return strings.Join(parts, ":"), nil
}
