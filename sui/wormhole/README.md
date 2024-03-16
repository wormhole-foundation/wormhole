# Sui Wormhole Core Bridge Design

## State

The `State` object is created exactly once during the initialisation of the
contract. Normally, run-once functionality is implemented in the special `init`
function of a module (this code runs once, when the module is first deployed),
but this function takes no arguments, while our initialisation code does (to
ease deployment to different environments without recompiling the contract).

To allow configuring the state with arguments, it's initialised in the
`init_and_share_state` function, which also shares the state object. To ensure
this function can only be called once, it consumes a `DeployerCap` object
which in turn is created and transferred to the deployer in the `init` function.
Since `init_and_share_state` consumes this object, it won't be possible to call
it again.

## Dynamic fields

TODO: up to date notes on where and how we use dynamic fields.

## Epoch Timestamp

Sui currently does not have fine-grained timestamps, so we use
`tx_context::epoch(ctx)` in place of on-chain time in seconds.

# Source verification

Verify that the Move sources here match the package published on-chain by running
`sui client verify-source --verify-deps`

Make sure that `sui client active-env` is connected to a Sui RPC for the
network the contract is deployed to (`mainnet/testnet`).
