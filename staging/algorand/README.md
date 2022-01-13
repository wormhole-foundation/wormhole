
# Pricecaster Service

## Introduction

This service consumes prices from "price fetchers" and feeds blockchain publishers. There are two basic flows implemented:

* A basic Algorand publisher class with a TEAL program for messages containing signed price data. The program code validates signature and message validity, and if successful, subsequently stores the price information in the global application information for other contracts to retrieve. For the description of the data format used, see below.

* A Wormhole client that uses the JS SDK to get VAAs from Pyth network and feed the payload and cryptographic verification data to a transaction group for validation. Subsequently, the data is optionally processed and stored, either price or metrics. For details regarding Wormhole VAAs see design documents: 

  https://github.com/certusone/wormhole/tree/dev.v2/whitepapers

All gathered price information is stored in a buffer by the Fetcher component -with a maximum size determined by settings-.  The price to get from that buffer is selected by the **IStrategy** class implementation; the default implementation being to get the most recent price and clear the buffer for new items to arrive. 

Alternative strategies for different purposes, such as getting averages and forecasting, can be implemented easily.

## System Overview


**The objective is to receive signed messages -named as Verifiable Attestments (VAAs) in Wormhole jargon- from our relayer backend (Pricecaster) , verify them against a fixed (and upgradeable) set of "guardian public keys" and process them, publishing on-chain price information or doing governance chores depending on the VAA payload.**


The design is based in two contracts that work in tandem, a  **Stateful contract (VAA_Processor)** that accepts calls for verifying and commiting VAAs, and also mantains the global guardian set; and a **verifier stateless contract** that does the computational work of ECDSA signature verification.

Due to computation and space limits, the validation of the 19 guardian signatures against the payload is partitioned so each stateless contract validates a subset of the guardian signatures. If ECDSA decompress and validation opcodes are used, that yields 650+1750 = 2400 computation units * 7 = 16800, leaving 3200 free units for remaining opcodes.
In our design, We call **verification step** to each of the app calls + stateless logic involved  in verifying a block of signatures.

The number of signatures in each verification step is fixed at contract compilation stage, so with this in mind and example values:

* let $N_S$ be the total signatures to verify $(19)$
* let $N_V$ be the number of signatures per verification step $(7)$,   
* the required number of transactions $N_T = \lceil{N_S/N_V}\rceil = \lceil{19/7}\rceil = 3$
* Each transaction-step $T_i$ will verify signatures $[j..k]$ where $j = i \times N_V$, $k = min(N_S-1, j+N_V-1)$, so for $T_0 = [0..6]$, $T_1 = [7..13]$, $T_2 = [14..18]$. 

The verification process inputs consist of: 
1. the set of current guardian public keys, 
2. the signed message digest (VAA information fields + generic payload), 
3. the set of signatures in the VAA header.  

With the above in mind, and considering the space and computation limits in the current Algorand protocol, the typical flow for verifying a VAA for 19 guardians using step-size of 7, would be based on the following transaction group:


| TX# | App calls | Stateless logic |
| --- | --------- | --------------- |
|  0  | _args_: guardian_pk[0..6], _txnote_: signed_digest          | _args_: sig[0..6]    |
|  1  | _args_: guardian_pk[7..13], _txnote_: signed_digest          | _args_: sig[7..13]   |
|  2  | _args_: guardian_pk[14..18], _txnote_: signed_digest          | _args_: sig[14..18]  |

Regarding stateless logic we can say that,

* Its code is constant and it's known program hash is validated by the stateful program.
* Asserts that the appropiate stateful program is called using known AppId embedded at compile stage.
* Passing signature subset through arguments does not pose any higher risk since any tampered signature will make the operation to fail; 
* The signed digest and public keys are retrieved through transaction note field and argument. This limits for the current design the maximum digest size to 1000 bytes and the maximum number of public keys -and guardians to ~64.
* Verification is performed using TEAL5 ECDSA opcodes. If any signature do not verify, transaction fails and subsequently, the entire transaction group aborts.

For the stateful app-calls we consider,

* Global state stores guardian public-keys, entry count (set size) and guardian set expiration time.
* Initial state after deployment could be set through a bootstrap call, using last guardian-set-change governance VAA if available.
* Sender must be stateless logic 
* Argument 1 must contain guardian public keys for guardians $[k..j]$
* Argument 2 must contain current guardian size set
* Note field must contain signed digest.
* Passed guardian keys $[k..j]$ must match the current global state.
* Passed guardian size set must match the current global state.
* Last TX in group triggers VAA processing according to fields (e.g: do governance chores, unpack Pyth price ticker, etc)

**VAA Structure**

VAA structure is defined in: 
 https://github.com/certusone/wormhole/blob/dev.v2/whitepapers/0001_generic_message_passing.md

 Governance VAAs:
 https://github.com/certusone/wormhole/blob/dev.v2/whitepapers/0002_governance_messaging.md

 Sample Ethereum Struct Reference: 
 https://github.com/certusone/wormhole/blob/dev.v2/ethereum/contracts/Structs.sol

