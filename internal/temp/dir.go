package temp

import (
	"io"
	"os"
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
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func(srcFile *os.File) {
		if e := srcFile.Close(); e != nil {
			panic(e)
		}
	}(srcFile)

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func(dstFile *os.File) {
		if e := dstFile.Close(); e != nil {
			panic(e)
		}
	}(dstFile)

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	if err = dstFile.Sync(); err != nil {
		return err
	}

	return os.Remove(src)
}
