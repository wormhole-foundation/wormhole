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
