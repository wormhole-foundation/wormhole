# IBC Generic Message Emission

## Objective

Since Wormchain is a cosmos-sdk based chain that is IBC-enabled, we can leverage IBC generic messaging to reduce the operational burden required to run a guardian node. Wormhole guardians should, therefore, be capable of scaling to support all IBC-enabled chains at minimal cost while only running a full node for Wormchain.

## Background

[IBC](https://ibcprotocol.org/) is the canonical method of generic message passing within the cosmos ecosystem. IBC is part of the cosmos-sdk and Cosmos chains can enable it to connect with other cosmos chains.

[Wormchain](https://github.com/wormhole-foundation/wormhole/tree/main/wormchain) is a cosmos-sdk based blockchain that has been purpose-built to support Wormhole. It allows the guardians to store global state on all the blockchains which Wormhole connects to, and enables a suite of product and security features.

## Goals

- Remove the requirement for guardians to run full nodes for IBC-enabled chains. Guardians should be able to support any IBC-enabled chain by only running a full Wormchain node.
- Define a custom IBC specification for passing Wormhole messages from IBC-enabled chains to Wormchain.
- Ensure this design is backwards-compatible with existing Cosmos integrations.
- Ensure this design does not violate any of Wormhole's existing security assumptions.

## Non-Goals

This document does not propose new cosmos networks for Wormhole to support. It is focused on the technical design of using IBC generic messaging to reduce the operational load on Wormhole guardians.

This document is also not meant to describe how Wormhole can be scaled beyond the cosmos ecosystem.

## Overview

Currently, Wormhole guardians run full nodes for every chain that Wormhole is connected to. This is done to maximize security and decentralization. Since each guardian runs a full node for each chain, they are able to independently verify the authenticity of Wormhole messages that are posted on different blockchains. However, running full nodes has its drawbacks. Specifically, adding new chains to Wormhole has a high operational cost per chain, which makes it difficult to scale Wormhole.

Luckily, we can leverage standards such as IBC to scale Wormhole's support for the cosmos ecosystem and other chains that implement IBC. Since IBC messages are trustlessly verified by tendermint light clients, we can pass Wormhole messages from any IBC enabled chain over IBC to Wormchain, which will then emit that message for the Wormhole guardians to pick up. This way, the Wormhole guardians only need to run a full node for Wormchain to be able to verify the authenticity of messages on all other IBC-enabled chains.

## Detailed Design

### External Chain -> Cosmos Chain

This will work exactly the same way it works today. We will deploy our Wormhole cosmwasm contract stack to every cosmos chain we want to support. Wormhole relayers will be able to post VAAs produced for any source chain directly to the cosmos destination chain.

### Cosmos Chain -> External Chain

Typically, the Wormhole core bridge contract emits a message which the guardians then pick up from their full nodes.

For cosmos chains, we update the core bridge contract to instead send this message over IBC to Wormchain. Then a Wormchain contract receives the message to emit and actually emits it, which the guardians then pick up.

Specifically, we implement two new cosmwasm smart contracts: `wormhole-ibc` and `wormchain-ibc-receiver`.

The `wormhole-ibc` contract is meant to replace the `wormhole` core bridge contract on cosmos chains. It imports the `wormhole` contract as a library and delegates core functionality to it before and after running custom logic:
- During execution, it delegates all logic to the core bridge library. After the core bridge library has finished execution, it will send the response as an IBC message to the `wormchain-ibc-receiver` contract **if and only if** the execution message is of type `wormhole::msg::ExecuteMsg::PostMessage`.

Sending an IBC packet requires choosing an IBC channel to send over. IBC `(channel_id, port_id)` pairs are unique. All IBC-enabled cosmwasm contracts follow a standard `port_id` format: `wasm.<contract_address>`. Therefore, to choose a correct channel the `wormhole-ibc` contract will need to know the address of the `wormchain-ibc-receiver` contract - it will select any channel that is paired with the expected port. 

The `wormchain-ibc-receiver` contract will be deployed on Wormchain and is meant to receive the IBC messages the `wormhole-ibc` contract sends from various IBC enabled chains. Its only responsibility is to receive the IBC message, send an IBC acknowledgement to the source chain, and emit the message for the guardian node to observe.

### IBC Relayers

All IBC communication is facilitated by [IBC relayers](https://ibcprotocol.org/relayers/). Since these are lightweight processes that need to only listen to blockchain RPC nodes, each (only several is also acceptable) Wormhole guardian can run a relayer.

The guardian IBC relayers are configured to connect the `wormchain-ibc-receiver` contract on Wormchain to the various `wormhole-ibc` contracts on the cosmos chains that Wormhole supports.

### Guardian Node Watcher

We will add a new IBC guardian watcher to watch the `wormchain-ibc-receiver` contract on Wormchain for the messages from the designated `wormhole-ibc` contracts on supported IBC enabled chains. This is nearly identical to the current cosmwasm watcher and can be a drop in replacement.

The new guardian watcher verifies that messages originate from the chain they claim to originate from by checking the IBC connection ID. The `wormchain-ibc-receiver` contract logs the channel ID the message was received from. The watcher can lookup the corresponding connection ID for the channel ID by querying the IBC module on Wormchain. Since IBC connections are unique and can never be updated or closed ([docs](https://github.com/cosmos/ibc/tree/main/spec/core/ics-003-connection-semantics#sub-protocols)), we will maintain a mapping of connection IDs to chain IDs in the `wormchain-ibc-receiver` contract. This is a trusted location for the mapping which can only be updated using the new `IbcReceiverUpdateChainConnection` governance VAA type. 

### API / database schema

```rust
/// This is the message we send over the IBC channel
#[cw_serde]
pub enum WormholeIbcPacketMsg {
    Publish { msg: Response }
}
```

## Deployment

There are several steps required to deploy this feature. Listed in order:

1. Deploying the new contracts: `wormhole-ibc` contracts to IBC enabled chains and the `wormhole-ibc-receiver` contract to Wormchain.
2. Upgrading existing `wormhole` contracts on IBC enabled chains to use the new `wormhole-ibc` bytecode.
3. Establishing IBC connections between the `wormhole-ibc` contracts and the `wormhole-ibc-receiver` contract.
4. Upgrading the guardian software.

First, we need to deploy the `wormhole-ibc-receiver` contract on Wormchain. This will require 2 governance VAAs to deploy and instantiate the bytecode.

Once we know the `wormhole-ibc-receiver` contract address, we can hardcode this (or, using an environment variable during compilation) into the `wormhole-ibc` contracts. Recall that these need to know the address of the `wormhole-ibc-receiver` contract to look up the correct IBC channel over which to send messages.

Next, we should deploy the compiled `wormhole-ibc` contracts to IBC-enabled chains we already support (Terra Classic, Terra2, XPLA, and Injective). We can migrate the existing `wormhole` contract to the new `wormhole-ibc` bytecode so that we don't need to redeploy and re-instantiate the token bridge contracts with new core bridge contract addresses.

Next, we should use relayers to establish connections between the `wormhole-ibc` contracts and the `wormchain-ibc-receiver` contract on Wormchain. After we establish these IBC connections and upgrade the guardians to support the new `IbcReceiverUpdateChainConnection` governance VAA type, we can perform governance to build mappings of `connectionId -> chainId` on the `wormchain-ibc-receiver` contract.

Finally, we'll be ready to upgrade the guardians to a new software version.