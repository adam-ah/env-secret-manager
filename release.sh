#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

if [[ $# -ne 1 || ! "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
  echo "usage: ./release.sh vMAJOR.MINOR.PATCH[-PRERELEASE]" >&2
  exit 1
fi

./build.sh

tag="$1"
release_dir="dist/$tag"
mkdir -p "$release_dir"

for target_os in linux darwin; do
  for target_arch in amd64 arm64; do
    package="secretenv_${tag}_${target_os}_${target_arch}"
    staging="$release_dir/$package"
    mkdir -p "$staging"
    GOOS="$target_os" GOARCH="$target_arch" CGO_ENABLED=0 go build \
      -mod=readonly -trimpath -buildvcs=false -ldflags="-s -w" \
      -o "$staging/secretenv" ./cmd/secretenv
    cp README.md "$staging/"
    mkdir -p "$staging/docs"
    cp docs/usage.md docs/workspace.png docs/ai-access.png docs/demo.gif "$staging/docs/"
    mkdir -p "$staging/skills/secretenv"
    cp skills/secretenv/SKILL.md "$staging/skills/secretenv/"
    tar -czf "$release_dir/$package.tar.gz" -C "$staging" secretenv README.md docs skills/secretenv/SKILL.md
  done
done

(
  cd "$release_dir"
  shasum -a 256 ./*.tar.gz > checksums.txt
)

echo "Release archives and checksums: $release_dir"
