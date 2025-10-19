package install

import "context"

const DefaultIp = "192.168.42.42"

type Parameters struct {
	Ip                string
	FirmwareRevision  string
	NewestFirmware    int
	FirmwarePath      string
	ForceFirmware     bool
	CurrentPassword   string
	PromptPassword    func() (string, bool)
	PromptNewPassword func() (string, bool)
	AWSToken          string
	AWSEcrUrl         string
	ContainerImage    string
	ContainerFlags    string
	ConfigPath        string
	Context           context.Context
}

var usersList = []string{"root", "admin", "user"}
