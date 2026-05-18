//go:build windows

package runner

import (
	"os"
	"os/exec"
	"syscall"
)

// Signals forwarded to the child process on Windows.
var forwardSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

// applySysProcAttr creates a new process group for the child.
func applySysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
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
