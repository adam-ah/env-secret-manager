#!/usr/bin/env bash
set -euo pipefail

APP="secretenv"
OUT_DIR="${OUT_DIR:-bin}"
OUT="$OUT_DIR/$APP"

mkdir -p "$OUT_DIR"

go mod tidy

CGO_ENABLED=0 go build \
  -trimpath \
  -buildvcs=false \
  -ldflags="-s -w" \
  -o "$OUT" \
  .

chmod +x "$OUT"

echo "built: $OUT"
