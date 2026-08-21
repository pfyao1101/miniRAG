#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd
)"

readonly ROOT_DIR="$(cd -- "$SCRIPT_DIR/../.." && pwd)"

readonly BUF_BIN="${BUF_BIN:-buf}"

cd "$ROOT_DIR"
"$BUF_BIN" generate
