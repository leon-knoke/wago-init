package install

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
	"golang.org/x/crypto/ssh"
)

var (
	SerialCommand_   = "/etc/config-tools/get_typelabel_value -n UII"
	FirmwareCommand_ = "/etc/config-tools/get_coupler_details firmware-revision"
)

func ChangeUserPasswords(client *ssh.Client, logFn func(string), newPassword string) error {

	hash, err := hashPasswordSHA512(newPassword)
	if err != nil {
		return err
	}
	for _, user := range usersList {
		_, err := runSSHCommand(client, "usermod -p '"+hash+"' "+user, shortSessionTimeout)
		if err != nil {
			return err
		}
	}

	logFn("Successfully changed user passwords")
	return nil
}

func hashPasswordSHA512(password string) (string, error) {
	seed := make([]byte, 12)
	if _, err := rand.Read(seed); err != nil {
		return "", err
	}
	salt := "$6$" + base64.RawStdEncoding.EncodeToString(seed)

	c := sha512_crypt.New()
	hash, err := c.Generate([]byte(password), []byte(salt))
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
