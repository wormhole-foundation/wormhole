# Wormhole SDK

> Note: This is a pre-alpha release and in active development. Function names and signatures are subject to change.

## What is Wormhole?

[Wormhole](https://wormholenetwork.com/) allows for the transmission of arbitrary data across multiple blockchains. Wormhole currently supports the following platforms:

- Solana
- Ethereum
- Terra
- Binance Smart Chain

Wormhole is, at its base layer, a very simple protocol. A Wormhole smart contract has been deployed on each of the supported blockchains, and users can emit messages in the Wormhole Network by submitting data to the smart contracts. These messages are quite simple and only have the following six fields.

- _emitterChain_ - The blockchain from which this message originates.
- _emitterAddress_ - The public address which submitted the message.
- _consistencyLevel_ - The number of blocks / slots which should pass before this message is considered confirmed.
- _timestamp_ - The timestamp when the Wormhole Network confirmed the message.
- _sequence_ - An incrementing sequence which denotes how many messages this _emitterAddress_ has emitted.
- _payload_ - The arbitrary contents of this message.

Whenever a wormhole contract processes one of these messages, participants in the Wormhole Network ( individually known as **Guardians** ), will observe the transaction and create a **SignedVAA** (Signed Verifiable Action Approval) once the transaction has reached the specified confirmation time on the emitter chain.

The SignedVAA is essentially an affirmation from the Wormhole Network that a transaction has been finalized on the emitter chain, and that any dependent actions on other chains may proceed.

While simple, the Wormhole Protocol provides a powerful base layer upon which many 'bridge' applications can be built. Because Wormhole is capable of verifying arbitrary data, bridges utilizing it are able to transfer native currencies, tokens, NFTs, oracle data, governance votes, and a whole host of other forms of decentralized data.

## How the Core Wormhole Bridge Works

The core Wormhole bridge operates by running smart contracts on both the _Source Chain_ (where the data currently resides) and the _Target Chain_ (where the data will be moved), and generally follows this workflow:

    1) An end user or another smart contract publishes a message using the Bridge Contract on the Source Chain.

    2) The Wormhole Network observes this transaction and issues a SignedVAA once it crosses its confirmation threshold.

    3) An off-chain process collects the SignedVAA and submits it in a transaction to the Bridge Contract on the Target Chain, which can parse and verify the message.

## How the Wormhole Token Bridge Works

It is important to note that the Wormhole Token Bridge is not, strictly speaking, part of the Wormhole protocol, but rather a bridge on top of it. However, as token transfers are such an important use-case of the bridge, it is built and packaged as part of the Wormhole SDK.

The Token Bridge works in the same fashion as above, leveraging the Core Bridge to publish messages. However, there are actually two different functions in the token bridge: Attest and Transfer.

### Attest

Attestation is the process by which a token is 'registered' with the token bridge. Before being transferred, tokens must first be attested on their **Origin Chain** and have corresponding wrapped tokens created on the **Foreign Chain** to which they will be transferred. Attesting on the Origin Chain will create requisite addresses and metadata that will allow the wrapped asset to exist on Foreign Chains.

### Transfer

Once attested, tokens are mapped from their Native Chain to 'wrapped' assets on the Foreign Chains. Transferring an Ethereum-native token to Solana will result in a 'wrapped asset' on Solana, and transferring that same asset back to Ethereum will restore the native token.

It is important to note that Wormhole wrapped tokens are distinct from and incompatible with tokens wrapped by other bridges. Transferring a token which was wrapped by a different bridge will not redeem the native token, but rather will result in a 'double-wrapped' token.

## Examples

