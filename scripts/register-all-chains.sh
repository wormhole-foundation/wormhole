rm -rf my_proposal

./register-chain-governance.sh -m TokenBridge -c solana -o my_proposal > governance.md
./register-chain-governance.sh -m TokenBridge -c ethereum -o my_proposal > governance.md
./register-chain-governance.sh -m TokenBridge -c terra -o my_proposal > governance.md
./register-chain-governance.sh -m TokenBridge -c bsc -o my_proposal > governance.md
./register-chain-governance.sh -m TokenBridge -c polygon -o my_proposal > governance.md
./register-chain-governance.sh -m TokenBridge -c avalanche -o my_proposal > governance.md
./register-chain-governance.sh -m TokenBridge -c oasis -o my_proposal > governance.md
./register-chain-governance.sh -m TokenBridge -c fantom -o my_proposal > governance.md
./register-chain-governance.sh -m TokenBridge -c aurora -o my_proposal > governance.md

# These are already on the current guardian set.
# ./register-chain-governance.sh -m TokenBridge -c karura -o my_proposal > governance.md
# ./register-chain-governance.sh -m TokenBridge -c klaytn -o my_proposal > governance.md
# ./register-chain-governance.sh -m TokenBridge -c celo -o my_proposal > governance.md
# ./register-chain-governance.sh -m TokenBridge -c acala -o my_proposal > governance.md
# ./register-chain-governance.sh -m TokenBridge -c terra2 -o my_proposal > governance.md
# ./register-chain-governance.sh -m TokenBridge -c algorand -o my_proposal > governance.md
# ./register-chain-governance.sh -m TokenBridge -c near -o my_proposal > governance.md

./register-chain-governance.sh -m NFTBridge -c solana -o my_proposal > governance.md
./register-chain-governance.sh -m NFTBridge -c ethereum -o my_proposal > governance.md
./register-chain-governance.sh -m NFTBridge -c bsc -o my_proposal > governance.md
./register-chain-governance.sh -m NFTBridge -c polygon -o my_proposal > governance.md
./register-chain-governance.sh -m NFTBridge -c avalanche -o my_proposal > governance.md
./register-chain-governance.sh -m NFTBridge -c oasis -o my_proposal > governance.md
./register-chain-governance.sh -m NFTBridge -c fantom -o my_proposal > governance.md
./register-chain-governance.sh -m NFTBridge -c aurora -o my_proposal > governance.md

# These are already on the current guardian set.
# ./register-chain-governance.sh -m NFTBridge -c karura -o my_proposal > governance.md
# ./register-chain-governance.sh -m NFTBridge -c klaytn -o my_proposal > governance.md
# ./register-chain-governance.sh -m NFTBridge -c celo -o my_proposal > governance.md
# ./register-chain-governance.sh -m NFTBridge -c acala -o my_proposal > governance.md