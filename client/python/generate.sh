#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd
)"

readonly ROOT_DIR="$(cd -- "$SCRIPT_DIR/../.." && pwd)"

readonly PYTHON_BIN="${PYTHON_BIN:-python3}"

"$PYTHON_BIN" -m grpc_tools.protoc \
  -I "$ROOT_DIR/api" \
  --python_out="$ROOT_DIR/client/python" \
  --pyi_out="$ROOT_DIR/client/python" \
  --grpc_python_out="$ROOT_DIR/client/python" \
  "$ROOT_DIR/api/minirag/v1/storage.proto"
