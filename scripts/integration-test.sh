#!/bin/sh
# Integration test for gope.
#
# Builds a small test program, packs it with gope across compression methods
# and run modes, then runs the packed executable and verifies:
#   * the wrapped program actually runs and its stdout is passed through,
#   * the process exit code is propagated,
#   * command-line arguments are forwarded,
#   * in mask mode the wrapped program resolves its own location
#     (os.Executable) to the packed file's directory, not a temp directory
#     (regression guard for the mask-mode working-directory bug).
#
# Environment:
#   BINARY  path to the gope binary to test (default: ./gope)
#   GO      go command to use (default: go)
set -eu

BINARY="${BINARY:-./gope}"
GO="${GO:-go}"

if [ ! -x "$BINARY" ]; then
	echo "error: gope binary not found at $BINARY" >&2
	exit 1
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT INT TERM

# A tiny program that reports enough to verify pass-through behaviour.
mkdir -p "$WORK/prog"
cat > "$WORK/prog/main.go" <<'EOF'
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	fmt.Println("GOPE_MARKER")
	fmt.Printf("args=%v\n", os.Args[1:])
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	fmt.Printf("exedir=%s\n", filepath.Dir(exe))
	// Exit code 42 lets the harness verify exit-code propagation.
	os.Exit(42)
}
EOF

( cd "$WORK/prog" && GO111MODULE=off "$GO" build -o "$WORK/prog-bin" . )

pass=0
fail=0

# run_case METHOD MODE
run_case() {
	method="$1"
	mode="$2"
	name="${method}-${mode}"

	casedir="$WORK/case-$name"
	mkdir -p "$casedir"
	packed="$casedir/packed"

	if ! "$BINARY" -q -i "$WORK/prog-bin" -o "$packed" -m "$method" -r "$mode"; then
		echo "[FAIL] $name: packing failed"
		fail=$((fail + 1))
		return
	fi

	# Run from an unrelated directory to make sure exedir reflects the packed
	# file's location and not the caller's working directory. Capture stdout
	# and the exit code in one shot.
	out="$("$packed" alpha beta 2>/dev/null)" && code=0 || code=$?

	ok=1
	echo "$out" | grep -q '^GOPE_MARKER$' || { echo "[FAIL] $name: missing marker"; ok=0; }
	echo "$out" | grep -q '^args=\[alpha beta\]$' || { echo "[FAIL] $name: args not forwarded: $out"; ok=0; }
	if [ "$code" -ne 42 ]; then
		echo "[FAIL] $name: exit code not propagated (got $code, want 42)"
		ok=0
	fi

	# Regression guard: exedir must be the packed file's directory.
	want_dir="$(cd "$casedir" && pwd -P)"
	got_dir="$(echo "$out" | sed -n 's/^exedir=//p')"
	# Resolve symlinks (macOS /tmp -> /private/tmp) for a stable comparison.
	if [ -d "$got_dir" ]; then
		got_dir="$(cd "$got_dir" && pwd -P)"
	fi
	if [ "$got_dir" != "$want_dir" ]; then
		echo "[FAIL] $name: wrapped exedir is $got_dir, want $want_dir"
		ok=0
	fi

	if [ "$ok" -eq 1 ]; then
		echo "[PASS] $name"
		pass=$((pass + 1))
	else
		fail=$((fail + 1))
	fi
}

run_case zstd mask
run_case gzip mask
run_case zstd replace
run_case gzip replace

echo "integration: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
