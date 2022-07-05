#!/bin/bash

# This script is checks to that all our Docker images are pinned to a specific SHA256 hash
#
# References as to why...
#   - https://nickjanetakis.com/blog/docker-tip-18-please-pin-your-docker-image-versions
#   - https://snyk.io/blog/10-docker-image-security-best-practices/ (Specifically: USE FIXED TAGS FOR IMMUTABILITY)
#
find . -name 'Dockerfile*' -print0  -type f | xargs -0 grep -s "FROM" {} | egrep -v 'scratch|sha256|solana AS (builder|ci_tests)|node_module'
if [ $? -eq 0 ]; then
   echo "[!] Unpinned docker files" >&2
   exit 1
else
   echo "[+] No unpinned docker files"
fi