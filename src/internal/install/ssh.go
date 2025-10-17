package install

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
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

func InitSshClient(ip string, promptPassword func() (string, bool)) (*ssh.Client, string, error) {
	addr := net.JoinHostPort(ip, "22")
	password := DefaultSSHPassword

	for {
		client, err := dialSSH(addr, password)
		if err == nil {
			return client, password, nil
		}

		if !isAuthError(err) {
			return nil, password, fmt.Errorf("ssh dial failed: %w", err)
		}

		if promptPassword == nil {
			return nil, password, fmt.Errorf("ssh authentication failed: %w", err)
		}

		pwd, ok := promptPassword()
		if !ok {
			return nil, password, fmt.Errorf("ssh authentication cancelled by user")
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

func runSSHCommandStreaming(client *ssh.Client, cmd string, timeout time.Duration, logFn func(string)) error {
	if client == nil {
		return fmt.Errorf("ssh client is nil")
	}
	if logFn == nil {
		logFn = func(string) {}
	}

	sess, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer sess.Close()

	stdout, err := sess.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := sess.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	var wg sync.WaitGroup
	stderrBuf := &strings.Builder{}

	streamPipe := func(reader io.Reader, streamName, prefix string, collect *strings.Builder) {
		defer wg.Done()
		scanner := bufio.NewScanner(reader)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			text := scanner.Text()
			segments := splitStreamLine(text)
			for _, segment := range segments {
				line := strings.TrimSpace(segment)
				if line == "" {
					continue
				}
				if collect != nil {
					if collect.Len() > 0 {
						collect.WriteByte('\n')
					}
					collect.WriteString(line)
				}
				if prefix != "" {
					logFn(prefix + line)
				} else {
					logFn(line)
				}
			}
		}
		if err := scanner.Err(); err != nil {
			logFn(fmt.Sprintf("stream error (%s): %v", streamName, err))
		}
	}

	wg.Add(2)
	go streamPipe(stdout, "stdout", "", nil)
	go streamPipe(stderr, "stderr", "", stderrBuf)

	if err := sess.Start(cmd); err != nil {
		return fmt.Errorf("start command '%s': %w", cmd, err)
	}

	done := make(chan error, 1)
	go func() {
		done <- sess.Wait()
	}()

	var runErr error
	select {
	case runErr = <-done:
	case <-time.After(timeout):
		runErr = fmt.Errorf("command '%s' timed out after %s", cmd, timeout)
		_ = sess.Signal(ssh.SIGKILL)
		_ = sess.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}

	wg.Wait()

	if runErr != nil {
		if stderrBuf.Len() > 0 {
			return fmt.Errorf("%w (stderr: %s)", runErr, stderrBuf.String())
		}
		return runErr
	}
	return nil
}

func splitStreamLine(text string) []string {
	replaced := strings.ReplaceAll(text, "\r", "\n")
	return strings.Split(replaced, "\n")
}

func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}
	replaced := strings.ReplaceAll(arg, "'", "'\"'\"'")
	return "'" + replaced + "'"
}
