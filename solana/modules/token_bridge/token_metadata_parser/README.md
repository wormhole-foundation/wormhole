# What is this

This crate is a 0-dependency parser for the token2022 [metadata pointer and metadata extensions](https://solana.com/developers/courses/token-extensions/token-extensions-metadata).

# Why

In order for the [token bridge program](../program) to support token2022 metadata, it needs to be able to parse the extensions out of the mint account.
The official parser is implemented in the [spl-token-2022](https://crates.io/crates/spl-token-2022) crate. That crate has a *massive* dependency tree, and is incompatible with the token bridge's dependencies.
Resolving the dependency issues would require upgrading the token bridge dependencies beyond several major versions, which is risky. Instead, we re-implement the parsing logic from scratch without any external dependencies.

# Why is it excluded from the workspace

In order to verify our parser works, we construct real token2022 mint accounts with the `spl-token-2022` crate. As such, this crate has a *dev* dependency on that crate. If it were included in the workspace, cargo would try to reconcile the dev dependencies with the regular dependencies of the other crates, which would defeat the purpose.
