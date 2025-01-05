git clone, then run via `make test` to observe gas costs in forge test traces:

gas costs of `parseAndVerifyVM` (not including transaction overhead):
```
134,689 original
134,689 thirteen sigs (matches as expected)
133,601 using CoreBridgeLib from Solidity SDK which parses the vaa itself and calls verifyVM
 27,677 single signature
 88,686 guardian set from calldata (gscd) optimizations (does not agree with 83k number in the monorepo PR despite additional optimizations)
 73,646 optimized, backwards compatible implementation
 14,075 threshold signature (i.e. single address) optimized version
```

Original gas costs also match gas used field of Action[3] in [Etherscan Parity trace of the sample transaction](https://etherscan.io/vmtrace?txhash=0xedd3ac96bc37961cce21a33fd50449dba257737c168006b40aa65496aaf92449&type=parity).

[guardian set from calldata monorepo PR](https://github.com/wormhole-foundation/wormhole/pull/3366) that passes the guardian set as calldata and only checks the hash - its README is the source of the 83k number above
