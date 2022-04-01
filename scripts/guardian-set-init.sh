#!/usr/bin/env bash
# This script allows devnet initalization with more than one guardian.
# First argument is the number of guardians for the initial guardian set.
set -e

numGuardians=$1
echo "number of guardians to initialize: ${numGuardians}"

addressesJson="./scripts/devnet-consts.json"


# create an array of strings containing the ECDSA public keys of the devnet guardians in the guardianset

# guardiansPublicEth has the leading "0x" that Eth scripts expect.
guardiansPublicEth=$(jq -c --argjson lastIndex $numGuardians '.devnetGuardians[:$lastIndex] | [.[].public]' $addressesJson)
# guardiansPublicHex does not have a leading "0x", just hex strings.
guardiansPublicHex=$(jq -c --argjson lastIndex $numGuardians '.devnetGuardians[:$lastIndex] | [.[].public[2:]]' $addressesJson)


# copy the eth defaults to a new file so we can override just the things we need
cp ./ethereum/.env.test ./ethereum/.env

# override the default INIT_SIGNERS with the list created above
sed -i "/INIT_SIGNERS=/c\INIT_SIGNERS=$guardiansPublicEth" ./ethereum/.env

# create a local .env file, to be used by solana & terra
echo "INIT_SIGNERS=$guardiansPublicHex" > ./scripts/.env



# create an array of strings containing the private keys of the devnet guardians in the guardianset
guardiansPrivate=$(jq -c --argjson lastIndex $numGuardians '.devnetGuardians[:$lastIndex] | [.[].private]' $addressesJson)

# create a CSV string with the private keys of the guardians in the guardianset, that will be used to create registration VAAs
guardiansPrivateCSV=$( echo ${guardiansPrivate} | jq --raw-output -c  '. | join(",")')

echo "getting contract addresses from $addressesJson"
# get addresses from the constants file
solTokenBridge=$(jq --raw-output '.chains."1".contracts.tokenBridgeEmitterAddress' $addressesJson)
ethTokenBridge=$(jq --raw-output '.chains."2".contracts.tokenBridgeEmitterAddress' $addressesJson)
terraTokenBridge=$(jq --raw-output '.chains."3".contracts.tokenBridgeEmitterAddress' $addressesJson)
bscTokenBridge=$(jq --raw-output '.chains."4".contracts.tokenBridgeEmitterAddress' $addressesJson)

solNFTBridge=$(jq --raw-output '.chains."1".contracts.nftBridgeEmitterAddress' $addressesJson)
ethNFTBridge=$(jq --raw-output '.chains."2".contracts.nftBridgeEmitterAddress' $addressesJson)
terraNFTBridge=$(jq --raw-output '.chains."3".contracts.nftBridgeEmitterAddress' $addressesJson)

# generate the registration VAAs

# fetch dependencies for the clients/token_bridge script that generates token bridge registration VAAs
if [[ ! -d ./clients/token_bridge/node_modules ]]; then
    echo "going to install node modules in clients/token_bridge"
    npm ci --prefix clients/token_bridge && npm run build --prefix clients/token_bridge
fi
# create token bridge registration VAAs
echo "generating VAAs for token bridges"
solTokenBridgeVAA=$(npm --prefix clients/token_bridge run --silent main -- generate_register_chain_vaa 1 0x${solTokenBridge} --guardian_secret ${guardiansPrivateCSV})
ethTokenBridgeVAA=$(npm --prefix clients/token_bridge run --silent main -- generate_register_chain_vaa 2 0x${ethTokenBridge} --guardian_secret ${guardiansPrivateCSV} )
terraTokenBridgeVAA=$(npm --prefix clients/token_bridge run --silent main -- generate_register_chain_vaa 3 0x${terraTokenBridge} --guardian_secret ${guardiansPrivateCSV})
bscTokenBridgeVAA=$(npm --prefix clients/token_bridge run --silent main -- generate_register_chain_vaa 4 0x${bscTokenBridge} --guardian_secret ${guardiansPrivateCSV})


