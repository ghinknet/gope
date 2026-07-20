# GoPE

GoPE is a cross-platform executable compressor built around the Go compiler. It compresses an input executable, embeds the compressed payload in a small launcher, and restores and runs the original program when the packed executable starts.

## Requirements

- Building GoPE requires Go 1.26.5 or later.
- Packing an executable requires an available Go toolchain. Cross-compilation support is determined by `go tool dist list`.
- Running an already packed executable does not require Go.
- UPX is entirely optional. Wine and a Windows UPX executable are required only when `--upx` is enabled for a Windows target from a Linux host.

## Build and Test

The gope binary embeds the decompressor source tree via `//go:embed`. Because Go
cannot embed a directory that contains a `go.mod`, the decompressor is kept as a
normal part of the parent module (no `go.mod` on disk) and the launcher's module
files are generated from the parent module at build time. The `Makefile` handles
this automatically, so build and test through `make` rather than calling `go`
directly:

```sh
make build   # build ./gope for the host platform
make test    # run the test suite (default and gzip tags)
make vet     # go vet ./...
make check   # test + vet
make integration  # pack a sample program and verify it runs correctly
```

Build GoPE for every supported desktop and server target:

```sh
make dist BUILD_ARGS="-n GoPE -p 4"
make dist VERSION=v0.0.1          # inject the version: GoPE-v0.0.1-<os>-<arch>
```

Artefacts are written to `dists/`. When `VERSION` is set it is both embedded in
the binary (`gope --version`) and included in each artefact's file name, ready for
a release upload. The builder returns a non-zero exit code if any target fails.
`make clean` removes build artefacts and any generated module files.

> A bare `go build .` will fail because the generated `decompressor/go.mod.template`
> is not present outside a `make` build; always use `make`.

## Usage

Pack an executable with the default zstd compression method:

```sh
./gope -i ./my-program -o ./my-program-packed
```

Use gzip with a higher compression level and cross-compile the launcher:

```sh
./gope \
  -i ./my-program-linux-amd64 \
  -o /tmp/my-program-packed \
  -m gzip \
  -l 9 \
  -s linux \
  -a amd64
```

Build a Windows launcher from Linux or macOS without UPX or Wine:

```sh
./gope \
  -i ./my-program-windows-amd64.exe \
  -o ./my-program-packed.exe \
  -s windows \
  -a amd64
```

GoPE preserves command-line arguments, standard input and output, the caller's working directory, and the wrapped program's exit code.

## Run Modes

- `mask` is the default. The original program is restored into a private temporary directory on every run and removed afterward. The directory containing the launcher does not need to be writable.
- `replace` atomically replaces the launcher with the original program on its first run and then starts the original program. This operation is permanent and requires the launcher's directory to be writable. Windows does not support this mode.

## Optional UPX Compression

Pass `--upx` to compress the generated launcher a second time with UPX:

```sh
./gope -i ./my-program -o ./packed --upx --upx-level 9
```

UPX is not involved unless `--upx` is explicitly specified. The currently supported host and target combinations are:

- Windows host to Windows target: native UPX
- Linux host to Linux target: native UPX
- Linux host to Windows target: Windows UPX through Wine

For the last combination, Wine is needed only because the optional UPX step cannot use the native Linux UPX executable for this cross-target packing workflow. Normal Go-based Windows packing remains independent of Wine.

`--upx-level 10` maps to UPX's maximum level, 9.

## Notes and Limitations

- GoPE does not verify that the input executable matches `--system` and `--arch`; these options describe the generated launcher target.
- `mask` mode requires a system temporary directory that permits executable files. A temporary directory mounted with `noexec` prevents the launcher from running.
- Small or already compressed programs can become larger because of the Go launcher overhead.
- Platform-protected macOS binaries, signatures, and entitlements can restrict execution after copying. Test packed artefacts on their destination systems before release.
