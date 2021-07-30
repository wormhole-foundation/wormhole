#!/usr/bin/env bash
set -euo pipefail

(
  cd tools/
  ./build.sh
)

# TODO(leo): remove after a while
rm -rf third_party/googleapis

rm -rf bridge/pkg/proto

tools/bin/buf mod update
tools/bin/buf lint
tools/bin/buf generate
