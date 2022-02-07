#!/bin/sh
npm --prefix ../sdk/js run test-ci
npm --prefix ../spydk/js run test-ci 
npm --prefix ../bridge_ui run test 