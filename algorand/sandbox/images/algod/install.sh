#!/usr/bin/env bash

# Script to install algod in all sorts of different ways.
#
# Parameters:
#    -d    : Location where binaries will be installed.
#    -c    : Channel to install. Mutually exclusive with source options.
#    -u    : Git repository URL. Mutually exclusive with -c.
#    -b    : Git branch. Mutually exclusive with -c.
#    -s    : (optional) Git Commit SHA hash. Mutually exclusive with -c.

set -e

rootdir=`dirname $0`
pushd $rootdir

BINDIR=""
CHANNEL=""
URL=""
BRANCH=""
SHA=""

while getopts "d:c:u:b:s:" opt; do
  case "$opt" in
    d) BINDIR=$OPTARG; ;;
    c) CHANNEL=$OPTARG; ;;
    u) URL=$OPTARG; ;;
    b) BRANCH=$OPTARG; ;;
    s) SHA=$OPTARG; ;;
  esac
done

if [ -z BINDIR ]; then
  echo "-d <bindir> is required."
  exit 1
fi

if [ ! -z $CHANNEL ] && [ ! -z $BRANCH ]; then
  echo "Set only one of -c <channel> or -b <branch>"
  exit 1
fi

if [ ! -z $BRANCH ] && [ -z $URL ]; then
  echo "If using -b <branch>, must also set -u <git url>"
  exit 1
fi

echo "Installing algod with options:"
echo "  BINDIR = ${BINDIR}"
echo "  CHANNEL = ${CHANNEL}"
echo "  URL = ${URL}"
echo "  BRANCH = ${BRANCH}"
echo "  SHA = ${SHA}"

if [ ! -z $CHANNEL ]; then
  ./update.sh -i -c $CHANNEL -p $BINDIR -d $BINDIR/data -n
  exit 0
fi

if [ ! -z $BRANCH ]; then
    git clone --single-branch --branch "${BRANCH}" "${URL}"
    cd go-algorand
    if [ "${SHA}" != "" ]; then
      echo "Checking out ${SHA}"
      git checkout "${SHA}"
    fi

    git log -n 5

    ./scripts/configure_dev.sh
    make build
    ./scripts/dev_install.sh -p $BINDIR
fi

$BINDIR/algod -v
