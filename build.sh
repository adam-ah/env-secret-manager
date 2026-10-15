#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

./test.sh

APP="secretenv"
OUT_DIR="${OUT_DIR:-bin}"
OUT="$OUT_DIR/$APP"

mkdir -p "$OUT_DIR"

CGO_ENABLED=0 go build \
  -mod=readonly \
  -trimpath \
  -buildvcs=false \
  -ldflags="-s -w" \
  -o "$OUT" \
  .

chmod +x "$OUT"

echo "built: $OUT"
