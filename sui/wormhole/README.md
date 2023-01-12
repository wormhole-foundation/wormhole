# Sui Wormhole Core Bridge Design

## State
State is created once and only once in `init` (a reserved keyword). It is initially returned to sender as an owned object. The admin/sender should then call `init_and_share_state` to initialize it with the proper arguments and share it for others to access.

## Child Objects
The rationale behind using child objects, and attaching them to State (the parent object), is that the alternative of direct wrapping can lead
to large objects, which require higher gas fees in transactions. Child objects also make it easy to store a collection of hetergeneous types in one place. In addition, if we instead wrapped an object (e.g. guardian set) inside of State, the object cannot be directly used in a transaction or queried by its ID.

## Epoch Timestamp
Sui currently does have fine-grained timestamps, so we use `tx_context::epoch(ctx)` in place of on-chain time in seconds.