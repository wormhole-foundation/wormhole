#!/bin/bash -f

sui client call --function init_and_share_state --module bridge_state --package 0x975030c700e4afe389b04a0b0627bc1797729189 --gas-budget 20000 --args 0x22f2c58912bc6674cd2ee4eeb41bf655024af623 0x0a398059600906accfb8a9b30c393b930cab360a
