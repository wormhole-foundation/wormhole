#!/bin/sh
exec npm --prefix ../sdk/js run test-ci ; echo "JS test results from sdk: $?"
exec npm --prefix ../spydk/js run test-ci ; echo "JS test results from spydk: $?" 
exec npm --prefix ../bridge_ui run test; echo "JS test results from bridge_ui: $?" 