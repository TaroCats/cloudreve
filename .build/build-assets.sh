#!/bin/bash
# Build the frontend and wrap it into application/statics/assets.zip,
# which the Go binary embeds via `//go:embed assets.zip`.
#
# Usage:
#   build-assets.sh            # just build (no version change) — local dev
#   build-assets.sh <version>  # build and stamp package.json version — CI release
#
# Frontend toolchain is bun (project convention), not yarn/npm.
set -euo pipefail

# Vite/metadata minification can peak well above the default heap on CI runners.
export NODE_OPTIONS="--max-old-space-size=8192"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}/assets"

echo "==> Installing frontend dependencies (bun)"
bun install --frozen-lockfile 2>/dev/null || bun install

# Only stamp a version when the caller explicitly asks for one.
# Never bump versions implicitly — that is a deliberate release action.
if [ -n "${1:-}" ]; then
    echo "==> Stamping package.json version -> ${1}"
    bun pm pkg set version="${1#v}"
fi

echo "==> Building frontend"
bun run build

echo "==> Packaging assets.zip"
cd "${ROOT_DIR}"
if [ -d build ]; then
    echo "note: unexpected ./build at repo root, ignoring"
fi
# zip paths must be relative to the repo root so the embed layout stays
# `assets/build/...`, matching what statics.go expects.
rm -f application/statics/assets.zip
zip -r -q application/statics/assets.zip assets/build

echo "==> Done: application/statics/assets.zip"
du -h application/statics/assets.zip
