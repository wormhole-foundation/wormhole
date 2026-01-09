#!/usr/bin/env bash

ctx=$(mktemp --directory)
docker build --tag tls-gen --file ./tls.Dockerfile --progress=plain "$ctx"
rm -rf "$ctx"