package install

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const copyToDeviceTimeout = 20 * time.Minute

// CopyPathToDevice replicates the contents of localPath onto remotePath using an existing SSH client.
// localPath can point to either a single file or a directory. Directories are copied recursively.
// Collected output is streamed through logFn so the user can monitor progress.
func CopyPathToDevice(client *ssh.Client, localPath, remotePath string, logFn func(string, string)) error {
	localPath = filepath.Clean(localPath)
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return errors.New("remote path must not be empty")
	}

	info, err := os.Lstat(localPath)
	if err != nil {
		return fmt.Errorf("stat local path: %w", err)
	}

	logFn(fmt.Sprintf("Copying %s to %s", localPath, remotePath), "")

	if _, err := runSSHCommand(client, fmt.Sprintf("mkdir -p %s", shellQuote(remotePath)), shortSessionTimeout); err != nil {
		return fmt.Errorf("ensure remote directory: %w", err)
	}

	sess, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer sess.Close()

	pipeReader, pipeWriter := io.Pipe()
	sess.Stdin = pipeReader

	cmd := fmt.Sprintf("tar -xpf - -C %s", shellQuote(remotePath))
	if err := sess.Start(cmd); err != nil {
		pipeReader.Close()
		pipeWriter.Close()
		return fmt.Errorf("start remote extract: %w", err)
	}

	streamErrCh := make(chan error, 1)
	go func() {
		err := streamLocalPathToTar(pipeWriter, localPath, info, logFn)
		if err != nil {
			pipeWriter.CloseWithError(err)
		} else {
			pipeWriter.Close()
		}
		streamErrCh <- err
	}()

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- sess.Wait()
	}()

	timer := time.NewTimer(copyToDeviceTimeout)
	defer timer.Stop()

	var streamErr error
	streamDone := false
	var sessionErr error
	sessionDone := false

	for !streamDone || !sessionDone {
		select {
		case err := <-streamErrCh:
			streamErr = err
			streamDone = true
		case err := <-waitCh:
			sessionErr = err
			sessionDone = true
		case <-timer.C:
			logFn("Copy operation timed out; attempting to abort remote extraction", "")
			if !sessionDone {
				sessionErr = fmt.Errorf("copy command timed out after %s", copyToDeviceTimeout)
				_ = sess.Signal(ssh.SIGKILL)
				_ = sess.Close()
			}
		}
	}

	if streamErr != nil {
		return fmt.Errorf("package local content: %w", streamErr)
	}
	if sessionErr != nil {
		return fmt.Errorf("remote extraction: %w", sessionErr)
	}

	logFn("Copy complete.", "")
	return nil
}

func streamLocalPathToTar(w io.Writer, basePath string, info os.FileInfo, logFn func(string, string)) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	if info.IsDir() {
		return filepath.Walk(basePath, func(path string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(basePath, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if rel == "." {
				return nil
			}
			return writeTarEntry(tw, path, rel, fileInfo, logFn)
		})
	}

	return writeTarEntry(tw, basePath, filepath.Base(basePath), info, logFn)
}

func writeTarEntry(tw *tar.Writer, fullPath, rel string, info os.FileInfo, logFn func(string, string)) error {
	mode := info.Mode()
	linkTarget := ""
	if mode&os.ModeSymlink != 0 {
		target, err := os.Readlink(fullPath)
		if err != nil {
			return fmt.Errorf("read symlink '%s': %w", fullPath, err)
		}
		linkTarget = target
	}

	header, err := tar.FileInfoHeader(info, linkTarget)
	if err != nil {
		return fmt.Errorf("tar header for '%s': %w", fullPath, err)
	}
	header.Name = strings.TrimPrefix(filepath.ToSlash(rel), "./")
	header.Format = tar.FormatPAX

	if info.IsDir() && !strings.HasSuffix(header.Name, "/") {
		header.Name += "/"
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("write header for '%s': %w", fullPath, err)
	}

	if mode.IsRegular() {
		file, err := os.Open(fullPath)
		if err != nil {
			return fmt.Errorf("open '%s': %w", fullPath, err)
		}
		defer file.Close()
		if _, err := io.Copy(tw, file); err != nil {
			return fmt.Errorf("copy '%s' contents: %w", fullPath, err)
		}
		logFn("Copied file: "+header.Name, "")
	} else if mode&os.ModeSymlink != 0 {
		logFn("Copied symlink: "+header.Name, "")
	} else if info.IsDir() {
		logFn("Created directory: "+header.Name, "")
	}

	return nil
}
