#!/bin/bash

set -x

cd aptos-core/
target/debug/aptos-node --test
