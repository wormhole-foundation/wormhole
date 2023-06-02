#!/bin/sh
set -e
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' guardian:6060/readyz)" != "200" ]]; do sleep 5; done
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' spy:6060/metrics)" != "200" ]]; do sleep 5; done
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' ibc-relayer:7597/debug/pprof/)" != "200" ]]; do sleep 5; done
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' relayer-engine:3000/metrics)" != "200" ]]; do sleep 5; done
CI=true npm --prefix ../sdk/js run test-ci
