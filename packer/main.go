package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
)

func main() {
	if err := run(); err != nil {
		fmt.Println("error packing decompressor:", err)
	}
}

func run() error {
	outFile, err := os.Create("decompressorSourceCode")
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer func(outFile *os.File) {
		if e := outFile.Close(); e != nil {
			panic(e)
		}
	}(outFile)

	zstdWriter, err := zstd.NewWriter(outFile)
	if err != nil {
		return fmt.Errorf("error creating zstd compressor: %w", err)
	}
	defer func(zstdWriter *zstd.Encoder) {
		if e := zstdWriter.Close(); e != nil {
			panic(e)
		}
	}(zstdWriter)

	tarWriter := tar.NewWriter(zstdWriter)
	defer func(tarWriter *tar.Writer) {
		if e := tarWriter.Close(); e != nil {
			panic(e)
		}
	}(tarWriter)

	return filepath.Walk("decompressor", func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel("decompressor", filePath)
		if err != nil {
			return err
		}
		header.Name = relPath
		if relPath == "." {
			return nil
		}

		if err = tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		f, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer func(f *os.File) {
			if e := f.Close(); e != nil {
				panic(e)
			}
		}(f)

		_, err = io.Copy(tarWriter, f)
		return err
	})
}
