#!/bin/sh
set -e
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' guardian:6060/readyz)" != "200" ]]; do sleep 5; done
INTEGRATION=true go test -v .
