#!/bin/sh
set -e
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' guardian:6060/readyz)" != "200" ]]; do sleep 5; echo "Waiting for guardian to exist"; done
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' spy:6060/metrics)" != "200" ]] do sleep 5; echo "Waiting for spy to exist"; done
CI=true npm --prefix ../spydk/js run test-ci
