1. bring up local net using ```aptos node run-local-testnet --with-faucet --force-restart```
2. initialize account using ```aptos account fund-with-faucet --account 277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b```
3. publish modules using ```aptos move publish --private-key 0x537c1f91e56891445b491068f519b705f8c0f1a1e66111816dd5d4aa85b8113d --profile default```
4. run ```init_wormhole.ts```
5. run ```publish_wormhole_message.ts```