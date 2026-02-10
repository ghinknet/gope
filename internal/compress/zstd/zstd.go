package zstd

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// Compress compresses the source stream
func Compress(src io.Reader, dst io.Writer) (int64, error) {
	// Create zstd compressor
	encoder, err := zstd.NewWriter(dst)
	if err != nil {
		return 0, err
	}
	defer func(encoder *zstd.Encoder) {
		if e := encoder.Close(); err != nil {
			panic(e)
		}
	}(encoder)

	// Compress
	return io.Copy(encoder, src)
}

// BatchDecompress decompress a batch archive file
func BatchDecompress(archiveFile, dstDir string) error {
	// Open archive file
	f, err := os.Open(archiveFile)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		if e := f.Close(); e != nil {
			panic(e)
		}
	}(f)

	// Create zstd reader
	zstdReader, err := zstd.NewReader(f)
	if err != nil {
		return err
	}
	defer zstdReader.Close()

	// Create tar reader on the zstd stream
	tarReader := tar.NewReader(zstdReader)

	// Iterate over each file/directory in the tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // Archive end
		}
		if err != nil {
			return err
		}

		// Construct target path
		targetPath := filepath.Join(dstDir, header.Name)
		if !(func(path, dir string) bool {
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return false
			}
			return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
		}(targetPath, dstDir)) {
			return fmt.Errorf("invalid path: %s", targetPath)
		}

		// Process file type
		switch header.Typeflag {
		case tar.TypeDir: // Dir
			if err = os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg: // File
			// Make sure parent directory exists
			if err = os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			// Create and write file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			if _, err = io.Copy(outFile, tarReader); err != nil {
				err = outFile.Close()
				if err != nil {
					return err
				}
				return err
			}
			err = outFile.Close()
			if err != nil {
				return err
			}
			// Set file permissions
			err = os.Chmod(targetPath, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
		case tar.TypeSymlink: // Symbolic link
			if err = os.Symlink(header.Linkname, targetPath); err != nil {
				return err
			}
		default:
			log.Printf("Skipping unsupported type: %c in %s\n", header.Typeflag, header.Name)
		}
	}
	return nil
}