```
 VAA
 i Bytes        Field   
 0 1            Version
 1 4            GuardianSetIndex
 5 1            LenSignatures (LN)
 6 66*LN        Signatures where each S = { guardianIndex (1),r(32),s(32),v(1) }
 -------------------------------------< hashed/signed body starts here.
 4            timestamp
 4            Nonce
 2            emitterChainId
 32           emitterAddress
 8            sequence
 1            consistencyLevel
 N            payload
 --------------------------------------< hashed/signed body ends here.
```
**VAA Commitment**

Each VAA is uniquely identified by tuple (emitter_chain_id, emitter_address, sequence). We are currently interested in VAAs for:

* Governance operations:
    * Upgrade guardian set
    * Upgrade contract [this is necessary for non-publishers?]

* Pyth Ticker Data

After all signatures are verified the stateful app will execute the code to handle the VAA according to the tuple fields. 



## System Overview (proof-of-concept work)

:warning: You can consider this a first proof-of-concept design, and in terms of our current approach, an obsolete technique.

This flow uses a validator to sign the messages when they arrive. This trusts the price feed, so this is not recommended for production purposes. It may be used as a base design for other data flows. **See the Wormhole Flow below for the verified, cryptographically safe and trustless approach**.

The Pricecaster backend can be configured with any class implementing **IPriceFetcher** and **IPublisher** interfaces. The following diagram shows the service operating with a fetcher from ![Pyth Network](https://pyth.network/), feeding the Algorand chain through the `StdAlgoPublisher` class.

![PRICECASTER](https://user-images.githubusercontent.com/4740613/136037362-bed34a49-6b83-42e1-821d-1df3d9a41477.png)

### Input Message Format

The TEAL contract expects a fixed-length message consisting of:

```
  Field size
  9           header      Literal "PRICEDATA"
  1           version     int8 (Must be 1)
  8           dest        This appId 
  16          symbol      String padded with spaces e.g ("ALGO/USD        ")
  8           price       Price. 64bit integer.
  8           priceexp    Price exponent. Interpret as two-compliment, Big-Endian 64bit
  8           conf        Confidence (stdev). 64bit integer. 
  8           slot        Valid-slot of this aggregate price.
  8           ts          timestamp of this price submitted by PriceFetcher service
  32          s           Signature s-component
  32          r           Signature r-component 

  Size: 138 bytes. 
```

### Global state

The global state that is mantained by the contract consists of the following fields:

```
sym      : byte[] Symbol to keep price for   
vaddr    : byte[] Validator account          
price    : uint64 current price 
stdev    : uint64 current confidence (standard deviation)
slot     : uint64 slot of this onchain publication
exp      : byte[] exponent. Interpret as two-compliment, Big-Endian 64bit
ts       : uint64 last timestamp
```

#### Price parsing

The exponent is stored as a byte array containing a signed, two-complement 64-bit Big-Endian integer, as some networks like Pyth publish negative values here. For example, to parse the byte array from JS:

```
    const stExp = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'exp')
    const bufExp = Buffer.from(stExp, 'base64')
    const val = bufExp.readBigInt64BE()
```

## Installation

Prepare all Node packages with:

```
npm install
```

## Deployment of Applications

Use the deployment tools in `tools` subdirectory.

* To deploy the proof-of-concept "Pricekeeper" system, use the `deploy` tool with proper arguments, and later point the settings file to the deployed Appid.

* To deploy the VAA processor to use with Wormhole, make sure you have Python environment running (preferably >=3.7.0), and `pyteal` installed with `pip3`.

For example, using `deploy-wh` with sample output: 

```
node tools\deploy-wh.js tools\v1.prototxt.testnet 1000  OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU testnet

VAA Processor for Wormhole Deployment Tool -- (c)2021-22 Randlabs, Inc.
-----------------------------------------------------------------------

Parameters for deployment:
From: OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU
Network: testnet
Guardian expiration time: 1000
Guardian Keys: (19) 13947Bd48b18E53fdAeEe77F3473391aC727C638,F18AbBac073741DD0F002147B735Ff642f3D113F,9925A94DC043D0803f8ef502D2dB15cAc9e02D76,9e4EC2D92af8602bCE74a27F99A836f93C4a31E4,9C40c4052A3092AfB8C99B985fcDfB586Ed19c98,B86020cF1262AA4dd5572Af76923E271169a2CA7,1937617fE1eD801fBa14Bd8BB9EDEcBA7A942FFe,9475b8D45DdE53614d92c779787C27fE2ef68752,15A53B22c28AbC7B108612146B6aAa4a537bA305,63842657C7aC7e37B04FBE76b8c54EFe014D04E1,948ca1bBF4B858DF1A505b4C69c5c61bD95A12Bd,A6923e2259F8B5541eD18e410b8DdEE618337ff0,F678Daf4b7f2789AA88A081618Aa966D6a39e064,8cF31021838A8B3fFA43a71a50609877846f9E6d,eB15bCF2ae4f957012330B4741ecE3242De96184,cc3766a03e4faec44Bda7a46D9Ea2A9D124e9Bf8,841f499Ba89a6a8E9dD273BAd82Beb175094E5d7,f5F2b82576e6CA17965dee853d08bbB471FA2433,2bC2B1204599D4cA0d4Dde4a658a42c4dD13103a

Enter YES to confirm parameters, anything else to abort. YES

Enter mnemonic for sender account.
BE SURE TO DO THIS FROM A SECURED SYSTEM
.
.
.
Compiling VAA Processor program code...
,VAA Processor Program, (c) 2021-22 Randlabs Inc.
Compiling approval program...
Written to teal/wormhole/build/vaa-processor-approval.teal
Compiling clear state program...
Written to teal/wormhole/build/vaa-processor-clear.teal
,
Creating new app...
txId: DX7YIQ6L5QELSNZHJGKSZ4MQA7U26KJCPUJ42UFEGU22MJWDLY5Q
Deployment App Id: 43816461
Bye.
```

## Backend Configuration

The backend will read configuration from a `settings.ts` file pointed by the `PRICECASTER_SETTINGS` environment variable.  

## Running the system

Check the `package.json` file for `npm run tart-xxx`  automated commands. 

## Tests

Tests can be run for the old `Pricekeeper` contract, and for the new set of Wormhole client contracts:

`npm run pkeeper-sc-test`

`npm run wormhole-sc-test`

Backend tests will come shortly.

## Appendix

### Sample Pyth VAA

This is a sample signed VAA from Pyth that we process.

**Base64**
```
AQAAAAABAFv4FwzmQ+mPX0PYbc4TC5rX/z0B5OxZSJ80YZyjJN+CZLespNQSyq/qJHqvqjbM09AoCYQCzFv5oz9Sv8hnwaYBYaX/mgAACFkAATr9qEHB9D3X1UbIpYG6H5KhOfQTP59qsJVVj2o1nfXUAAAAAAAAABIgUDJXSAABASMKv+DsO0YL1V/E+zY1ZxYymRUUVJcgK464vxr2oKO5/mUPA2fUp++YFaWT6hXTZZPwZDqq8BSbsEvmerhR3s0BAAAALxclQ4j////3AAAALu1z2QAAAAAAcNO0PwAAAAA3+qA9AAAAAA6eVVEAAAAAiUrxHAAAAAA3+qA9AAAAAA3abrgBAAAAAABhpf+a
```

**Hex-Decoded**
```
010000000001005bf8170ce643e98f5f43d86dce130b9ad7ff3d01e4ec59489f34619ca324df8264b7aca4d412caafea247aafaa36ccd3d028098402cc5bf9a33f52bfc867c1a60161a5ff9a0000085900013afda841c1f43dd7d546c8a581ba1f92a139f4133f9f6ab095558f6a359df5d400000000000000122050325748000101230abfe0ec3b460bd55fc4fb36356716329915145497202b8eb8bf1af6a0a3b9fe650f0367d4a7ef9815a593ea15d36593f0643aaaf0149bb04be67ab851decd010000002f17254388fffffff70000002eed73d9000000000070d3b43f0000000037faa03d000000000e9e555100000000894af11c0000000037faa03d000000000dda6eb801000000000061a5ff9a
```

**Field-Decoded**
```
01                version
00000000          guardian-set-index
01                signature-count
00                sig index 0
5bf8170ce643e98f5f43d86dce130b9ad7ff3d01e4ec59489f34619ca324df8264b7aca4d412caafea247aafaa36ccd3d028098402cc5bf9a33f52bfc867c1a601   sig 0
61a5ff9a          timestamp
00000859          nonce 
0001              chain-id
3afda841c1f43dd7d546c8a581ba1f92a139f4133f9f6ab095558f6a359df5d4      emitter-address
0000000000000012  sequence
20                consistency-level

payload:

503257480001      header
01                payload-id
230abfe0ec3b460bd55fc4fb36356716329915145497202b8eb8bf1af6a0a3b9      product_id
fe650f0367d4a7ef9815a593ea15d36593f0643aaaf0149bb04be67ab851decd      price_id
01                price_type
0000002f17254388  price
fffffff7          exponent
0000002eed73d900  twap value
0000000070d3b43f  twap numerator for next upd
0000000037faa03d  twap denom for next upd
000000000e9e5551  twac value
00000000894af11c  twac numerator for next upd
0000000037faa03d  twac denom for next upd
000000000dda6eb8  confidence
01                status
00                corporate_act
0000000061a5ff9a  timestamp (based on Solana contract call time)
```

