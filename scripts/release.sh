#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
./scripts/build-agents.sh
mkdir -p dist
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
    target_os=${target%/*}
    target_arch=${target#*/}
    release_dir="dist/sess-$target_os-$target_arch"
    mkdir -p "$release_dir"
    CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build -trimpath -ldflags='-s -w' -o "$release_dir/sess" ./cmd/sess
    cp README.md LICENSE "$release_dir/"
    tar -czf "$release_dir.tar.gz" -C "$release_dir" sess README.md LICENSE
done
(cd dist && shasum -a 256 sess-*.tar.gz > SHA256SUMS)