# fetch dependencies for the clients/nft_bridge script that generates nft bridge registration VAAs
if [[ ! -d ./clients/nft_bridge/node_modules ]]; then
    echo "going to install node modules in clients/nft_bridge"
    npm ci --prefix clients/nft_bridge && npm run build --prefix clients/nft_bridge
fi
# create nft bridge registration VAAs
echo "generating VAAs for nft bridges"
solNFTBridgeVAA=$(npm --prefix clients/nft_bridge run --silent main -- generate_register_chain_vaa 1 0x${solNFTBridge} --guardian_secret ${guardiansPrivateCSV})
ethNFTBridgeVAA=$(npm --prefix clients/nft_bridge run --silent main -- generate_register_chain_vaa 2 0x${ethNFTBridge} --guardian_secret ${guardiansPrivateCSV})
terraNFTBridgeVAA=$(npm --prefix clients/nft_bridge run --silent main -- generate_register_chain_vaa 3 0x${terraNFTBridge} --guardian_secret ${guardiansPrivateCSV})


# write the registration VAAs to env files
echo "updating .env files"

# solana token bridge
sed -i "/REGISTER_SOL_TOKEN_BRIDGE_VAA=/c\REGISTER_SOL_TOKEN_BRIDGE_VAA=$solTokenBridgeVAA" ./ethereum/.env
echo "REGISTER_SOL_TOKEN_BRIDGE_VAA=$solTokenBridgeVAA" >> ./scripts/.env
# solana nft bridge
sed -i "/REGISTER_SOL_NFT_BRIDGE_VAA=/c\REGISTER_SOL_NFT_BRIDGE_VAA=$solNFTBridgeVAA" ./ethereum/.env
echo "REGISTER_SOL_NFT_BRIDGE_VAA=$solNFTBridgeVAA" >> ./scripts/.env


# ethereum token bridge
sed -i "/REGISTER_ETH_TOKEN_BRIDGE_VAA=/c\REGISTER_ETH_TOKEN_BRIDGE_VAA=$ethTokenBridgeVAA" ./ethereum/.env
echo "REGISTER_ETH_TOKEN_BRIDGE_VAA=$ethTokenBridgeVAA" >> ./scripts/.env
# ethereum nft bridge
sed -i "/REGISTER_ETH_NFT_BRIDGE_VAA=/c\REGISTER_ETH_NFT_BRIDGE_VAA=$ethNFTBridgeVAA" ./ethereum/.env
echo "REGISTER_ETH_NFT_BRIDGE_VAA=$ethNFTBridgeVAA" >> ./scripts/.env


# terra token bridge
sed -i "/REGISTER_TERRA_TOKEN_BRIDGE_VAA=/c\REGISTER_TERRA_TOKEN_BRIDGE_VAA=$terraTokenBridgeVAA" ./ethereum/.env
echo "REGISTER_TERRA_TOKEN_BRIDGE_VAA=$terraTokenBridgeVAA" >> ./scripts/.env
# terra nft bridge
sed -i "/REGISTER_TERRA_NFT_BRIDGE_VAA=/c\REGISTER_TERRA_NFT_BRIDGE_VAA=$terraNFTBridgeVAA" ./ethereum/.env
echo "REGISTER_TERRA_NFT_BRIDGE_VAA=$terraNFTBridgeVAA" >> ./scripts/.env


# bsc token bridge
sed -i "/REGISTER_BSC_TOKEN_BRIDGE_VAA=/c\REGISTER_BSC_TOKEN_BRIDGE_VAA=$bscTokenBridgeVAA" ./ethereum/.env
echo "REGISTER_BSC_TOKEN_BRIDGE_VAA=$bscTokenBridgeVAA" >> ./scripts/.env


# copy the local .env file to the solana & terra dirs
cp ./scripts/.env ./solana
cp ./scripts/.env ./terra/tools


echo "guardian set init complete!"
