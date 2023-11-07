# Wormhole SDK

The Wormhole SDK is a TypeScript SDK for interacting with the chains Wormhole supports and the [protocols](#protocols) built on top of Wormhole. 

## Installation

Install the primary package

```bash
npm install @wormhole-foundation/wormhole-sdk
```

As well as any other platforms you wish to use

```bash
npm install @wormhole-foundation/wormhole-sdk-evm
npm install @wormhole-foundation/wormhole-sdk-solana
npm install @wormhole-foundation/wormhole-sdk-cosmwasm
```

And any protocols you intend to use

```bash
npm install @wormhole-foundation/wormhole-sdk-evm-core
npm install @wormhole-foundation/wormhole-sdk-evm-tokenbridge
npm install @wormhole-foundation/wormhole-sdk-solana-core
npm install @wormhole-foundation/wormhole-sdk-solana-tokenbridge
```

## Usage

A developer would use the core wormhole-sdk package in conjunction with 1 or more of the chain context packages. Most developers don't use every single chain and may only use a couple, this allows developers to import only the dependencies they actually need.

Getting started is simple, just import and pass in the [Platform](#platforms) modules you wish to support as an argument to the Wormhole class.

```ts
import { Wormhole, Signer } from '@wormhole-foundation/wormhole-sdk';
import { EvmContext } from '@wormhole-foundation/wormhole-sdk-evm';
import { SolanaContext } from '@wormhole-foundation/wormhole-sdk-solana';

// include the protocols you wish to use
import "@wormhole-foundation/wormhole-sdk-evm-core"
import "@wormhole-foundation/wormhole-sdk-evm-tokenbridge"
import "@wormhole-foundation/wormhole-sdk-solana-core"
import "@wormhole-foundation/wormhole-sdk-solana-tokenbridge"


const network = "Mainnet"; // Or "Testnet"
const wh = new Wormhole(network, [EvmContext, SolanaContext]);

// Get a ChainContext object for a specific chain 
// Useful to do things like get the rpc client, make Protocol clients, 
// look up configuration parameters or  even parse addresses 
const srcChain = wh.getChain('Ethereum');

srcChain.parseAddress("0xdeadbeef...") // => NativeAddress<'Evm'>
await srcChain.getTokenBridge() // => TokenBridge<'Evm'>
srcChain.getRpcClient() // => RpcClient<'Evm'>
```

### Wormhole Transfer

While using the [ChainContext](#chain-context) and [Protocol](#protocols) clients directly is possible, to do things like transfer tokens, the SDK provides some helpful abstractions.

The `WormholeTransfer` interface provides a convenient abstraction to encapsulate the steps involved in a cross-chain transfer.


### Token Transfers

Performing a Token Transfer is trivial for any source and destination chains. 

We can create a new `WormholeTransfer` object (`TokenTransfer`, `CircleTransfer`, `GatewayTransfer`, ...) and use it to transfer tokens between chains.  The `WormholeTransfer` object is responsible for tracking the transfer through the process and providing updates on its status. 

```ts

// we'll send the native gas token on source chain
const token = 'native'

// format it for base units
const amt = normalizeAmount(1, srcChain.config.nativeTokenDecimals)

// Create a TokenTransfer object, allowing us to shepard the transfer through the process and get updates on its status
const manualXfer = wh.tokenTransfer(
  token,            // TokenId of the token to transfer or 'native'
  amt,              // amount in base units
  senderAddress,    // Sender address on source chain
  recipientAddress, // Recipient address on destination chain
  false,            // No Automatic transfer
)

// 1) Submit the transactions to the source chain, passing a signer to sign any txns
const srcTxids = await manualXfer.initiateTransfer(src.signer);

// 2) Wait for the VAA to be signed and ready (not required for auto transfer)
// Note: Depending on chain finality, this timeout may need to be increased.
// See https://docs.wormhole.com/wormhole/reference/constants#consistency-levels for more info on specific chain finality.
const timeout = 60_000; 
const attestIds = await manualXfer.fetchAttestation(timeout);

// 3) Redeem the VAA on the dest chain
const destTxids = await manualXfer.completeTransfer(dst.signer);
```

Internally, this uses the [TokenBridge](#token-bridge) protocol client to transfer tokens.  The `TokenBridge` protocol, like other Protocols, provides a consistent set of methods across all chains to generate a set of transactions for that specific chain. 

See the example [here](https://github.com/wormhole-foundation/connect-sdk/blob/develop/examples/src/tokenBridge.ts)


### Native USDC Transfers

We can also transfer native USDC using [Circle's CCTP](https://www.circle.com/en/cross-chain-transfer-protocol)

```ts
// OR for an native USDC transfer
const usdcXfer = wh.cctpTransfer(
  1_000_000n,       // amount in base units (1 USDC)
  senderAddress,    // Sender address on source chain
  recipientAddress, // Recipient address on destination chain
  false,            // Automatic transfer
)

// 1) Submit the transactions to the source chain, passing a signer to sign any txns
const srcTxids = await usdcXfer.initiateTransfer(src.signer);

// 2) Wait for the Circle Attestations to be signed and ready (not required for auto transfer)
// Note: Depending on chain finality, this timeout may need to be increased.
// See https://developers.circle.com/stablecoin/docs/cctp-technical-reference#mainnet for more
const timeout = 120_000;
const attestIds = await usdcXfer.fetchAttestation(timeout);

// 3) Redeem the Circle Attestation on the dest chain
const destTxids = await usdcXfer.completeTransfer(dst.signer);
```

See the [example here](https://github.com/wormhole-foundation/connect-sdk/blob/develop/examples/src/cctp.ts)


### Automatic Transfers

Some transfers allow for automatic relaying to the destination, in that case only the `initiateTransfer` is required. The status of the transfer can be tracked by periodically checking the status of the transfer object.

```ts
// OR for an automatic transfer
const automaticXfer = wh.tokenTransfer(
  'native',         // send native gas on source chain
  amt,              // amount in base units
  senderAddress,    // Sender address on source chain
  recipientAddress, // Recipient address on destination chain
  true,             // Automatic transfer
)

// 1) Submit the transactions to the source chain, passing a signer to sign any txns
const srcTxids = await automaticXfer.initiateTransfer(src.signer);
// 2) If automatic, we're done, just wait for the transfer to complete
if (automatic) return waitLog(automaticXfer) ;
```

### Gateway Transfers

Gateway transfers are transfers that are passed through the Wormhole Gateway to or from Cosmos chains.


See example [here](https://github.com/wormhole-foundation/connect-sdk/blob/develop/examples/src/cosmos.ts)


### Recovering Transfers

It may be necessary to recover a transfer that was abandoned before being completed. This can be done by instantiating the Transfer class with the `from` static method and passing one of several types of identifiers.

A `TransactionId` may be used

```ts
// Note, this will attempt to recover the transfer from the source chain
// and attestation types so it may wait depending on the chain finality
// and when the transactions were issued.
const timeout = 60_000;
const xfer = await TokenTransfer.from({
  chain: 'Ethereum',
  txid: '0x1234...',
}, timeout);

const dstTxIds = await xfer.completeTransfer(dst.signer)
```

Or a `WormholeMessageId` if a `VAA` is generated 

```ts
const xfer = await TokenTransfer.from({
  chain: 'Ethereum',
  emitter: toNative('Ethereum', emitterAddress).toUniversalAddress(),
  sequence: '0x1234...',
});

const dstTxIds = await xfer.completeTransfer(dst.signer)
```

## Concepts

Understanding several higher level concepts of the SDK will help in using it effectively.

### Platforms

Every chain is its own special snowflake but many of them share similar functionality.  The `Platform` modules provide a consistent interface for interacting with the chains that share a platform.

Each platform can be installed separately so that dependencies can stay as minimal as possible.


### Chain Context

The `Wormhole` class provides a `getChain` method that returns a `ChainContext` object for a given chain.  This object provides access to the chain specific methods and utilities.  Much of the functionality in the `ChainContext` is provided by the `Platform` methods but the specific chain may have overriden methods. 

The ChainContext object is also responsible for holding a cached rpc client and protocol clients. 

```ts
// Get the chain context for the source and destination chains
// This is useful to grab direct clients for the protocols 
const srcChain = wh.getChain(senderAddress.chain);
const dstChain = wh.getChain(receiverAddress.chain);


srcChain.parseAddress("0xdeadbeef...") // => NativeAddress<'Evm'>
await srcChain.getTokenBridge() // => TokenBridge<'Evm'>
srcChain.getRpcClient() // => RpcClient<'Evm'>
```

### Protocols

While Wormhole itself is a Generic Message Passing protocol, a number of protocols have been built on top of it to provide specific functionality. 

#### Token Bridge

The most familiar protocol built on Wormhole is the Token Bridge.

Every chain has a `TokenBridge` protocol client that provides a consistent interface for interacting with the Token Bridge.  This includes methods to generate the transactions required to transfer tokens, as well as methods to generate and redeem attestations. 

Using the `WormholeTransfer` abstractions is the recommended way to interact with these `Protocols` but it is possible to use them directly

```ts
import {signSendWait} from '@wormhole-foundation/wormhole-sdk';

import "@wormhole-foundation/wormhole-sdk-evm-core"
import "@wormhole-foundation/wormhole-sdk-evm-tokenbridge"

// ...

const tb = await srcChain.getTokenBridge() // => TokenBridge<'Evm'>

const token = '0xdeadbeef...';
const txGenerator = tb.createAttestation(token) // => AsyncGenerator<UnsignedTransaction, ...> 
const txids = await signSendWait(srcChain, txGenerator, src.signer) // => TxHash[]

```

Supported protocols are defined in the [definitions module](https://github.com/wormhole-foundation/connect-sdk/tree/develop/core/definitions/src/protocols)

### Signers

In order to sign transactions, an object that fulfils the `Signer` interface is required.  This is a simple interface that can be implemented by wrapping a web wallet or other signing mechanism.  

```ts
// A Signer is an interface that must be provided to certain methods
// in the SDK to sign transactions. It can be either a SignOnlySigner
// or a SignAndSendSigner depending on circumstances. 
// A Signer can be implemented by wrapping an existing offline wallet
// or a web wallet 
export type Signer = SignOnlySigner | SignAndSendSigner;

// A SignOnlySender is for situations where the signer is not
// connected to the network or does not wish to broadcast the
// transactions themselves 
export interface SignOnlySigner {
    chain(): ChainName;
    address(): string;
    // Accept an array of unsigned transactions and return
    // an array of signed and serialized transactions.
    // The transactions may be inspected or altered before
    // signing.
    // Note: The serialization is chain specific, if in doubt,
    // see the example implementations linked below
    sign(tx: UnsignedTransaction[]): Promise<SignedTx[]>;
}

// A SignAndSendSigner is for situations where the signer is
// connected to the network and wishes to broadcast the
// transactions themselves 
export interface SignAndSendSigner {
    chain(): ChainName;
    address(): string;
    // Accept an array of unsigned transactions and return
    // an array of transaction ids in the same order as the  
    // UnsignedTransactions array.
    signAndSend(tx: UnsignedTransaction[]): Promise<TxHash[]>;
}
```


See the testing signers ([Evm](https://github.com/wormhole-foundation/connect-sdk/blob/develop/platforms/evm/src/testing/signer.ts), [Solana](https://github.com/wormhole-foundation/connect-sdk/blob/develop/platforms/solana/src/testing/signer.ts), ...) for an example of how to implement a signer for a specific chain or platform.


```ts
// Create a signer for the source and destination chains
const sender: Signer =  // ...
const receiver: Signer = // ...

```

### Addresses

Within the Wormhole context, addresses are [normalized](https://docs.wormhole.com/wormhole/blockchain-environments/evm#addresses) to 32 bytes and referred to in this SDK as a `UniversalAddresses`.

Each platform comes with an address type that understands the native address formats, unsuprisingly referred to a NativeAddress. This abstraction allows the SDK to work with addresses in a consistent way regardless of the underlying chain. 

```ts
// Convert a string address to its Native address
const ethAddr: NativeAddress<'Evm'> = toNative('Ethereum', '0xbeef...');
const solAddr: NativeAddress<'Solana'> = toNative('Solana', 'Sol1111...')

// Convert a Native address to its string address
ethAddr.toString() // => '0xbeef...'

// Convert a Native address to a UniversalAddress
ethAddr.toUniversalAddresS()

// A common type in the SDK is the `ChainAddress`. 
// A helper exists to provide a ChainAddress for a signer, or [ChainName, string address]
const senderAddress: ChainAddress = nativeChainAddress(sender)     
const receiverAddress: ChainAddress = nativeChainAddress(receiver) 
```


## See also

The tsdoc is available [here](https://wormhole-foundation.github.io/connect-sdk/)


## WIP

:warning: This package is a Work in Progress so the interface may change and there are likely bugs.  Please [report](https://github.com/wormhole-foundation/connect-sdk/issues) any issues you find. :warning:
