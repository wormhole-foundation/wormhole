# 3. Cosmos SDK v0.47 upgrade will be implemented in Wormhole Fork

Date: 2024-07-10

## Status

Accepted

## Context

The Wormhole Foundation has made a copy of the Cosmos SDK repository, which can be found in their Github organization [here](https://github.com/wormhole-foundation/cosmos-sdk). They are referencing this copied repository in the wormchain [go.mod](https://github.com/wormhole-foundation/wormhole/blob/6236a9a6cbd0dc00a940e6654c6f6106d0904ece/wormchain/go.mod#L142) file, referencing the [v0.45.9-wormhole-2](https://github.com/wormhole-foundation/cosmos-sdk/releases/tag/v0.45.9-wormhole-2) tag. This tag has [commits](https://github.com/wormhole-foundation/cosmos-sdk/commits/v0.45.9-wormhole-2/?since=2022-10-19&until=2022-12-21) made by the wormhole-foundation team that fundamentally change the behavior of the staking module, particularly implementing proof of authority based on the wormchain guardian set.

## Decision

With the use of a forked Cosmos SDK and fundamental changes to the staking module, the initial Cosmos SDK v0.45 to v0.47 upgrade will be done on their fork in the following manner:

1. Pull in the [v0.47.12](https://github.com/cosmos/cosmos-sdk/releases/tag/v0.47.12) (latest in the v0.47 series) tag from the Cosmos SDK repository into the 
2. Re-implement the changes made in the v0.45.9-wormhole-2 tag into the v0.47.12 tag.
3. Release a new tag that includes the changes from the v0.47.12 tag and the changes made on top of it from the v0.45.9-wormhole-2 tag.
4. Reference this new version in the wormchain go.mod file.

## Consequences

With the changes to the staking module being applied directly in the Wormhole Foundation Cosmos SDK fork, this will have the following good and bad consequences:

1. It will maintain the required changes made to the staking module as they were at the time of the v0.45.9-wormhole-2 tag.
2. It will be easier to maintain the changes made to the staking module as they will be directly applied to the forked repository.
3. Maintaining the fork of the Cosmos SDK will be more difficult as the Cosmos SDK repository will continue to evolve.
