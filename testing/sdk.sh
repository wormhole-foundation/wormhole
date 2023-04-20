#!/bin/sh
set -e
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' guardian:6060/readyz)" != "200" ]]; do sleep 5; done
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' spy:6060/metrics)" != "200" ]]; do sleep 5; done
cd ..
CI=true npm run test-ci -w sdk/js
