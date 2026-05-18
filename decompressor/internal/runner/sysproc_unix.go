//go:build !windows

package runner

import (
	"os"
	"os/exec"
	"syscall"
)

// Signals forwarded to the child process group on Unix.
var forwardSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT}

// applySysProcAttr configures a new process group for signal fan-out.
func applySysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// forwardSignal sends the signal to the child process group.
func forwardSignal(cmd *exec.Cmd, sig os.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	if s, ok := sig.(syscall.Signal); ok {
		return syscall.Kill(-cmd.Process.Pid, s)
	}
	return cmd.Process.Signal(sig)
}

// forceKill terminates the child process group.
func forceKill(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
