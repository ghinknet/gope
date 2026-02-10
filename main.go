package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/ghinknet/gope/internal/compress/zstd"
	"github.com/ghinknet/gope/internal/constant"
	"github.com/ghinknet/gope/internal/runner"
	"github.com/ghinknet/gope/internal/temp"
	"github.com/ghinknet/toolbox/expr"
	"github.com/spf13/cobra"
)

//go:embed decompressorSourceCode
var embeddedDecompressor []byte

// ReleaseEmbedded releases the embedded compressed executable
func releaseEmbedded(temp string) error {
	return os.WriteFile(
		filepath.Join(temp, "decompressorSourceCode"), embeddedDecompressor, 0644,
	)
}

var rootCmd = &cobra.Command{
	Use: constant.Name,
	Long: `Go Packer for Executable
A go-based cross platform binary compressor.

"Well. Your physical education teacher is not sick :D"`,
	Version: constant.VersionText,
	RunE:    Runner,
}

var input string
var output string
var method string
var system string
var arch string
var upx bool
var quiet bool

var goPath string
var upxPath string
var winePath string

func init() {
	rootCmd.Flags().StringVarP(&input, "input", "i", "", "Input file")
	rootCmd.Flags().StringVarP(&output, "output", "o", "output", "Output file")
	rootCmd.Flags().StringVarP(&method, "method", "m", "zstd", "Compress method")
	rootCmd.Flags().StringVarP(&system, "system", "s", runtime.GOOS, "Destination system")
	rootCmd.Flags().StringVarP(&arch, "arch", "a", runtime.GOARCH, "Destination arch")
	rootCmd.Flags().BoolVarP(&upx, "upx", "u", false, "Use UPX to compress again")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Quiet mode")

	rootCmd.Flags().StringVarP(&goPath, "go-path", "g", "go", "Go path")
	rootCmd.Flags().StringVarP(&upxPath, "upx-path", "p", "upx", "UPX path")
	rootCmd.Flags().StringVarP(&winePath, "wine-path", "w", "wine", "Wine path")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

func Runner(cmd *cobra.Command, args []string) error {
	// Check flags
	if input == "" {
		return fmt.Errorf("input file is required")
	}
	if !slices.Contains(constant.SupportedMethods, method) {
		return fmt.Errorf("method %s is not supported", method)
	}
	if !slices.Contains(constant.SupportedPlatforms, system+"/"+arch) {
		return fmt.Errorf("unsupported platform: %s/%s", system, arch)
	}

	// Get work dir
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting work dir: %v", err)
	}

	// Read input file
	inFile, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("error opening input file: %v", err)
	}
	defer func(inFile *os.File) {
		if e := inFile.Close(); e != nil {
			panic(e)
		}
	}(inFile)

	// Make temp dir
	mkTemp, err := temp.MkTemp()
	if err != nil {
		return fmt.Errorf("error making temp dir: %v", err)
	}
	defer func(mkTemp temp.Dir) {
		if e := mkTemp.Release(); e != nil {
			panic(e)
		}
	}(mkTemp)
	if !quiet {
		log.Println("Made temp dir:", mkTemp.Path())
	}

	// Create output file
	outFile, err := os.Create(filepath.Join(mkTemp.Path(), "compressed"))
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer func(outFile *os.File) {
		if e := outFile.Close(); e != nil {
			panic(e)
		}
	}(outFile)

	// Create compress
	compressed, err := zstd.Compress(inFile, outFile)
	if err != nil {
		return fmt.Errorf("error compressing file: %v", err)
	}
	if !quiet {
		log.Println("Compressed", input, "to", compressed, "bytes")
	}

	// Release decompressor source code packet
	if err = releaseEmbedded(mkTemp.Path()); err != nil {
		return fmt.Errorf("error releasing embedded resource code: %v", err)
	}
	if !quiet {
		log.Println("Released embedded resource code")
	}

	// Decompress decompressor
	if err = zstd.BatchDecompress(
		filepath.Join(mkTemp.Path(), "decompressorSourceCode"),
		mkTemp.Path(),
	); err != nil {
		return fmt.Errorf("error decompressing decompressor: %v", err)
	}
	if !quiet {
		log.Println("Decompressed decompressor")
	}

	// Tidy go mod
	// Pack up with decompressor
	_, err = runner.Run(
		[]string{},
		goPath, []string{
			"mod", "tidy",
		}, mkTemp.Path(),
		quiet,
	)
	if err != nil {
		return fmt.Errorf("error tiding mod with go: %v", err)
	}
	if !quiet {
		log.Println("Tidied up with go")
	}

	// Pack up with decompressor
	_, err = runner.Run(
		[]string{"GOOS=" + system, "GOARCH=" + arch},
		goPath, []string{
			"build",
			"-ldflags=-s", "-ldflags=-w",
			"-trimpath",
			"-tags", method,
			"-o", "output",
		}, mkTemp.Path(),
		quiet,
	)
	if err != nil {
		return fmt.Errorf("error packing with decompressor: %v", err)
	}
	if !quiet {
		log.Println("Packed with decompressor")
	}

	// If UPX enabled then pack with UPX
	if upx {
		// TODO: works very weird ?
		if runtime.GOOS != "windows" && system == "windows" {
			_, err = runner.Run(
				[]string{},
				winePath, []string{
					expr.Ternary(strings.HasSuffix(upxPath, ".exe"), upxPath, upxPath+".exe"),
					"output",
					"-9",
				},
				mkTemp.Path(),
				quiet,
			)
			if err != nil {
				return fmt.Errorf("error packing with UPX by wine: %v", err)
			}
			if !quiet {
				log.Println("Packed with UPX by wine")
			}
		} else {
			_, err = runner.Run(
				[]string{},
				upxPath, []string{
					"output",
					"-9",
				},
				mkTemp.Path(),
				quiet,
			)
			if err != nil {
				return fmt.Errorf("error packing with UPX: %v", err)
			}
			if !quiet {
				log.Println("Packed with UPX")
			}
		}
	}

	// Move to output destination
	err = temp.MvFile(
		filepath.Join(mkTemp.Path(), "output"),
		filepath.Join(
			workDir,
			expr.Ternary(strings.HasSuffix(output, ".exe") || system != "windows", output, output+".exe"),
		),
	)
	if err != nil {
		return fmt.Errorf("error moving result: %v", err)
	}
	if !quiet {
		log.Println("Moved result")
	}

	// Set executable permission
	if runtime.GOOS != "windows" && system != "windows" {
		if err = os.Chmod(filepath.Join(workDir, output), 0755); err != nil {
			return fmt.Errorf("setting executable permission failed: %v", err)
		}
	}

	return nil
}
