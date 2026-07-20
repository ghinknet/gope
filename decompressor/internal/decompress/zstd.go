//go:build zstd || (!gzip && !zstd)

package decompress

import (
	"io"

	"github.com/klauspost/compress/zstd"
)

// Decompress decompresses the source zstd stream.
func Decompress(src io.Reader, dst io.Writer) (int64, error) {
	// Create zstd decompressor
	decoder, err := zstd.NewReader(src)
	if err != nil {
		return 0, err
	}
	defer decoder.Close()

	// Compress
	return io.Copy(dst, decoder)
}
