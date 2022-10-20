#!/usr/bin/env bash

# This script checks to ensure that all our NPM packages have an appropriate scope.
#
git ls-files | grep "package.json" | xargs grep -s "\"name\":" | egrep -v '@certusone/|@wormhole-foundation/'
if [ $? -eq 0 ]; then
   echo "[!] Unscoped npm packages" >&2
   exit 1
else
   echo "[+] No unscoped npm packages"
fi
