#/bin/bash

## tilt's rpcs
FUJI="https://api.avax-test.network/ext/bc/C/rpc"
CELO="https://alfajores-forno.celo-testnet.org"


echo "deploying to fuji devnet"
forge script forge-scripts/deploy_contracts.sol \
   --rpc-url $FUJI \
   --private-key $PRIVATE_KEY \
  --interactives 1 \
  --sig "run(address wormholeAddress)" \
   "0x7bbcE28e64B3F8b84d876Ab298393c38ad7aac4C"\
   -vvv \
   --broadcast --slow --legacy 

# echo "deploying to celo devnet"
# forge script forge-scripts/deploy_contracts.sol \
#     --rpc-url $CELO \
#     --private-key $PRIVATE_KEY \
#     --interactives 1 \
#     --sig "run(address wormholeAddress)" \
#     "0x88505117CA88e7dd2eC6EA1E13f0948db2D50D56" \
#     -vvv \
#     --broadcast --slow --legacy 
