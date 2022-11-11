#!/bin/bash
TMP=$(mktemp -d)
f1="$TMP/$1.interface"
f2="$TMP/$2.interface"
mkdir -p $(dirname "$f1")
mkdir -p $(dirname "$f2")
function clean_up () {
    ARG=$?
    rm -rf "$TMP"
    exit $ARG
}
trap clean_up SIGINT SIGTERM EXIT
forge inspect $1 mi > "$f1"
forge inspect $2 mi > "$f2"
git diff --no-index "$f1" "$f2" --exit-code && echo "✅ Method interfaces are identical" || (echo "❌ Method interfaces are different" >&2 && exit 1)
