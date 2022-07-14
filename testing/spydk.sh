#!/bin/sh
set -e
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' spy:6060/metrics)" != "200" ]]; do sleep 5; done
CI=true npm --prefix ../spydk/js run test-ci
