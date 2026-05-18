//go:build gzip

package decompress

import (
	"compress/gzip"
	"io"
)

// Decompress decompresses the source gzip stream.
func Decompress(src io.Reader, dst io.Writer) (int64, error) {
	reader, err := gzip.NewReader(src)
	if err != nil {
		return 0, err
	}
	defer func(reader *gzip.Reader) {
		_ = reader.Close()
	}(reader)

	return io.Copy(dst, reader)
}
