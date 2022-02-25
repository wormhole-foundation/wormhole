# Token/NFT Bridge Shutdown (aka: "KillSwitch") [DRAFT]

[TOC]

## Objective

To enable/disable send and recieve transactions on the Wormhole Token Bridge and NFT Bridge.

## Background

Wormhole facilitates value exchange cross-chain via the Token Bridge and the NFT bridge.  Given the significant value exchange using smart contracts and accepting that security bugs may exist in the future, Wormhole security guardians (a subset of the 19) may need to take quick and decisive action (typically minutes) to enable/disable transactions without a 2/3s governance vote of the full 19 (typically hours/days).

## Goals

Implement a safe guard in smart contracts that facilitate the enabling or disabling of Token Bridge and NFT Bridge send/recieve functionality in the event of an existential security threat.

* Enable/Disable send/recieve Token/NFT bridge transactions
* Effectiveness of the control within minutes of defined need (instead of hours/days)

## Non-Goals

* Solving for undercolleralized assets
* Solving for transaction thresholding/volume limits
* Solving at the core layer
* Solving for wallet segmentation or hot/cold wallets
* Solving for 2/3+ Guardian Consensus

## Overview

The shutdown functionality aims to extend exisiting smart contracts that enable Token and NFT bridge to respond to a set of votes from the Wormhole Security Counsel (a subset of the 19 Guardians) that would enable/disable send and receive functionality during an existential threat scenario at a speed greater than could be acheived with a 2/3+ majority vote from the full guardian set of 19.

During an existential threat scenario where a member of the security counsel believes there is a need to disable send/receive transactions on the Token/NFT Bridge they would send a message to the relevant smart contract(s) to indicate their vote to disable send/receive transactions.  Once these bridge smart contracts receive a majority of security guardian votes to shutdown the disabling of send/receive becomes effective.  If concensus votes falls out of majority, send/receive functionality would be restored.

## Detailed Design (Option 1)

Each smart contract must implement the following 3 storage components:

* `nonce`: an `int` that increments for each signer to guard against replay attacks
* `signers`: a `list` of public keys from the security guardians who are authorized to vote
* `votes`: a `list` of security guardian votes as signers + bools (true = up, false = down)

For each send/receive capability for a token/nft bridge contract, it must implement a check against votes to ensure a majority up vote to enable/disable transactions.

To initiate a vote, each security guardian must retrieve the current `nonce` and send a `signed message` that includes the `nonce` and their `vote`.  Once a `vote` is placed, it increments the nonce by 1.  Any implementation must check the `signed message` and make sure it matches the existing `nonce`  and is a valid signer before registering a vote.

Each smart contract must implement one function:

* `shutdown`: a function that accepts `signed_message` which includes `nonce` and `vote` from a given signer.  This method will validate the signer ID is in `signers` and the `nonce` is valid (to prevent replay attacks) before adding the `vote` (true = down, false = up) for that signer and incrementing the nonce.

### Pros

* Eliminates the burden of binding shutdown actions from a specific wallet
* Normal governance can be used to update signers as the security gaurdians change

### Cons

* Will require some CLI tooling or manual steps for a signer to obtain the current nonce before signing a message to ensure speed
* If for some reason the nonce rolls over and becomes non-unique, it could be succeptable to replay attacks that the nonce aims to solve for
* Will require each signer to wait until the next block before signing


## Detailed Design (Option 2)

Each smart contract must implement the following 3 storage components:

* `wallets`: a `list` of wallets for which we will receive signed votes from
* `signers`: a `list` keys of security guardians within the security counsel who are authorized to vote
* `votes`: a `list` of security guardian votes as signers + bools (true = up, false = down)

For each send/receive capability for a token/nft bridge contract, it must implement a check against votes to ensure a majority up vote to enable/disable transactions.

Each smart contract must implment one function:

* `shutdown`: a function that accepts `signed_message` which includes `vote` from a given signer.  This method will validate the `signer` and the `wallet` are in `signers` and `wallets` before adding the `vote` (true = down, false = up) for that signer.

### Pros

* Less complexity and thus less potential for abuse when compared to option 1
* Normal governance can be used to update signers as the security gaurdians change

### Cons

* Security Guardians would be limited from sending a `vote` from a single wallet.  Alternatively, if an implementation didn't bind wallets and signers 1:1, a single guardian could send `signed message` on another security guardians behalf if that guardian couldn't use their wallet.

## Detailed Design (Option N)

If you have other proposals, please add them here or suggest them via the PR.