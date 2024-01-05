#!/usr/bin/env bash

# This script is checks to that all our Docker images are pinned to a specific SHA256 hash
#
# References as to why...
#   - https://nickjanetakis.com/blog/docker-tip-18-please-pin-your-docker-image-versions
#   - https://snyk.io/blog/10-docker-image-security-best-practices/ (Specifically: USE FIXED TAGS FOR IMMUTABILITY)
#
# Explanation of regex ignore choices
#   - We ignore sha256 because it suggests that the image dep is pinned
#   - We ignore scratch because it's literally the docker base image
#   - We ignore solana AS (builder|ci_tests) because it's a relative reference to another FROM call
#   - We ignore cosmwasm_artifacts AS artifacts because it's a local reference only, is built in tilt
#   - We ignore base AS (ignite-go-build|ignite-vue-build) because the base image is already pinned in wormchain/Dockerfile.proto
#
git ls-files -z | grep -z "Dockerfile*" | xargs -r -0 grep -s "FROM" | egrep -v 'sha256|scratch|solana|aptos|sui|base|cosmwasm_artifacts|cli-gen|const-gen AS (application|base|builder|ci_tests|tests|artifacts|ignite-go-build|ignite-vue-build|cli-export|const-export)'
if [ $? -eq 0 ]; then
   echo "[!] Unpinned docker files" >&2
   exit 1
else
   echo "[+] No unpinned docker files"
fi
