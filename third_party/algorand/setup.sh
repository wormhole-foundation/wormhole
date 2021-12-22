#!/bin/bash

dn="$(dirname "$0")"
cd $dn

pipenv install
pipenv run python3 setup.py
