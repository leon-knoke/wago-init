package install

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	DefaultSSHUser      = "root"
	DefaultSSHPassword  = "wago"
	sshTimeout          = 60 * time.Second
	shortSessionTimeout = 10 * time.Second
	longSessionTimeout  = 60 * time.Second
)

func InitSshClient(ip string, promptPassword func() (string, bool)) (*ssh.Client, error) {
	addr := net.JoinHostPort(ip, "22")
	password := DefaultSSHPassword

	for {
		client, err := dialSSH(addr, password)
		if err == nil {
			return client, nil
		}

		if !isAuthError(err) {
			return nil, fmt.Errorf("ssh dial failed: %w", err)
		}

		if promptPassword == nil {
			return nil, fmt.Errorf("ssh authentication failed: %w", err)
		}

		pwd, ok := promptPassword()
		if !ok {
			return nil, fmt.Errorf("ssh authentication cancelled by user")
		}
		password = pwd
	}
}

func dialSSH(addr, password string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            DefaultSSHUser,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sshTimeout,
	}

	return ssh.Dial("tcp", addr, config)
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unable to authenticate") || strings.Contains(msg, "permission denied")
}

func runSSHCommand(client *ssh.Client, cmd string, timeout time.Duration) (string, error) {
	sess, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	defer sess.Close()

	var outBuf, errBuf bytes.Buffer
	sess.Stdout = &outBuf
	sess.Stderr = &errBuf

	done := make(chan error, 1)
	go func() { done <- sess.Run(cmd) }()

	select {
	case runErr := <-done:
		stdout := strings.TrimSpace(outBuf.String())
		stderr := strings.TrimSpace(errBuf.String())
		if runErr != nil {
			return stderr, fmt.Errorf("command '%s' failed: %w (stderr: %s)", cmd, runErr, stderr)
		}
		return stdout, nil
	case <-time.After(timeout):
		_ = sess.Signal(ssh.SIGKILL)
		return "", fmt.Errorf("command '%s' timed out", cmd)
	}
}
