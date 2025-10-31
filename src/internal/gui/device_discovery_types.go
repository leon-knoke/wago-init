package gui

import (
	"net"
	"sort"
)

const (
	deviceScanLimit       = 4096
	deviceScanConcurrency = 128
	deviceDiscoveryTitle  = "Device discovery"
)

type discoveredDevice struct {
	IP  string
	MAC string
}

// sortDiscoveredDevices orders devices by their IPv4 address (ascending).
func sortDiscoveredDevices(devices *[]discoveredDevice) {
	if devices == nil {
		return
	}

	sort.Slice(*devices, func(i, j int) bool {
		left := (*devices)[i]
		right := (*devices)[j]

		leftIP := net.ParseIP(left.IP)
		rightIP := net.ParseIP(right.IP)

		leftVal, leftErr := ipToUint32(leftIP)
		rightVal, rightErr := ipToUint32(rightIP)

		if leftErr == nil && rightErr == nil {
			if leftVal == rightVal {
				return left.MAC < right.MAC
			}
			return leftVal < rightVal
		}

		if leftErr != nil && rightErr == nil {
			return false
		}
		if leftErr == nil && rightErr != nil {
			return true
		}

		return left.IP < right.IP
	})
}
