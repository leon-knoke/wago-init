//go:build !windows

package install

import "os/exec"

func hideConsoleWindow(cmd *exec.Cmd) {}
