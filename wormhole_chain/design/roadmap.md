# Wormhole Chain Roadmap

## Why Wormhole Chain?

At the time of writing, the Wormhole Guardian Network is a decentralized network of 19 validators operating in a Proof-of-Authority consensus mechanism. They validate using the guardiand program from the Wormhole core repository, and perform governance off-chain. There are currently no features to the Wormhole Guardian Network except the core function of observing and signing VAAs.

The roadmap is that as Wormhole grows and matures, it will add more advanced features, such as Accounting, Wormhole Pipes, Cross-Chain Queries, and more. It will also move its governance, VAA generation, and validator on-boarding to a formalized process, and change its governance structure from PoA to another mechanism (possibly a modified Proof of Stake mechanism).

Considering that future Wormhole governance is intended to be token based, and that many of these upcoming features are incentivized via fee or staking mechanisms, the obvious path forward is to launch a purpose-built blockchain to support this vision.

## Why Cosmos?

When building a new blockchain, there are not particularly many options. The primary options are:

- Fork an existing blockchain
- Use the Cosmos SDK
- Build one from scratch
- Implement into an existing environment (parachain), or implement as a layer 2.

There is not any blockchain in particular which stands out to be forked, and most forked blockchains (such as Ethereum), would require maintaining a smart-contract runtime, which is an unnecessary overhead for Wormhole Chain.

The Cosmos SDK is the most sensible choice, as its extensible 'module' system allows for the outlined features of Wormhole Chain to be easily added to the out-of-the-box runtime. Additionally, the Cosmos SDK's use of Tendermint for its consensus mechanism can easily be modified to support Proof of Authority for block production, while still allowing stake-weighted on-chain voting for governance. In the future, PoA can also be seamlessly swapped out for an entirely PoS system. Using the Cosmos SDK will also open up the opportunity for the Wormhole ecosystem to directly leverage the IBC protocol.

Because the Cosmos SDK is able to support the planned feature set of Wormhole Chain, building it from scratch would largely be a case of 'reinventing the wheel', and even potentially sacrifice features, such as IBC support.

The last option would be to implement the chain as a layer two, or integrate as a parachain in an existing environment like Polkadot. Both of these create dependencies and constraints for Wormhole which would make it hard to hand-roll a consensus mechanism or unilaterally develop new functionality.

## Wormhole Chain Explorer

Every blockchain should have an explorer, as it is a useful tool. The primary Cosmos blockchain explorers are shown here:

https://github.com/cosmos/awesome#block-explorers

Of the explorers listed, the two most popular and well-supported appear to be PingPub, and Big Dipper v2. 

Big Dipper seems to be the more robust, popular, and feature-rich of the two. Its only downside appears to be that it (quite reasonably) requires an external database. It also has the added benefit of being built in React, which will be easier to support, and allow the blockchain explorer to be more easily merged with the existing Wormhole Network Explorer. For these reasons, it stands out as being the best production candidate.

PingPub (LOOK Explorer) has the benefit of only requiring an LCD connection. Because it is very easy to run, it may be useful as a development tool.

# Feature Roadmap

The upcoming Wormhole Chain features fall into four categories, roughly arranged in their dependency ordering.

- Basic Functionality
- Accounting
- Governance
- New Cross-Chain Functionality

## Basic Functionality

This category contains the critical features which allow Wormhole Chain to produce blocks, and for downstream functions to exist.

### Proof of Authority Block Production

The 19 Guardians of the current Wormhole Network will also serve as the 19 validators for the Wormhole Chain. Furthermore, new Guardians must be able to register as validators on the Wormhole Chain, so that the Guardian Set Upgrade VAAs can be submitted in a Wormhole Chain transaction to change its validator set, akin to the process on other chains.

### Core Bridge and Token Bridge

Wormhole Chain will contain critical functions for the Wormhole network, but it will also be another connected chain to the network. As such it will need an implementation of the Wormhole Core Bridge and Token Bridge.

## Accounting

Accounting is a defense-in-depth security mechanism, whereby the guardians will keep a running total of the circulating supply of each token on each chain, and refuse to issue VAAs for transfers which are logically impossible (as they must be the result of an exploit or 51% attack).

More advanced mechanisms of accounting are planned as well, which would track token custody at a finer level to detect and prevent exploits from propagating across chains.

There is also the option to move the current gossip network by which the guardians sign VAAs on-chain, which could allow for accounting to be more tightly integrated into the signing process.

## Governance
While block production will initially be PoA, the governance mechanism is intended to launch with on-chain voting in a PoS system. 

In order to vote, users will have to transfer and stake $WORM tokens from another chain. The on-chain voting process should otherwise be quite similar to other Cosmos chains.

## New Cross-Chain Functionality

### Cross-Chain Queries
Cross Chain queries are a mechanism by which read-only data can be requested from a chain without actually submitting a transaction on that chain. For example, a user could request a VAA for the current balance of an Ethereum wallet by submitting a transaction Wormhole Chain. This would be a unified location to request data from any chain, and be a tremendous cost-saving mechanism over executing transactions on other L1s.

### Wormhole Pipes
Wormhole pipes are similar to Cross-Chain queries, but act via a 'Push' model whereby contracts can subscribe to systematically read data from other chains. Subscriptions would be managed via Wormhole Chain.

### Many others
Going forward, Wormhole Chain will be an excellent mechanism for both the public to interact with the Guardian network, and for the Guardians to communicate between themselves. As such, Wormhole Chain should become the primary mechanism by which requests are made to the Guardians, and how new oracle features are implemented.
