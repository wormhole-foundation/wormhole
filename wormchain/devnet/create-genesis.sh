#!/usr/bin/env bash
set -euo pipefail

if [ -z "${NUM_GUARDIANS}" ]; then
    echo "Error: NUM_GUARDIANS is unset, cannot create wormchain genesis."
    exit 1
fi

pwd=$(pwd)
genesis="$pwd/devnet/base/config/genesis.json"

# TODO
# create a sequence of the wormchain instances to include
# loop through the sequence, reading the data from the instance's dir
# add the genesis account to:
#   app_state.auth.accounts
#   app_state.bank.balances
# add the gentx
# add the guardian pubkey base64 to wormhole.guardianSetList[0].keys
# add the validator obj to wormhole.guardianValidatorList


# TEMP manually add the second validator info to genesis.json
if [ $NUM_GUARDIANS -ge 2 ]; then
  echo "number of guardians is >= 2, adding second validator to genesis.json."
  # the validator info for wormchain-1
  guardianKey="iNfYsyqRBdIoEA5y3/4vrgcF0xw="
  validatorAddr="cBxHWxmj9o0/3r8JWRSH+s7y1jY="

  # add the validatorAddr to guardianSetList.keys.
  # use jq to add the object to the list in genesis.json. use cat and a sub-shell to send the output of jq to the json file.
  cat <<< $(jq --arg new "$guardianKey" '.app_state.wormhole.guardianSetList[0].keys += [$new]' ${genesis})  > ${genesis}

  # create a guardianValidator config object and add it to the guardianValidatorList.
  validatorConfig="{\"guardianKey\": \"$guardianKey\", \"validatorAddr\": \"$validatorAddr\"}"
  cat <<< $(jq --argjson new "$validatorConfig" '.app_state.wormhole.guardianValidatorList += [$new]' ${genesis})  > ${genesis}
fi



echo "done with genesis, exiting."
