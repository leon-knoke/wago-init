package install

var (
	NtpCommand    = "/etc/config-tools/config_sntp state=enabled time-server-1=pool.ntp.org update-time=600"
	DockerCommand = "/etc/config-tools/config_docker activate"
)

func ConfigureServices(installParameters Parameters, logFn func(string)) error {
	ip := installParameters.Ip
	conn, err := InitSshClient(ip)
	if err != nil {
		return err
	}

	ntpOut, err := runSSHCommand(conn, NtpCommand, shortSessionTimeout)
	if err != nil {
		return err
	}
	logFn("NTP set to pool.ntp.org " + ntpOut)

	dockerOut, err := runSSHCommand(conn, DockerCommand, longSessionTimeout)
	if err != nil {
		return err
	}
	logFn("Docker Service activated " + dockerOut)

	return nil
}
