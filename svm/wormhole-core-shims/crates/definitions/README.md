# Wormhole SVM Definitions

Definitions relating to Wormhole SVM programs. These definitions include finding
PDA addresses (and corresponding bump seeds), various consts (like program IDs),
and other things that define these program accounts and data.

The crate is parameterized over a set of program IDs (the Wormhole program, the
post-vaa shim program, and the verify-vaa shim program). Since this crate may be
used in many different scenarios, and many different SVM networks, we expose
feature flags to control which program IDs are used to derive the other constants.

For Solana, we have a hardcoded set of addresses for `mainnet`, `devnet` and a
local testing environment. If the `solana` feature is specified, it will default
to the mainnet addresses. Otherwise, the `testnet` flag provides the addresses
for Solana Devnet, and the `localnet` flag provides the addresses for the local
testing environment.

Alternatively, when using on another SVM chain (or on Solana, but against an
independent Wormhole deployment) just specify the `from-env` feature flag (and
don't specify Solana, even if the chain is Solana). In this case, the following
4 environment variables are needed:
- `CHAIN_ID`: the Wormhole ID of the chain deployed on. e.g. Solana is 1, Fogo is 51.
- `BRIDGE_ADDRESS`: program ID of the Wormhole program
- `POST_MESSAGE_SHIM_PROGRAM_ID`: program ID of the [../../programs/post-message/](post-message shim).
- `VERIFY_VAA_SHIM_PROGRAM_ID`: program ID of the [../../programs/verify-vaa](verify-vaa shim).

The definitions crate can be compiled without either the `solana` or the
`from-env` feature flags. In this case, it will not expose any addresses in the top-level crate, for example `crate::CORE_BRIDGE_FEE_COLLECTOR` won't be available. However, the predefined Solana addresses are still available via their qualified paths:
e.g. `crate::solana::devnet::CORE_BRIDGE_FEE_COLLECTOR`.

A crate wishing to build on top of this crate should pass through the
`from-env`, `solana`, `testnet`, and `devnet` flags, but may wish to specify
`from-env` as the default.

### Other Features

- `borsh`: Accounts and events relating to Wormhole SVM programs that follow
  Borsh serialization. This feature also supports deserializing data with
  discriminators.
