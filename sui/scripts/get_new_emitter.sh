#!/bin/bash -f

. env.sh

sui client call --function get_new_emitter --module wormhole --package 0xae77cb8d8dd113d539123ee9bccc18a05e1d16d1  --gas-budget 20000 --args 0xf4c56900b4d4e41e519cbcc5ee4a750369446adc
