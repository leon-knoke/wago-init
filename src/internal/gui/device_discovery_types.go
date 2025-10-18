package gui

const (
	deviceScanLimit       = 4096
	deviceScanConcurrency = 100
	deviceDiscoveryTitle  = "Device discovery"
)

type discoveredDevice struct {
	IP  string
	MAC string
}
