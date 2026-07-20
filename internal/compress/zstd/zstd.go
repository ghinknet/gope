package zstd

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// Compress compresses the source stream.
func Compress(src io.Reader, dst io.Writer, level int) (int64, error) {
	// Create zstd compressor
	counted := &countingWriter{writer: dst}
	encoder, err := zstd.NewWriter(counted, zstd.WithEncoderLevel(mapLevel(level)))
	if err != nil {
		return 0, err
	}

	// Compress
	_, copyErr := io.Copy(encoder, src)
	closeErr := encoder.Close()
	return counted.written, errors.Join(copyErr, closeErr)
}

type countingWriter struct {
	writer  io.Writer
	written int64
}

func (w *countingWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.written += int64(n)
	return n, err
}

// mapLevel converts a 1-10 level into a zstd encoder level.
func mapLevel(level int) zstd.EncoderLevel {
	if level < 1 {
		level = 1
	}
	if level > 10 {
		level = 10
	}
	return zstd.EncoderLevelFromZstd(level)
}

// BatchDecompress expands a zstd-compressed tar archive into dstDir.
func BatchDecompress(archiveFile, dstDir string) error {
	// Open archive file
	f, err := os.Open(archiveFile)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

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
		if err = rejectSymlinkPath(dstDir, targetPath); err != nil {
			return err
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
			_, copyErr := io.Copy(outFile, tarReader)
			closeErr := outFile.Close()
			if err = errors.Join(copyErr, closeErr); err != nil {
				_ = os.Remove(targetPath)
				return err
			}
			// Set file permissions
			err = os.Chmod(targetPath, os.FileMode(header.Mode))
			if err != nil {
				_ = os.Remove(targetPath)
				return err
			}
		case tar.TypeSymlink:
			return fmt.Errorf("symbolic links are not supported in archive: %s", header.Name)
		default:
			return fmt.Errorf("unsupported archive entry type %q in %s", header.Typeflag, header.Name)
		}
	}
	return nil
}

func rejectSymlinkPath(baseDir, targetPath string) error {
	rel, err := filepath.Rel(baseDir, targetPath)
	if err != nil {
		return err
	}
	current := filepath.Clean(baseDir)
	paths := append([]string{current}, strings.Split(rel, string(filepath.Separator))...)
	for index, part := range paths {
		if index > 0 {
			current = filepath.Join(current, part)
		}
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			return nil
		}
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic link in extraction path: %s", current)
		}
	}
	return nil
}
