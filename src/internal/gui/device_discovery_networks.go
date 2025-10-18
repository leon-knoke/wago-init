package gui

import (
	"net"
	"sort"
	"strings"
)

func discoverHostNetworks() []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	ignoreNames := []string{
		"docker", "br-", "vmnet", "vbox", "virtual", "tunnel", "vpn", "wg", "tailscale", "zerotier", "zt", "utun", "ham", "hyper-v", "loopback",
	}

	seen := make(map[string]struct{})

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagPointToPoint != 0 {
			continue
		}

		lowerName := strings.ToLower(iface.Name)
		skip := false
		for _, needle := range ignoreNames {
			if strings.Contains(lowerName, needle) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ipv4 := ipNet.IP.To4()
			if ipv4 == nil {
				continue
			}
			if ipv4.IsLoopback() || ipv4.IsLinkLocalUnicast() || ipv4.IsLinkLocalMulticast() {
				continue
			}

			ones, bits := ipNet.Mask.Size()
			if bits != 32 {
				continue
			}
			if ones <= 0 || ones > 30 {
				continue
			}

			networkIP := ipv4.Mask(ipNet.Mask)
			network := (&net.IPNet{IP: networkIP, Mask: ipNet.Mask}).String()
			if _, exists := seen[network]; exists {
				continue
			}
			seen[network] = struct{}{}
		}
	}

	if len(seen) == 0 {
		return nil
	}

	networks := make([]string, 0, len(seen))
	for network := range seen {
		networks = append(networks, network)
	}
	sort.Strings(networks)
	return networks
}
