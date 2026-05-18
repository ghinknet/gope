//go:build !windows

package runner

import (
	"os"
	"os/exec"
	"syscall"
)

var forwardSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT}

func applySysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func forwardSignal(cmd *exec.Cmd, sig os.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	if s, ok := sig.(syscall.Signal); ok {
		return syscall.Kill(-cmd.Process.Pid, s)
	}
	return cmd.Process.Signal(sig)
}

func forceKill(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

