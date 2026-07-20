//go:build windows

package runner

import (
	"os"
	"os/exec"
	"syscall"
)

// Signals forwarded to the child process on Windows.
var forwardSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

// applySysProcAttr keeps the child attached to the current console.
func applySysProcAttr(cmd *exec.Cmd) {
}

// forwardSignal forwards a signal to the child.
func forwardSignal(cmd *exec.Cmd, sig os.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Signal(sig)
}

// forceKill terminates the child process.
func forceKill(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