The integration tests in the [source code](https://github.com/certusone/wormhole/blob/dev.v2/sdk/js/src/token_bridge/__tests__/integration.ts) have many full-path examples, while the [example Token Bridge UI](https://github.com/certusone/wormhole/tree/dev.v2/bridge_ui) demonstrates how to integrate it.

### Attest

#### Solana to Ethereum

```js
// Submit transaction - results in a Wormhole message being published
const transaction = await attestFromSolana(
  connection,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  payerAddress,
  mintAddress
);
const signed = await wallet.signTransaction(transaction);
const txid = await connection.sendRawTransaction(signed.serialize());
await connection.confirmTransaction(txid);
// Get the sequence number and emitter address required to fetch the signedVAA of our message
const info = await connection.getTransaction(txid);
const sequence = parseSequenceFromLogSolana(info);
const emitterAddress = await getEmitterAddressSolana(SOL_TOKEN_BRIDGE_ADDRESS);
// Fetch the signedVAA from the Wormhole Network (this may require retries while you wait for confirmation)
const { signedVAA } = await getSignedVAA(
  WORMHOLE_RPC_HOST,
  CHAIN_ID_SOLANA,
  emitterAddress,
  sequence
);
// Create the wrapped token on Ethereum
await createWrappedOnEth(ETH_TOKEN_BRIDGE_ADDRESS, signer, signedVAA);
```

#### Ethereum to Solana

```js
// Submit transaction - results in a Wormhole message being published
const receipt = await attestFromEth(
  ETH_TOKEN_BRIDGE_ADDRESS,
  signer,
  tokenAddress
);
// Get the sequence number and emitter address required to fetch the signedVAA of our message
const sequence = parseSequenceFromLogEth(receipt, ETH_BRIDGE_ADDRESS);
const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
// Fetch the signedVAA from the Wormhole Network (this may require retries while you wait for confirmation)
const { signedVAA } = await getSignedVAA(
  WORMHOLE_RPC_HOST,
  CHAIN_ID_ETH,
  emitterAddress,
  sequence
);
// On Solana, we have to post the signedVAA ourselves
await postVaaSolana(
  connection,
  wallet,
  SOL_BRIDGE_ADDRESS,
  payerAddress,
  signedVAA
);
// Finally, create the wrapped token
const transaction = await createWrappedOnSolana(
  connection,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  payerAddress,
  signedVAA
);
const signed = await wallet.signTransaction(transaction);
const txid = await connection.sendRawTransaction(signed.serialize());
await connection.confirmTransaction(txid);
```

### Transfer

#### Solana to Ethereum

```js
// Submit transaction - results in a Wormhole message being published
const transaction = await transferFromSolana(
  connection,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  payerAddress,
  fromAddress,
  mintAddress,
  amount,
  targetAddress,
  CHAIN_ID_ETH,
  originAddress,
  originChain
);
const signed = await wallet.signTransaction(transaction);
const txid = await connection.sendRawTransaction(signed.serialize());
await connection.confirmTransaction(txid);
// Get the sequence number and emitter address required to fetch the signedVAA of our message
const info = await connection.getTransaction(txid);
const sequence = parseSequenceFromLogSolana(info);
const emitterAddress = await getEmitterAddressSolana(SOL_TOKEN_BRIDGE_ADDRESS);
// Fetch the signedVAA from the Wormhole Network (this may require retries while you wait for confirmation)
const { signedVAA } = await getSignedVAA(
  WORMHOLE_RPC_HOST,
  CHAIN_ID_SOLANA,
  emitterAddress,
  sequence
);
// Redeem on Ethereum
await redeemOnEth(ETH_TOKEN_BRIDGE_ADDRESS, signer, signedVAA);
```

#### Ethereum to Solana

```js
// determine destination address - an associated token account
const solanaMintKey = new PublicKey(
  (await getForeignAssetSolana(
    connection,
    SOLANA_TOKEN_BRIDGE_ADDRESS,
    CHAIN_ID_ETH,
    hexToUint8Array(nativeToHexString(tokenAddress, CHAIN_ID_ETH) || "")
  )) || ""
);
const recipientAddress = await Token.getAssociatedTokenAddress(
  ASSOCIATED_TOKEN_PROGRAM_ID,
  TOKEN_PROGRAM_ID,
  solanaMintKey,
  walletAddress
);
// Submit transaction - results in a Wormhole message being published
const receipt = await transferFromEth(
  ETH_TOKEN_BRIDGE_ADDRESS,
  signer,
  tokenAddress,
  amount,
  CHAIN_ID_SOLANA,
  recipientAddress
);
// Get the sequence number and emitter address required to fetch the signedVAA of our message
const sequence = parseSequenceFromLogEth(receipt, ETH_BRIDGE_ADDRESS);
const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
// Fetch the signedVAA from the Wormhole Network (this may require retries while you wait for confirmation)
const { signedVAA } = await getSignedVAA(
  WORMHOLE_RPC_HOST,
  CHAIN_ID_ETH,
  emitterAddress,
  sequence
);
// On Solana, we have to post the signedVAA ourselves
await postVaaSolana(
  connection,
  wallet,
  SOL_BRIDGE_ADDRESS,
  payerAddress,
  signedVAA
);
// Finally, redeem on Solana
const transaction = await redeemOnSolana(
  connection,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  payerAddress,
  signedVAA,
  isSolanaNative,
  mintAddress
);
const signed = await wallet.signTransaction(transaction);
const txid = await connection.sendRawTransaction(signed.serialize());
await connection.confirmTransaction(txid);
```
