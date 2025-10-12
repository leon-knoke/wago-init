package install

import "golang.org/x/crypto/ssh"

var (
	NtpCommand    = "/etc/config-tools/config_sntp state=enabled time-server-1=pool.ntp.org update-time=600"
	DockerCommand = "/etc/config-tools/config_docker activate"
)

func ConfigureServices(client *ssh.Client, logFn func(string)) error {
	ntpOut, err := runSSHCommand(client, NtpCommand, shortSessionTimeout)
	if err != nil {
		return err
	}
	logFn("NTP set to pool.ntp.org " + ntpOut)

	dockerOut, err := runSSHCommand(client, DockerCommand, longSessionTimeout)
	if err != nil {
		return err
	}
	logFn("Docker Service activated " + dockerOut)

	return nil
}
