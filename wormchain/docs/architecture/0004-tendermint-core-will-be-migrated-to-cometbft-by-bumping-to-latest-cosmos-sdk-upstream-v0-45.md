# 4. Tendermint Core Will be Migrated to CometBFT by Bumping to Latest Cosmos SDK Upstream v0.45

Date: 2024-07-12

## Status

Accepted

## Context

The Wormhole Cosmos SDK was forked at [v0.45.9](https://github.com/cosmos/cosmos-sdk/releases/tag/v0.45.9) of the Cosmos SDK releases. This can be seen by looking at their [in-use tag's](https://github.com/wormhole-foundation/cosmos-sdk/commits/v0.45.9-wormhole-2/) last commit from the upstream [here](https://github.com/wormhole-foundation/cosmos-sdk/commit/2582f0aab7b2cbf66ade066fe570a4622cf0b098), which shares the commit history and SHA of the v0.45.9 release.

This version of the Cosmos SDK was released before the fork of Tendermint Core and migration provided by the Cosmos SDK core team in the [Comet BFT](https://github.com/cometbft/cometbft) project, the motivation of which was announced [here](https://informal.systems/blog/cosmos-meet-cometbft).

To facilitate a more modern usage of the Cosmos SDK, projects should move away from Tendermint Core to CometBFT, as it is more up-to-date, maintained and provides security and bug fixes.

The Cosmos SDK team slowly rolled out migrations from Tendermint Core to CometBFT in the Cosmos SDK repo, and this migration was implemented in the v0.45 line in release [v0.45.15](https://github.com/cosmos/cosmos-sdk/releases/tag/v0.45.15).

## Decision

The migration to Tendermint Core will take place by:

1. Pulling in the upstream latest version in the v0.45 line, which is [v0.45.16](https://github.com/cosmos/cosmos-sdk/releases/tag/v0.45.16) (one version after the v0.45.15 migration release)
2. Replaying the [commits](https://github.com/wormhole-foundation/cosmos-sdk/commits/v0.45.9-wormhole-2/?since=2022-10-19&until=2022-12-21) implemented by the Wormhole developers on top of this tag
3. Release a new tag that includes the changes from the v0.45.16 tag and the changes made on top of it from the v0.45.9-wormhole-2 tag.
4. Reference this new version in the wormchain go.mod file.

## Consequences

This change will have the following consequences:

1. It pulls in the latest bug fixes and security updates in the v0.45 line while work on moving to v0.47 continues (see ADR 3 for details)
2. It will require extensive testing and review to ensure the changes from v0.45.9 to v0.45.16 did not break the Wormchain repo's usage of the Cosmos SDK
