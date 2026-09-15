#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  target_os=${target%/*}
  target_arch=${target#*/}
  target_file="internal/provision/assets/$target_os-$target_arch"
  CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build -trimpath -ldflags='-s -w' -o "$target_file" ./cmd/sess-agent
  gzip -n -f "$target_file"
done
