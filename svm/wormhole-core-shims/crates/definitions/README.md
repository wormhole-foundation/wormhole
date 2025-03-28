# Wormhole SVM Definitions

Definitions relating to Wormhole SVM programs. These definitions include finding
PDA addresses (and corresponding bump seeds), various consts (like program IDs),
and other things that define these program accounts and data.

## Cargo Features

There are features that define network types and specific SVM networks.

### Network Types

The default network type is mainnet. There is no feature that defines mainnet.
But if one of the following features are defined, program IDs and account
addresses will not use the ones defined for mainnet.

- `localnet`: Wormhole's Tilt devnet. Programs like the Wormhole Core Bridge and
  its associated PDAs have addresses specific to this local development network.
- `testnet`: Public devnet or testnet depending on the specific SVM network. For
  Solana specifically, this feature corresponds to the public Solana devnet.

### Specific Networks

There are no default network features. These feature labels also exist as
submodules in this crate. By defining a particular SVM network feature, the
definitions found in this submodule are simply exported into the crate root.

- `solana`

### Other Features

- `borsh`: Accounts and events relating to Wormhole SVM programs that follow
  Borsh serialization. This feature also supports deserializing data with
  discriminators.