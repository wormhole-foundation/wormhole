#!/usr/bin/env bash
set -euo pipefail

# load the keys
hermes keys add --key-name keyterra --hd-path "m/44'/330'/0'/0/0" --chain localterra --mnemonic-file mnemonic_file.txt
hermes keys add --key-name keywormhole --chain wormchain --mnemonic-file mnemonic_file.txt

# create the IBC Generic Emission channel
hermes create channel \
--a-chain wormchain --b-chain localterra \
--a-port wasm.wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj \
--b-port wasm.terra1436kxs0w2es6xlqpp9rd35e3d0cjnw4sv8j3a7483sgks29jqwgsnyey7t \
--new-client-connection --channel-version ibc-wormhole-v1 --order unordered --yes

# create the ICS20 channel
hermes create channel \
--a-chain wormchain --b-chain localterra \
--a-port transfer --b-port transfer \
--new-client-connection --channel-version ics20-1 --order unordered --yes

# start the relayer
hermes start