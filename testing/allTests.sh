#!/bin/sh
CI=true npm --prefix ../sdk/js run test-ci
CI=true npm --prefix ../spydk/js run test-ci 
CI=true npm --prefix ../bridge_ui run test 