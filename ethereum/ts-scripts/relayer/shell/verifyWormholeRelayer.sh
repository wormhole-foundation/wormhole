# For Celo, use '--show-standard-json-input' and submit manually online
# To link Proxy and Implementation, go to the proxyContractChecker of the chain's etherscan

# Make sure the ETHERSCAN_API_KEY is set
# and the WORMHOLE_ADDRESS is set to the 32-byte wormhole address (no 0x)

# for testnet, remove 'FOUNDRY_PROFILE=production' and make the addresses correct (current addresses are hardcoded for mainnet - the proxy address, constructor args for proxy, and the implementation address)
# note: the first 5 testnets (avalanche, celo, bsc, mumbai, moonbeam) were deployed with evm_version London

FOUNDRY_PROFILE=production forge verify-contract 0x27428DD2d3DD32A4D7f7C497eAaa23130d894911 contracts/relayer/create2Factory/Create2Factory.sol:SimpleProxy --chain-id $CHAIN_ID --watch --constructor-args 00000000000000000000000025688636cec6ce0f1434b1e7dd0a223f3f258336 \
&& FOUNDRY_PROFILE=production forge verify-contract 0x00337a31aEE3Ed37f5D5FBF892031d0090Da2EeF WormholeRelayer --chain-id $CHAIN_ID --watch --constructor-args $WORMHOLE_ADDRESS