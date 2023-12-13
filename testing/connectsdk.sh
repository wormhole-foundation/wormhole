#!/bin/sh
set -e
# TODO: No need to wait until we're doing actual integration testing
CI=true npm --prefix ../sdk/js-connect run test

