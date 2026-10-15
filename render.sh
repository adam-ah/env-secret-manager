#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

agg_bin="${AGG_BIN:-agg}"
if ! command -v "$agg_bin" > /dev/null; then
  agg_bin=./agg-x86_64-unknown-linux-gnu
fi
if ! command -v "$agg_bin" > /dev/null || ! command -v ffmpeg > /dev/null; then
  echo "Rendering needs asciinema's agg and ffmpeg. Set AGG_BIN if agg is outside PATH." >&2
  exit 1
fi

SECRETENV_RENDER_DOCS=1 go test -run '^TestRenderDocumentation$' -count=1 .
"$agg_bin" --theme github-light --font-size 16 --idle-time-limit 10 \
  --last-frame-duration 4 demo.cast demo.gif
mkdir -p docs
ffmpeg -hide_banner -loglevel error -y -i demo.gif -vf 'select=eq(n\,1)' -frames:v 1 -update 1 docs/workspace.png
ffmpeg -hide_banner -loglevel error -y -i demo.gif -vf 'select=eq(n\,3)' -frames:v 1 -update 1 docs/ai-access.png

echo "Updated demo.gif, docs/workspace.png and docs/ai-access.png"
