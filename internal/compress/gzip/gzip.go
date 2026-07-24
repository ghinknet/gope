package gzip

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
)

// Compress compresses the source stream with gzip using a 1-10 level scale.
func Compress(src io.Reader, dst io.Writer, level int) (int64, error) {
	mapped, err := mapLevel(level)
	if err != nil {
		return 0, err
	}
	counted := &countingWriter{writer: dst}
	writer, err := gzip.NewWriterLevel(counted, mapped)
	if err != nil {
		return 0, err
	}

	_, copyErr := io.Copy(writer, src)
	closeErr := writer.Close()
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

// mapLevel converts 1-10 into gzip's level range.
func mapLevel(level int) (int, error) {
	if level < 1 || level > 10 {
		return 0, fmt.Errorf("compression level must be 1-10")
	}
	if level >= 9 {
		return gzip.BestCompression, nil
	}
	return level, nil
}
