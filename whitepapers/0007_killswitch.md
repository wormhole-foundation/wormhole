# Token/NFT Bridge Shutdown (aka: "KillSwitch") [DRAFT]

[TOC]

## Objective

To enable/disable send and receive transactions on the Wormhole Token Bridge and NFT Bridge.

## Background

Wormhole facilitates value exchange cross-chain via the Token Bridge and the NFT bridge.  Given value exchange using smart contracts and accepting that security bugs may exist in the future, Guardians may need to take quick and decisive action to enable/disable transactions without a full 2/3s governance vote to ensure the integrity of the value stored within the bridge.

## Goals

Implement a safeguard that facilitate the enabling or disabling of Token Bridge and NFT Bridge send/receive functionality in the event of an existential threat.

* Enable/Disable Token bridge transactions
* Enable/Disable NFT bridge transactions
* Effectiveness of the control acheived in minutes of defined need (instead of hours/days)
* Provide safety while security patches are public, but not yet effective while waiting for governance ceremony and contract upgrades to complete

## Non-Goals

* Solving for collateralization assets
* Solving for transaction thresholding/volume limits
* Solving at the core layer
* Solving for wallet segmentation or hot/cold wallets
* Solving for 2/3+ Guardian Consensus

## Overview

The shutdown functionality aims to extend existing smart contracts that enable Token and NFT bridge to respond to a smaller set of Guardian votes (eg. 5 instead of 13) that would enable/disable send and receive functionality during an existential threat scenario at a speed greater than could be achieved with a 2/3+ majority vote from the full guardian set of 19.

During an existential threat scenario where any Guardian believes there is a need to disable send/receive transactions on the Token/NFT Bridge they would send a message to the relevant smart contract(s) to indicate their vote to disable send/receive transactions.  Once these bridge smart contracts receive a set of votes (eg 5 votes) to shut down then the contract will no longer perform transactions.  Similarly, if the guardian votes lose a majority (eg less than 5) the contract will begin processing transactions again allowing for shorter downtime windows.

## Design Expectations

Contracts will only ever trust the Guardians to perform shutdown votes.

There will be a `votes` structure on each contract, this structure will retain the votes of any guardians who wishes to shutdown.  Each vote should contain `SignatureID` of the Guardian who voted to shutdown.  Guardians will be able to submit their shutdown vote or revoke it at any time without a Govenernance ceremony.  `votes` should be a set of Guardian votes and should only contain one vote status per Guardian, such that a single Guardian cannot vote more than once.

A Guardian would send a `signed_message` to the `shutdown` function, which would need to include the following:
- A rolling uint32 `nonce` to prevent replay attacks
- A bool `vote` to indicate intent to halt transactions

The `shutdown` function would need to perform the following:
* Receive the `signed_message`
* Verify that the `signatureID` is a valid Guardian
* Verify that the signed `nonce` is equal to the current `nonce`
* Set `vote` for `signatureID` within `votes` structure
* Increment `nonce` by 1 (this serves as replay detection)

For each send/receive capability for a token/nft bridge, it must consult the `votes` structure to determine whether transactions can flow.  If there are a certian number of votes to shutdown in `votes` transactions must be halted.

If at any point there was an abuse of this shutdown capability causing a transaction denial of service, the expected recourse would be that the Guardians would perform a governance ceremony and upgrade the contracts to patch the contracts directly.  This would be slow, but it is effectively the same speed in concept to what an effective shutdown behavior would look like if the shutdown feature did not exist.

### Pros

* Allows any subset of N (eg. 5) Guardians to stop value transfer in the face of an existential threat.
* Adds a layer of maintenance safety when Guardians need to publicize an upgrade, obtain governance, and upgrade contracts.
### Cons

* Will require CLI tooling or manual steps for a signer to obtain the current nonce before signing a message to ensure speed
* If for some reason the nonce rolls over and becomes non-unique, it could be succeptable to replay attacks that the nonce aims to solve for
* Will require each signer to wait until the next block before signing