package install

import (
	"fmt"
	"strings"
	"time"
	"wago-init/internal/fs"

	"golang.org/x/crypto/ssh"
)

const (
	containerCreateTimeout = 20 * time.Minute
)

func CreateContainer(client *ssh.Client, logFn func(string, string), params Parameters) error {

	ecrLogin(client, params.AWSToken, params.AWSEcrUrl)

	createCmd := buildDockerCreateCommand(params.ContainerFlags, params.ContainerImage)
	logFn("Creating container with image: "+params.ContainerImage, "")
	if err := runSSHCommandStreaming(client, createCmd, containerCreateTimeout, logFn); err != nil {
		return fmt.Errorf("docker create failed: %w", err)
	}

	logFn("Container created successfully.", "")
	return nil
}

func BuildContainerCommand(flagsRaw string) string {
	if flagsRaw == "" {
		return ""
	}

	raw := fs.DecodeMultilineValue(flagsRaw)
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	lines := strings.Split(raw, "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}

	return strings.Join(parts, " ")
}

func ecrLogin(client *ssh.Client, token, ecrUrl string) error {
	loginCmd := fmt.Sprintf("echo %s | docker login --username AWS --password-stdin %s",
		shellQuote(token), shellQuote(ecrUrl))
	if _, err := runSSHCommand(client, loginCmd, shortSessionTimeout); err != nil {
		return fmt.Errorf("ecr login failed: %w", err)
	}
	return nil
}

func buildDockerCreateCommand(flags, image string) string {
	parts := []string{"docker", "create"}
	trimmedFlags := strings.TrimSpace(flags)
	if trimmedFlags != "" {
		parts = append(parts, trimmedFlags)
	}
	parts = append(parts, shellQuote(image))
	return strings.Join(parts, " ")
}
