Design Document for Wormhole SDK
---------------------------------

# Organization

Code is organized into workspaces so that each can be its own module.

```
core/
    base/ -- Constants
    definitions/ -- VAA structures and Protocol definitions

connect/ -- Primary package and interface through Wormhole 

platforms/ -- Platform specific logic 
    evm/
        protocols/
            core/
            tokenBridge/
            cctp/
        src/
            chain.ts
            platform.ts
            ...
    solana/
        protocols/
            core/
            tokenBridge/
        src/
            chain.ts
            platform.ts
            ...
    cosmwasm/
        protocols/
            core/
            tokenBridge/
            ibc/
```

# Concepts

The `Wormhole` class provides methods to interact with the Wormhole protocol by mapping chain parameters to the `Platform` and `Chain` specific implementations.

A `Platform` is a blockchain runtime, often shared across a number of chains (e.g. `Evm` platform for `Ethereum`, `Bsc`, `Polygon`, etc ...). 

A `Chain` is a specific blockchain, potentially with overrides for slight differences in the platform implementation. 

A `Protocol` (fka `Module`) is a specific application on a `Chain`, it provides a set of methods that can be called to accomplish some action (e.g. `TokenBridge` allows send/receive/lookup token, etc...)

A `Signer` is an interface that provides a callback to sign one or more transaction objects. These signed transactions are sent to the blockchain to invoke some action.

An `Attestation` is _some_ proof that some _thing_ happened on a remote chain, sent to the target chain to complete a transfer.


# Details 

## Wormhole 

Registers Platforms

Allows overriding chain specific configs (rpc, contract addresses, ...)

Provides methods to get PlatformContext or ChainContext objects

```ts
const ep: EvmPlatform = wh.getPlatform("Evm")
const eth: ChainContext<"Evm"> = wh.getChain("Ethereum")
```

Provides methods to create a `WormholeTransfer` for any `Protocol`
```ts
const tt: TokenTransfer = wh.tokenTransfer(...)
const cctpXfer: CircleTransfer = wh.cctpTransfer(...)
//...
```

Provides methods to query an API for VAAs and token details
```ts
// request and parse a vaa with a timeout 
const parsedVaa = wh.getVaa(chainName, emitter, sequence, "TokenBridge:Transfer", 30*1000)

// get the token details 
wh.getOriginalToken(...)
wh.getWrappedToken(orig, chain)
```

## Platform

Base class, implements Platform specific logic

Parse Addresses

Parse Message out of Transaction

Sign/Send/Wait

## ChainContext

Defined in abstract ChainContext class

Note: Dispatches many calls to the Platform, filling in details like ChainName and RPC

The `getRpc` method is the only platform specific thing _required_ to implement.

Responsible for holding RPC connection, initialized from default or overrides
```ts
cc.getRPC() // for evm -> ethers.Provider, for sol -> web3.Connection
```

Holds cached references to Protocol client


## WormholeTransfer

Holds a reference to ChainContexts

Holds details about the transfer

Provides methods to step through the transfer process

## Glossary

- Network
    Mainnet, Testnet, Devnet
- Platform
    A chain or group of chains within the same ecosystem that share common logic (e.g. EVM, Cosmwasm, etc)
- Chain Context
    A class which implements a standardized format and set of methods. It may include additional chain-specific methods and logic.
- Protocol 
    A cross-chain application built on top of Wormhole (the core contracts are also considered a module)
- Universal Address
    A 32-byte address, used by the wormhole contracts
- Native Address (I think this should be called "Platform Address")
    An address in the standard chain-specific format
- Native
    The "home" chain (e.g. ETH is native to Ethereum)
- Foreign
    A non-native chain (e.g. ETH is foreign to Moonbeam)
- VAA (Verified Action Approval)
    The core messaging primitive in Wormhole, it contains information about a message and a payload encoded as bytes.  Once finality is achieved and it is observed by the majority of the guardians, it is signed and can then be used to complete a transfer on the destination chain
- Payload
    Bytes that can be passed along with any wormhole message that contain application-specific data
- Finality/Finality Threshold
    The required number of blocks to wait until a VAA is produced

# Discussion


## What's the purpose of the Wormhole class?

Wormhole class provides the main interface to do _everything_

- Registers Platforms to access later -- constructor
- Provides access to ChainContexts -- getContext(ChainName)
- Provides "shortcuts" to start a WormholeTransfer -- tokenTransfer/nftTransfer/cctpTransfer/...
- Helpers for getting VAAs or generally querying the API
- Abstract away chain-specific logic for easy mode access to methods

## What do we want from a Platform Module?

Provides Chain/Platform specific logic for a set of things

- Register Protocols (contract/app specific functionality)
- Translates Platform specific stuff to generic stuff (e.g. ethers.Provider => RPC connection)
- Deals with Platform specific interaction w/ chain (approve on eth, postvaa on sol, ...)
- Implements standardized method format

## What's the relationship between platforms/chains/providers?

- A Platform provides the logic for all chains that run on that platform, it can be used like a library.
- A Chain provides a convenient way to populate args for Platform methods and caches RPC and Protocol clients. 
- A Provider is just an RPC connection and is held by the Chain. Providers are an implementation detail.

## What's a signer vs. a wallet?

- A Signer is an interface to sign transactions, giving the SDK a way to sign transactions without needing to know about wallets
- It _may_ be backed by some other wallet by wrapping it in the appropriate interface. 




# Outstanding Questions: 

Can we provide some way to make other non-standard applications available to use through the WormholeTransfer?

    Say I have an app that defines its own protocol, can I provide something that adheres to the WormholeTransfer interface so a dev can install it and call it like the TokenTransfer?

How should we think about xchain concepts without having xchain context?

    Given eth address, and without installing evm platform, how do I turn it into a solana wrapped token without knowing how to format the address? 
    Given a transfer from Eth=>Sol, and without installing sol platform, how do I determine the ATA?


What is the benefit of costmap vs single fat object