#!/bin/bash

dn="$(dirname "$0")"
cd $dn

apt-get install -y python3-pip
pip install -r requirements.txt
python3 setup.py
