#!/bin/bash

set -x

echo "starting web server"

python3 -m http.server --directory /tmp/_sandbox 3031 &

echo "starting near sandbox"
rm -rf /tmp/_sandbox
nearcore/target/release/near-sandbox --home /tmp/_sandbox init
tmpfile="$(mktemp)"
jq '.store.max_open_files = 9000' /tmp/_sandbox/config.json > "$tmpfile"
cp "$tmpfile" /tmp/_sandbox/config.json
nearcore/target/release/near-sandbox --home /tmp/_sandbox run
