#!/usr/bin/env bash
# This script sorts the custom words in cspell-custom-words.txt alphabetically
# and includes duplicate words.
#
# Note: Run this script from the root of the project.
sort cspell-custom-words.txt -o cspell-custom-words.txt -f