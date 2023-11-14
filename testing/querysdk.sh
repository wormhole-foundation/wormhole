#!/bin/sh
set -e
num=${NUM_GUARDIANS:-1} # default value for NUM_GUARDIANS = 1
for ((i=0; i<num; i++)); do
    while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' guardian-$i.guardian:6060/readyz)" != "200" ]]; do sleep 5; done
done
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' query-server:6068/health)" != "200" ]]; do sleep 5; done
CI=true npm --prefix ../sdk/js-query run test-ci
