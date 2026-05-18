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

	"github.com/spf13/cobra"
	"go.gh.ink/gope/internal/compress/gzip"
	"go.gh.ink/gope/internal/compress/zstd"
	"go.gh.ink/gope/internal/constant"
	"go.gh.ink/gope/internal/runner"
	"go.gh.ink/gope/internal/temp"
	"go.gh.ink/toolbox/expr"
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
var runMode string
var level int
var upxLevel int

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
	rootCmd.Flags().StringVarP(&runMode, "run-mode", "r", "mask", "Decompressor run mode: mask or replace")
	rootCmd.Flags().IntVarP(&level, "level", "l", 3, "Compression level 1-10")
	rootCmd.Flags().IntVarP(&upxLevel, "upx-level", "U", 9, "UPX compression level 1-10")

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
	return run()
}

func run() error {
	if err := validateFlags(); err != nil {
		return err
	}

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting work dir: %w", err)
	}

	inFile, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("error opening input file: %w", err)
	}
	defer func(inFile *os.File) {
		if e := inFile.Close(); e != nil {
			panic(e)
		}
	}(inFile)

	mkTemp, err := temp.MkTemp()
	if err != nil {
		return fmt.Errorf("error making temp dir: %w", err)
	}
	defer func(mkTemp temp.Dir) {
		if e := mkTemp.Release(); e != nil {
			panic(e)
		}
	}(mkTemp)
	if !quiet {
		log.Println("Made temp dir:", mkTemp.Path())
	}

	if err = compressInput(inFile, mkTemp.Path()); err != nil {
		return err
	}

	if err = releaseEmbedded(mkTemp.Path()); err != nil {
		return fmt.Errorf("error releasing embedded resource code: %w", err)
	}
	if !quiet {
		log.Println("Released embedded resource code")
	}

	if err = zstd.BatchDecompress(
		filepath.Join(mkTemp.Path(), "decompressorSourceCode"),
		mkTemp.Path(),
	); err != nil {
		return fmt.Errorf("error decompressing decompressor: %w", err)
	}
	if !quiet {
		log.Println("Decompressed decompressor")
	}

	if err = tidyModule(mkTemp.Path()); err != nil {
		return err
	}

	if err = buildDecompressor(mkTemp.Path()); err != nil {
		return err
	}

	if err = packWithUpxIfNeeded(mkTemp.Path()); err != nil {
		return err
	}

	if err = moveOutput(mkTemp.Path(), workDir); err != nil {
		return err
	}

	if err = chmodOutputIfNeeded(workDir); err != nil {
		return err
	}

	return nil
}

func validateFlags() error {
	if input == "" {
		return fmt.Errorf("input file is required")
	}
	if !slices.Contains(constant.SupportedMethods, method) {
		return fmt.Errorf("method %s is not supported", method)
	}
	if !slices.Contains(constant.SupportedPlatforms, system+"/"+arch) {
		return fmt.Errorf("unsupported platform: %s/%s", system, arch)
	}
	if runMode != "mask" && runMode != "replace" {
		return fmt.Errorf("unsupported run mode: %s", runMode)
	}
	if system == "windows" {
		runMode = "mask"
	}
	if level < 1 || level > 10 {
		return fmt.Errorf("compression level must be 1-10")
	}
	if upxLevel < 1 || upxLevel > 10 {
		return fmt.Errorf("upx level must be 1-10")
	}
	return nil
}

func compressInput(inFile *os.File, tempDir string) error {
	outFile, err := os.Create(filepath.Join(tempDir, "compressed"))
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer func(outFile *os.File) {
		if e := outFile.Close(); e != nil {
			panic(e)
		}
	}(outFile)

	var compressed int64
	switch method {
	case "zstd":
		compressed, err = zstd.Compress(inFile, outFile, level)
	case "gzip":
		compressed, err = gzip.Compress(inFile, outFile, level)
	default:
		return fmt.Errorf("unsupported method: %s", method)
	}
	if err != nil {
		return fmt.Errorf("error compressing file: %w", err)
	}
	if !quiet {
		log.Println("Compressed", input, "to", compressed, "bytes")
	}
	return nil
}

func tidyModule(tempDir string) error {
	_, err := runner.Run(
		[]string{},
		goPath,
		[]string{"mod", "tidy"},
		tempDir,
		quiet,
	)
	if err != nil {
		return fmt.Errorf("error tiding mod with go: %w", err)
	}
	if !quiet {
		log.Println("Tidied up with go")
	}
	return nil
}

func buildDecompressor(tempDir string) error {
	ldFlags := []string{
		"-s",
		"-w",
		"-X", "main.buildMode=" + runMode,
	}
	_, err := runner.Run(
		[]string{"GOOS=" + system, "GOARCH=" + arch},
		goPath,
		[]string{
			"build",
			"-ldflags=" + strings.Join(ldFlags, " "),
			"-trimpath",
			"-tags", method,
			"-o", "output",
		},
		tempDir,
		quiet,
	)
	if err != nil {
		return fmt.Errorf("error packing up: %w", err)
	}
	if !quiet {
		log.Println("Packed up successfully")
	}
	return nil
}

func packWithUpxIfNeeded(tempDir string) error {
	if !upx {
		return nil
	}

	host := runtime.GOOS
	target := system

	switch {
	case host == "windows" && target == "windows":
		return runUpx(tempDir, upxPath, false)
	case host == "linux" && target == "linux":
		return runUpx(tempDir, upxPath, false)
	case host == "linux" && target == "windows":
		return runUpx(tempDir, upxPath, true)
	default:
		return fmt.Errorf("UPX is not supported for host %s and target %s", host, target)
	}
}

func runUpx(tempDir string, upxPath string, useWine bool) error {
	upxArg := fmt.Sprintf("-%d", mapUpxLevel(upxLevel))
	if useWine {
		_, err := runner.Run(
			[]string{},
			winePath,
			[]string{
				expr.Ternary(strings.HasSuffix(upxPath, ".exe"), upxPath, upxPath+".exe"),
				"output",
				upxArg,
			},
			tempDir,
			quiet,
		)
		if err != nil {
			return fmt.Errorf("error packing with UPX by wine: %w", err)
		}
		if !quiet {
			log.Println("Packed with UPX by wine")
		}
		return nil
	}

	_, err := runner.Run(
		[]string{},
		upxPath,
		[]string{"output", upxArg},
		tempDir,
		quiet,
	)
	if err != nil {
		return fmt.Errorf("error packing with UPX: %w", err)
	}
	if !quiet {
		log.Println("Packed with UPX")
	}
	return nil
}

func mapUpxLevel(level int) int {
	if level < 1 {
		return 1
	}
	if level >= 9 {
		return 9
	}
	return level
}

func moveOutput(tempDir string, workDir string) error {
	outputPath := filepath.Join(
		workDir,
		expr.Ternary(strings.HasSuffix(output, ".exe") || system != "windows", output, output+".exe"),
	)
	if err := temp.MvFile(filepath.Join(tempDir, "output"), outputPath); err != nil {
		return fmt.Errorf("error moving result: %w", err)
	}
	if !quiet {
		log.Println("Moved result")
	}
	return nil
}

func chmodOutputIfNeeded(workDir string) error {
	if runtime.GOOS == "windows" || system == "windows" {
		return nil
	}
	if err := os.Chmod(filepath.Join(workDir, output), 0755); err != nil {
		return fmt.Errorf("setting executable permission failed: %w", err)
	}
	return nil
}
