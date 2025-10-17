package install

const DefaultIp = "192.168.42.42"

type Parameters struct {
	Ip                string
	FirmwareRevision  string
	NewestFirmware    int
	FirmwarePath      string
	CurrentPassword   string
	PromptPassword    func() (string, bool)
	PromptNewPassword func() (string, bool)
	AWSToken          string
	AWSEcrUrl         string
	ContainerImage    string
	ContainerFlags    string
	ConfigPath        string
}

var usersList = []string{"root", "admin", "user"}
