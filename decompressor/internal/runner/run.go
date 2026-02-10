package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// Run an executable with passthrough
func Run(binaryPath string, args []string, workingDir string) (int, error) {
	ctx := context.Background()

	// Create command
	cmd := exec.CommandContext(ctx, binaryPath, args...)

	// Set working directory
	cmd.Dir = workingDir

	// Set standard input output
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables (optional custom environment variables)
	cmd.Env = os.Environ()

	// Start the command
	if err := cmd.Start(); err != nil {
		return -1, err
	}

	// Make a channel to receive signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Create a goroutine to forward signals
	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				// Forward the signal to the child process
				if err := cmd.Process.Signal(sig); err != nil {
					panic(err)
				}

				// If it's SIGINT, also send it to yourself
				if sig == syscall.SIGINT {
					// Optionally: You can set a timeout
					// if the child process doesn't exit
					// then force kill it
					time.AfterFunc(3*time.Second, func() {
						if cmd.Process != nil {
							_ = cmd.Process.Kill()
						}
					})
				}
			}
		}
	}()

	// Waiting for the command to finish
	err := cmd.Wait()

	// Clean up the signal handler
	signal.Stop(sigChan)
	close(sigChan)

	// Handle the exit code
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Get the exit status
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), nil
			}
			// If the status cannot be obtained, return 1
			return 1, nil
		}
		// Non-exit error
		return -1, err
	}

	// Normal exit
	return 0, nil
}
