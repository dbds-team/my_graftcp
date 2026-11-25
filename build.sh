#!/usr/bin/env bash
# Linux-only: the embedded C tracer requires Linux ptrace/seccomp APIs.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")"

case "${1:-linux}" in
  linux) ;;
  -h|--help)
    echo 'Usage: ./build.sh [linux|--help] [make arguments...]'
    echo 'Examples: ./build.sh; ./build.sh linux VERSION=v1.1.0; make CROSS_COMPILE=aarch64-linux-gnu-'
    echo 'Linux only. Windows and all (Linux+Windows/UPX) are unsupported: ptrace has no Windows equivalent.'
    exit 0 ;;
  windows|all)
    echo 'Unsupported target: graftcp requires Linux ptrace/seccomp; Windows builds are not available.' >&2
    exit 2 ;;
  *) echo "Unknown target: $1 (see --help)" >&2; exit 2 ;;
esac
if (($#)); then shift; fi

# Make intentionally uses GOTOOLCHAIN=local. Resolve/download the module's Go
# toolchain first so an older host Go cannot silently break the build.
go_root="$(cd local && go env GOROOT)"
exec make GO="$go_root/bin/go" "$@" all
