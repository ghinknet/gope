package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"gope/internal/compress/gzip"
	"gope/internal/compress/zstd"
	"gope/internal/meta"
	"gope/internal/runner"
	"gope/internal/temp"

	"github.com/spf13/cobra"
	"go.gh.ink/toolbox/expr"
)

// Embedded decompressor source tree used to build the runtime wrapper.
//
//go:embed decompressor
var embeddedDecompressor embed.FS

// releaseEmbedded releases the embedded decompressor module into temp.
func releaseEmbedded(temp string) error {
	return fs.WalkDir(embeddedDecompressor, "decompressor", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel("decompressor", path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		// The payload is generated for every packing operation and must never
		// be copied from the source placeholder.
		if filepath.ToSlash(rel) == "compressed" {
			return nil
		}
		if filepath.ToSlash(rel) == "go.mod.template" {
			rel = "go.mod"
		}
		target := filepath.Join(temp, filepath.FromSlash(rel))
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := embeddedDecompressor.ReadFile(path)
		if err != nil {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

// CLI configuration.
var rootCmd = &cobra.Command{
	Use: meta.Name,
	Long: `Go Packer for Executable
A go-based cross platform binary compressor.

"Well. Your physical education teacher is not sick :D"`,
	Version:       meta.Version,
	RunE:          Runner,
	Args:          cobra.NoArgs,
	SilenceErrors: true,
	SilenceUsage:  true,
}

// CLI flags.
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Runner
// Main pack pipeline: validate -> compress -> release decompressor -> build wrapper -> optional UPX -> move output.
func Runner(cmd *cobra.Command, args []string) error {
	return run()
}

func run() (retErr error) {
	if err := validateFlags(); err != nil {
		return err
	}
	if err := validateTargetPlatform(); err != nil {
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

	mkTemp, err := temp.MkTemp()
	if err != nil {
		return fmt.Errorf("error making temp dir: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, mkTemp.Release())
	}()
	if !quiet {
		log.Println("Made temp dir:", mkTemp.Path())
	}

	compressErr := compressInput(inFile, mkTemp.Path())
	closeErr := inFile.Close()
	if err = errors.Join(compressErr, closeErr); err != nil {
		return err
	}

	if err = releaseEmbedded(mkTemp.Path()); err != nil {
		return fmt.Errorf("error releasing embedded decompressor source: %w", err)
	}
	if !quiet {
		log.Println("Released embedded decompressor source")
	}

	if err = buildDecompressor(mkTemp.Path()); err != nil {
		return err
	}

	if err = packWithUpxIfNeeded(mkTemp.Path()); err != nil {
		return err
	}

	outputPath, err := moveOutput(mkTemp.Path(), workDir)
	if err != nil {
		return err
	}

	if err = chmodOutputIfNeeded(outputPath); err != nil {
		return err
	}

	return nil
}

// Validate and normalise user input.
func validateFlags() error {
	if input == "" {
		return fmt.Errorf("input file is required")
	}
	if !slices.Contains(meta.SupportedMethods, method) {
		return fmt.Errorf("method %s is not supported", method)
	}
	if runMode != "mask" && runMode != "replace" {
		return fmt.Errorf("unsupported run mode: %s", runMode)
	}
	if system == "windows" && runMode == "replace" {
		return fmt.Errorf("replace mode is not supported on windows")
	}
	if level < 1 || level > 10 {
		return fmt.Errorf("compression level must be 1-10")
	}
	if upxLevel < 1 || upxLevel > 10 {
		return fmt.Errorf("upx level must be 1-10")
	}
	return nil
}

func validateTargetPlatform() error {
	if !meta.IsSupportedSystem(system) {
		return fmt.Errorf("target system %s is not supported by GoPE", system)
	}
	cmd := exec.Command(goPath, "tool", "dist", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error listing Go target platforms: %w: %s", err, strings.TrimSpace(string(output)))
	}
	target := system + "/" + arch
	if !slices.Contains(strings.Fields(string(output)), target) {
		return fmt.Errorf("target platform %s is not supported by %s", target, goPath)
	}
	return nil
}

// Compress input into a temporary archive.
func compressInput(inFile *os.File, tempDir string) error {
	outFile, err := os.Create(filepath.Join(tempDir, "compressed"))
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	var compressed int64
	switch method {
	case "zstd":
		compressed, err = zstd.Compress(inFile, outFile, level)
	case "gzip":
		compressed, err = gzip.Compress(inFile, outFile, level)
	default:
		err = fmt.Errorf("unsupported method: %s", method)
	}
	closeErr := outFile.Close()
	if err = errors.Join(err, closeErr); err != nil {
		return fmt.Errorf("error compressing file: %w", err)
	}
	if !quiet {
		log.Println("Compressed", input, "to", compressed, "bytes")
	}
	return nil
}

// Build the decompressor wrapper with the selected build mode.
func buildDecompressor(tempDir string) error {
	ldFlags := []string{
		"-s",
		"-w",
		"-X", "main.buildMode=" + runMode,
	}
	err := runChecked(
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

// Apply UPX with strict host/target rules.
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

// Run UPX with or without wine.
func runUpx(tempDir string, upxPath string, useWine bool) error {
	upxArg := fmt.Sprintf("-%d", mapUpxLevel(upxLevel))
	if useWine {
		err := runChecked(
			[]string{},
			winePath,
			[]string{
				expr.Ternary(strings.EqualFold(filepath.Ext(upxPath), ".exe"), upxPath, upxPath+".exe"),
				upxArg,
				"output",
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

	err := runChecked(
		[]string{},
		upxPath,
		[]string{upxArg, "output"},
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

func runChecked(envs []string, binaryPath string, args []string, workingDir string, quiet bool) error {
	code, err := runner.Run(envs, binaryPath, args, workingDir, quiet)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("command exited with code %d", code)
	}
	return nil
}

// Move output to the requested destination.
func moveOutput(tempDir string, workDir string) (string, error) {
	outputPath := resolveOutputPath(workDir)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", fmt.Errorf("error creating output directory: %w", err)
	}
	if err := temp.MvFile(filepath.Join(tempDir, "output"), outputPath); err != nil {
		return "", fmt.Errorf("error moving result: %w", err)
	}
	if !quiet {
		log.Println("Moved result")
	}
	return outputPath, nil
}

func resolveOutputPath(workDir string) string {
	outputName := output
	if system == "windows" && !strings.EqualFold(filepath.Ext(outputName), ".exe") {
		outputName += ".exe"
	}
	if filepath.IsAbs(outputName) {
		return filepath.Clean(outputName)
	}
	return filepath.Join(workDir, outputName)
}

// Fix executable bit on non-Windows targets.
func chmodOutputIfNeeded(outputPath string) error {
	if runtime.GOOS == "windows" || system == "windows" {
		return nil
	}
	if err := os.Chmod(outputPath, 0755); err != nil {
		return fmt.Errorf("setting executable permission failed: %w", err)
	}
	return nil
}
