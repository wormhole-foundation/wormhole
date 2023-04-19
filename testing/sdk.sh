#!/bin/bash
set -ex
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' guardian:6060/readyz)" != "200" ]]; do sleep 5; done
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' spy:6060/metrics)" != "200" ]]; do sleep 5; done
echo "guardian and spy are ready, running tests"
CI=true npm --prefix ../sdk/js run test-ci
CI=true npm --prefix ../ethereum run relayer-ci-test
