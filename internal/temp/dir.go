package temp

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Dir is the temporary directory object
// Temp directory utilities used across pack/build flows.
type Dir struct {
	path string
}

// Release deletes the temporary directory
func (d *Dir) Release() error {
	return os.RemoveAll(d.path)
}

// MkTemp creates a temporary directory
func MkTemp() (Dir, error) {
	path, err := os.MkdirTemp("", "gope-")
	if err != nil {
		return Dir{}, err
	}
	return Dir{path: path}, nil
}

// Path returns the temporary directory path
func (d *Dir) Path() string {
	return d.path
}

// MvFile moves file from temp src to dst
func MvFile(src string, dst string) error {
	// Prefer an atomic rename when source and destination share a filesystem.
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}

	stagedFile, err := os.CreateTemp(filepath.Dir(dst), ".gope-output-*")
	if err != nil {
		_ = srcFile.Close()
		return err
	}
	stagedPath := stagedFile.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(stagedPath)
		}
	}()

	_, copyErr := io.Copy(stagedFile, srcFile)
	syncErr := stagedFile.Sync()
	stagedCloseErr := stagedFile.Close()
	srcCloseErr := srcFile.Close()
	if err = errors.Join(copyErr, syncErr, stagedCloseErr, srcCloseErr); err != nil {
		return err
	}

	if err = os.Rename(stagedPath, dst); err != nil {
		return err
	}
	committed = true

	return os.Remove(src)
}
