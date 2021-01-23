#!/usr/bin/env bash
# This script copies package{-lock}.json from a running container.
set -e

kubectl cp -c ganache eth-devnet-0:package.json package.json
kubectl cp -c ganache eth-devnet-0:package-lock.json package-lock.json
