#/bin/bash

pgrep tilt > /dev/null
if [ $? -ne 0 ]; then
    echo "tilt is not running"
    exit 1;
fi

## tilt's rpcs
ETH_DEVNET="http://localhost:8545"
BSC_DEVNET="http://localhost:8546"

## first account's private key
PRIVATE_KEY="0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"

## deploy to tilt (need --legacy because ganache in Tilt does not use eip-1559)
echo "deploying to eth devnet"
forge script forge-scripts/deploy_contracts.sol \
    --rpc-url $ETH_DEVNET \
    --private-key $PRIVATE_KEY \
    --interactives 1 \
    --sig "run(address wormholeAddress)" \
    "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"\
    -vvv \
    --broadcast --slow --legacy 

echo "deploying to bsc devnet"
forge script forge-scripts/deploy_contracts.sol \
    --rpc-url $BSC_DEVNET \
    --private-key $PRIVATE_KEY \
    --interactives 1 \
    --sig "run(address wormholeAddress)" \
    "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550" \
    -vvv \
    --broadcast --slow --legacy 
