//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

func platformExitCode(exitErr *exec.ExitError) int {
	if status, ok := exitErr.ProcessState.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return exitErr.ExitCode()
}
