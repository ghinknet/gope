//go:build zstd || (!gzip && !zstd)

package main

import (
	"bytes"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func compressTestPayload(t *testing.T, data []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	encoder, err := zstd.NewWriter(&buffer)
	if err != nil {
		t.Fatalf("new zstd writer: %v", err)
	}
	if _, err = encoder.Write(data); err != nil {
		t.Fatalf("compress payload: %v", err)
	}
	if err = encoder.Close(); err != nil {
		t.Fatalf("close zstd writer: %v", err)
	}
	return buffer.Bytes()
}
