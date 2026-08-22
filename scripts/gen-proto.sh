#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROTO_DIR="${ROOT}/contracts/proto"
GO_OUT="${ROOT}/contracts/gen/go"
PY_OUT="${ROOT}/contracts/gen/python"

mkdir -p "${GO_OUT}" "${PY_OUT}"

PROTO_FILES=(
  "${PROTO_DIR}/restick/v1/ping.proto"
  "${PROTO_DIR}/restick/v1/object.proto"
  "${PROTO_DIR}/restick/v1/job.proto"
  "${PROTO_DIR}/restick/v1/auth.proto"
)

protoc \
  -I "${PROTO_DIR}" \
  --go_out="${GO_OUT}" \
  --go_opt=paths=source_relative \
  --go-grpc_out="${GO_OUT}" \
  --go-grpc_opt=paths=source_relative \
  "${PROTO_FILES[@]}"

PYTHON="${ROOT}/.tools/proto-venv/bin/python"
if [[ ! -x "${PYTHON}" ]]; then
  python3 -m venv "${ROOT}/.tools/proto-venv"
  "${ROOT}/.tools/proto-venv/bin/pip" install -q grpcio-tools protobuf
fi

"${PYTHON}" -m grpc_tools.protoc \
  -I "${PROTO_DIR}" \
  --python_out="${PY_OUT}" \
  --grpc_python_out="${PY_OUT}" \
  --pyi_out="${PY_OUT}" \
  "${PROTO_FILES[@]}"

# Make restick.v1 importable on PYTHONPATH=contracts/gen/python
touch "${PY_OUT}/restick/__init__.py"
touch "${PY_OUT}/restick/v1/__init__.py"

"${PYTHON}" - "${ROOT}" <<'PY'
from pathlib import Path
import sys
root = Path(sys.argv[1]) / "contracts/gen/python/restick/v1"
for path in root.glob("*_pb2_grpc.py"):
    text = path.read_text()
    text = text.replace("import ping_pb2 as ping__pb2", "from restick.v1 import ping_pb2 as ping__pb2")
    text = text.replace("import object_pb2 as object__pb2", "from restick.v1 import object_pb2 as object__pb2")
    text = text.replace("import job_pb2 as job__pb2", "from restick.v1 import job_pb2 as job__pb2")
    text = text.replace("import auth_pb2 as auth__pb2", "from restick.v1 import auth_pb2 as auth__pb2")
    path.write_text(text)
PY

echo "generated go -> ${GO_OUT}"
echo "generated python -> ${PY_OUT}"
