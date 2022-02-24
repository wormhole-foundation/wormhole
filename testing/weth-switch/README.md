1. stand up a minimal devnet
2. manually run `cd ethereum/ && npx truffle exec scripts/deploy_test_token.js` to get a second WETH
3. update the BridgeImplementation.sol with the address
4. remove all migrations except `3_deploy_bridge` and comment out everything except the implementation and run `cd ethereum/ && npx truffle migrate`
5. create and sign a governance to upgrade the TokenBridge to the new implementation contract address - e.g. `cd clients/token_bridge/ && npm start -- generate_upgrade_chain_vaa 2 0x000000000000000000000000dAA71FBBA28C946258DD3d5FcC9001401f72270F`
6. update `test.js` and run it with `node test.js`
7. do some transfers in the ui to ensure that everything works as expected
