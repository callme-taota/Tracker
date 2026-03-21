#!/bin/sh
# Build tracker. On macOS, use external linker to avoid "missing LC_UUID" dyld error.
set -e
cd "$(dirname "$0")"
if [ "$(uname)" = "Darwin" ]; then
  go build -ldflags="-linkmode=external" -o tracker ./cmd/tracker
else
  go build -o tracker ./cmd/tracker
fi
echo "Built: ./tracker"
