# PublishMsg Contract
This is a simple contract that can be used to generate wormhole messages in testnet. It can be used to test the guardian watcher.

## Deployment
1. Make sure the chain is populated in truffle-config.js.
2. npm ci
3. npm run build
4. MNEMONIC="" TESTNET_WORMHOLE_CORE_ADDRESS="0x6b9C8671cdDC8dEab9c719bB87cBd3e782bA6a35" npm run truffle -- exec scripts/deploy_publish_msg.js --network neon_testnet
5. Make note of the value returned as "PublishMsg address", it will be used as PUBLISH_MSG_ADDRESS to run the tool.

## Generating a test message
1. MNEMONIC="" TESTNET_RPC="https://proxy.devnet.neonlabs.org/solana" PUBLISH_MSG_ADDRESS="0x95F81502b6DafbF1aAdac5400814f976d119B520" CONSISTENCY_LEVEL=200 node src/publishMsg.js