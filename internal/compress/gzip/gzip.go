package gzip

import (
	"compress/gzip"
	"fmt"
	"io"
)

// Compress compresses the source stream with gzip using a 1-10 level scale.
func Compress(src io.Reader, dst io.Writer, level int) (int64, error) {
	mapped, err := mapLevel(level)
	if err != nil {
		return 0, err
	}
	writer, err := gzip.NewWriterLevel(dst, mapped)
	if err != nil {
		return 0, err
	}
	defer func(writer *gzip.Writer) {
		if e := writer.Close(); e != nil {
			panic(e)
		}
	}(writer)

	return io.Copy(writer, src)
}

func mapLevel(level int) (int, error) {
	if level < 1 || level > 10 {
		return 0, fmt.Errorf("compression level must be 1-10")
	}
	if level >= 9 {
		return gzip.BestCompression, nil
	}
	return level, nil
}

