package runner

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// Run an executable with passthrough
func Run(envs []string, binaryPath string, args []string, workingDir string, quiet bool) (int, error) {
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workingDir
	cmd.Stdin = os.Stdin
	if !quiet {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.Env = append(os.Environ(), envs...)
	applySysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		return -1, err
	}

	stopForward := startSignalForwarding(cmd)
	defer stopForward()

	return exitCode(cmd.Wait())
}

func startSignalForwarding(cmd *exec.Cmd) func() {
	if len(forwardSignals) == 0 {
		return func() {}
	}
	// Buffer prevents missed signals during bursts.
	sigChan := make(chan os.Signal, 4)
	signal.Notify(sigChan, forwardSignals...)

	go func() {
		for sig := range sigChan {
			_ = forwardSignal(cmd, sig)
			if isTerminationSignal(sig) {
				time.AfterFunc(5*time.Second, func() {
					forceKill(cmd)
				})
			}
		}
	}()

	return func() {
		signal.Stop(sigChan)
		close(sigChan)
	}
}

func exitCode(err error) (int, error) {
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return -1, err
}

func isTerminationSignal(sig os.Signal) bool {
	switch sig {
	case os.Interrupt, syscall.SIGTERM, syscall.SIGINT:
		return true
	default:
		return false
	}
}
