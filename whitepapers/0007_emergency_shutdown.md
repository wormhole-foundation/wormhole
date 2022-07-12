# Token/NFT Bridge Shutdown (aka: "Emergency Shutdown") [DRAFT]

[TOC]

## Objective

To enable/disable send and receive transactions on the Wormhole Token Bridge and NFT Bridge.

## Background

Wormhole facilitates value exchange cross-chain via the Token Bridge and the NFT bridge.  Given value exchange using smart contracts and accepting that security bugs may exist in the future, Guardians may need to take an action to enable/disable transactions to ensure the integrity of the value stored within the bridge.
## Goals

Implement a safeguard that facilitate the enabling or disabling of Token Bridge and NFT Bridge send/receive functionality in the event of an existential threat.

* Enable/Disable Token bridge transactions
* Enable/Disable NFT bridge transactions
* Provide reactive safety for a direct smart contract vulnerability that allows asset movement
* Provide proactive safety when we need to ship an existential security bug in the token/nft bridges while waiting for consensus
## Non-Goals

* Solving for collateralization assets
* Solving for transaction thresholding/volume limits
* Solving at the core layer
* Solving for wallet segmentation or hot/cold wallets
* Solving for 24/7 monitoring to trigger the control
## Overview

The shutdown functionality aims to extend existing wormhole smart contracts that enable Token and NFT bridge functionality to enable/disable send and receive capability during an existential threat scenario.

During an existential threat scenario where Guardians believe there is a need to disable send/receive transactions on the Token/NFT Bridge they would send a message to the relevant smart contract(s) to indicate their vote to disable send/receive transactions.  Once these smart contracts receive a quorum (2/3rd+) set of votes to shutdown then the contracts will no longer perform transactions.  Similarly, if the guardian votes lose a super minority the contract will begin processing transactions again allowing for a quicker return to service.

## Design Expectations

Contracts will only ever trust the Guardians to perform shutdown votes.

There will be a `votes` structure on each contract, this structure will retain the votes of any guardians who wishes to shutdown.  Each vote should contain an `authProof` of the Guardian who voted to shutdown from their wallet address.  Guardians will be able to submit their shutdown vote up or down it at any time without an additional Govenernance ceremony.  `votes` should be a set of Guardian votes and should only contain one vote status per Guardian, such that a single Guardian cannot vote more than once.

A Guardian would send an `authProof` to the `castStartupVote()` or `castShutdownVote()` functions to vote up or down.

The `castStartupVote()` and `castShutdownVote()` function would need to perform the following actions:

* Using the msg.sender and authProof determine the voter
* Ensure the voter is a registered voter (eg. a Guardian)
* Update the vote status for that voter
* Toggle the enabled flag based on whether number of voters meet quorum