package temp

import "os"

// Temporary directory helpers for the decompressor runtime.
// Dir is the temporary directory object
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
