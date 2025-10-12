package install

const DefaultIp = "192.168.42.42"

type Parameters struct {
	Ip                string
	PromptPassword    func() (string, bool)
	PromptNewPassword func() (string, bool)
	AWSToken          string
}

var usersList = []string{"root", "admin", "user"}
