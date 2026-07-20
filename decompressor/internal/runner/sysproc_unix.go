//go:build !windows

package runner

import (
	"os"
	"os/exec"
	"syscall"
)

// Signals forwarded to the child process group on Unix.
var forwardSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT}

// applySysProcAttr keeps the child in the foreground process group so
// interactive terminal reads continue to work.
func applySysProcAttr(cmd *exec.Cmd) {
}

// forwardSignal sends the signal to the direct child.
func forwardSignal(cmd *exec.Cmd, sig os.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Signal(sig)
}

// forceKill terminates the direct child.
func forceKill(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
