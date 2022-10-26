#!/bin/bash
./sandbox reset dev -v
account=$(./sandbox goal account list | awk '{print $2; exit}')
./sandbox goal clerk send --amount 3999000000000000 -f $account -t HL6A24OGJX4FDZT36HOQ6VWJDF6GW3IEWB4FXB4OH5FQKVI46HZBZOZFAM
