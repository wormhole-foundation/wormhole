#!/usr/bin/env bash

DOCKER_BUILDKIT=1 tilt docker build -- -f Dockerfile.lint .
