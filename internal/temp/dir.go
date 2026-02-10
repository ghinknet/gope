package temp

import (
	"io"
	"os"
)

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

// MvFile moves file from temp src to dst
func MvFile(src string, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func(srcFile *os.File) {
		if e := srcFile.Close(); e != nil {
			panic(e)
		}
	}(srcFile)

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func(dstFile *os.File) {
		if e := dstFile.Close(); e != nil {
			panic(e)
		}
	}(dstFile)

	// Copy content
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Sync to disk
	err = dstFile.Sync()
	if err != nil {
		return err
	}

	// Delete source file
	return os.Remove(src)
}
