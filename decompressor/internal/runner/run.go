package runner

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Run starts a child process with IO and signal passthrough.
func Run(binaryPath string, args []string, workingDir string) (int, error) {
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workingDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	applySysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		return -1, err
	}

	stopForward := startSignalForwarding(cmd)
	waitErr := cmd.Wait()
	stopForward()

	return exitCode(waitErr)
}

// startSignalForwarding forwards OS signals to the child.
func startSignalForwarding(cmd *exec.Cmd) func() {
	if len(forwardSignals) == 0 {
		return func() {}
	}
	// Buffer prevents missed signals during bursts.
	sigChan := make(chan os.Signal, 4)
	signal.Notify(sigChan, forwardSignals...)
	done := make(chan struct{})
	var stopOnce sync.Once
	var timerMu sync.Mutex
	var killTimer *time.Timer

	go func() {
		for {
			select {
			case sig := <-sigChan:
				_ = forwardSignal(cmd, sig)
				if isTerminationSignal(sig) {
					timerMu.Lock()
					if killTimer != nil {
						killTimer.Stop()
					}
					killTimer = time.AfterFunc(5*time.Second, func() {
						forceKill(cmd)
					})
					timerMu.Unlock()
				}
			case <-done:
				return
			}
		}
	}()

	return func() {
		stopOnce.Do(func() {
			signal.Stop(sigChan)
			close(done)
			timerMu.Lock()
			if killTimer != nil {
				killTimer.Stop()
			}
			timerMu.Unlock()
		})
	}
}

// exitCode normalizes process exit status to an int.
func exitCode(err error) (int, error) {
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return platformExitCode(exitErr), nil
	}
	return -1, err
}

// isTerminationSignal returns true for signals that should trigger a forced kill.
func isTerminationSignal(sig os.Signal) bool {
	switch sig {
	case os.Interrupt, syscall.SIGTERM, syscall.SIGINT:
		return true
	default:
		return false
	}
}
