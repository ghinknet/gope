//go:build windows

package runner

import "os/exec"

func platformExitCode(exitErr *exec.ExitError) int {
	return exitErr.ExitCode()
}
