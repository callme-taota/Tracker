#!/bin/sh
# Run tests. On macOS, use external linker to avoid "missing LC_UUID" dyld error.
set -e
cd "$(dirname "$0")"
if [ "$(uname)" = "Darwin" ]; then
  go test -ldflags="-linkmode=external" ./...
else
  go test ./...
fi
