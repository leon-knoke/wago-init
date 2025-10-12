package install

const DefaultIp = "10.92.1.113"

type Parameters struct {
	Ip                string
	PromptPassword    func() (string, bool)
	PromptNewPassword func() (string, bool)
}

var usersList = []string{"root", "admin", "user"}
