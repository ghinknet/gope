# Makefile for GoPE
#
# The gope binary embeds the decompressor source tree via `//go:embed
# decompressor`. Go refuses to embed a directory that contains a `go.mod`
# (it treats it as a nested module), so the decompressor source is kept as a
# normal part of the parent module and carries NO go.mod on disk. The
# standalone launcher that gope releases at pack time still needs its own
# module files, so this Makefile generates them from the parent module right
# before any build that compiles the root package, then removes them again.
#
# Because the module files are derived from the parent go.mod / go.sum, the
# decompressor's dependency versions never need to be maintained by hand.
#
# Use `make` for building and testing; a bare `go build .` will fail unless the
# generated template is present.

GO      ?= go
BINARY  ?= gope

DECOMP_DIR      := decompressor
DECOMP_TEMPLATE := $(DECOMP_DIR)/go.mod.template
DECOMP_SUM      := $(DECOMP_DIR)/go.sum

# Generate the embedded launcher's module files from the parent module so the
# released decompressor builds standalone with matching dependency versions.
GEN_EMBED = \
	go_version=$$(awk '/^go /{print $$2; exit}' go.mod); \
	kp_version=$$(awk '$$1=="github.com/klauspost/compress"{print $$2; exit}' go.mod); \
	if [ -z "$$go_version" ] || [ -z "$$kp_version" ]; then \
		echo "error: could not resolve go/klauspost versions from go.mod" >&2; exit 1; \
	fi; \
	printf 'module gope/decompressor\n\ngo %s\n\nrequire github.com/klauspost/compress %s\n' \
		"$$go_version" "$$kp_version" > $(DECOMP_TEMPLATE); \
	grep '^github.com/klauspost/compress ' go.sum > $(DECOMP_SUM)

CLEAN_EMBED = rm -f $(DECOMP_TEMPLATE) $(DECOMP_SUM)

.DEFAULT_GOAL := build
.PHONY: build dist test vet check integration clean help

## build: build the gope binary for the host platform
build:
	@set -e; \
	trap '$(CLEAN_EMBED)' EXIT; \
	$(GEN_EMBED); \
	echo ">> building $(BINARY)"; \
	$(GO) build -ldflags="-s -w" -trimpath -o $(BINARY) .

## dist: cross-compile gope for every supported platform into dists/
## Pass VERSION=vX.Y.Z to inject the version and name artifacts GoPE-vX.Y.Z-os-arch.
## Extra builder flags may be passed via BUILD_ARGS.
dist:
	@set -e; \
	trap '$(CLEAN_EMBED)' EXIT; \
	$(GEN_EMBED); \
	echo ">> cross-compiling into dists/"; \
	$(GO) run ./builder/build.go $(if $(VERSION),-v $(VERSION)) $(BUILD_ARGS)

## test: run the test suite (default and gzip build tags)
test:
	@set -e; \
	trap '$(CLEAN_EMBED)' EXIT; \
	$(GEN_EMBED); \
	echo ">> testing"; \
	$(GO) test ./...; \
	$(GO) test -tags gzip ./decompressor/...

## vet: run go vet across the module
vet:
	@set -e; \
	trap '$(CLEAN_EMBED)' EXIT; \
	$(GEN_EMBED); \
	echo ">> vetting"; \
	$(GO) vet ./...

## check: run tests and vet
check: test vet

## integration: build gope and verify packing end-to-end (pack a program and run it)
integration: build
	@echo ">> running integration test"; \
	BINARY="$(CURDIR)/$(BINARY)" GO="$(GO)" scripts/integration-test.sh

## clean: remove build artifacts and any leftover generated module files
clean:
	rm -f $(BINARY)
	rm -rf dists
	$(CLEAN_EMBED)

## help: list available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //'
