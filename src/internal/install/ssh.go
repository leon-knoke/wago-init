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

func InitSshClient(ip string) (*ssh.Client, error) {
	addr := net.JoinHostPort(ip, "22")

	config := &ssh.ClientConfig{
		User:            DefaultSSHUser,
		Auth:            []ssh.AuthMethod{ssh.Password(DefaultSSHPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sshTimeout,
	}

	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("ssh dial failed: %w", err)
	}
	return conn, nil
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
