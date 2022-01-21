
# Pricecaster Service V2

## Introduction

This service consumes prices from "price fetchers" and feeds blockchain publishers. 

The current implementation is a Wormhole client that uses the JS SDK to get VAAs from Pyth network and feed the payload and cryptographic verification data to a transaction group for validation. Subsequently, the data is optionally processed and stored, either price or metrics. For details regarding Wormhole VAAs see design documents: 

  https://github.com/certusone/wormhole/tree/dev.v2/whitepapers

## System Overview

**The objective is to receive signed messages -named as Verifiable Attestments (VAAs) in Wormhole jargon- from our relayer backend (Pricecaster) , verify them against a fixed (and upgradeable) set of "guardian public keys" and process them, publishing on-chain price information or doing governance chores depending on the VAA payload.**


The design is based in two contracts that work in tandem, a  **Stateful contract (VAA_Processor)** that accepts calls for verifying and commiting VAAs, and also mantains the global guardian set; and a **verifier stateless contract** that does the computational work of ECDSA signature verification.

Due to computation and space limits, the validation of the 19 guardian signatures against the payload is partitioned so each stateless contract validates a subset of the guardian signatures. If ECDSA decompress and validation opcodes are used, that yields 650+1750 = 2400 computation units * 7 = 16800, leaving 3200 free units for remaining opcodes.
In our design, We call **verification step** to each of the app calls + stateless logic involved  in verifying a block of signatures.

Keep in mind that *not all* the 19 signatures must be present in a VAA verification, but at least 1 + (2/3)  of the current guardian set.

The maximum number of signatures in each verification step is fixed at contract compilation stage, so with this in mind and example values:

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
|  3  | VAA consume call | N/A |

The current design requires the last call to be a call to an authorized application. This is intended to process VAA price data. The authorized appid must be set accordingly using the `setauthid` call in the VAA Processor contract after deployment.
If no call is going to be made, a dummy app call must be inserted in group for the transaction group to succeed.

To mantain the long-term transaction costs predictable, when not all signatures are provided but > TRUNC(N_S*2/3)+1, the number of transactions in the group does not change, but a transaction may have zero signatures as input, e.g for a VAA with 14 signatures:

| TX# | App calls | Stateless logic |
| --- | --------- | --------------- |
|  0  | _args_: guardian_pk[0..6], _txnote_: signed_digest          | _args_: sig[0..6]    |
|  1  | _args_: guardian_pk[7..13], _txnote_: signed_digest          | _args_: sig[7..13]   |
|  2  | _args_: guardian_pk[14..18], _txnote_: signed_digest          | _args_: **empty**    | 
|  3  | VAA consume call | N/A |

The backend will currently **call the Pricekeeper V2 contract to store data** as the last TX group. See below for details on how Pricekeeper works.

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
* Last TX in the verification step (total group size-1) triggers VAA processing according to fields (e.g: do governance chores, unpack Pyth price ticker, etc).  Last TX in the entire group must be an authorized application call.

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

## Pricekeeper V2 App

The Pricekeeper V2 App mantains a record of product/asset symbols (e.g ALGO/USD, BTC/USDT) and the price and metrics information associated. As the original Pyth Payload is 150-bytes long and it wouldn't fit in the key-value entry of the global state, the Pricekeeper contract slices the Pyth fields to a more compact format, discarding unneeded information.

The Pricekeeper V2 App will allow storage to succeed only if:

* Sender is the contract owner.
* Call is part of a group where all application calls are from the expected VAA processor Appid, 
* Call is part of a group where the verification slot has all bits set.

At deployment, the priceKeeper V2 contract must have the "vaapid" global field set accordingly.

Consumers must interpret the stored bytes as fields organized as:

```
Bytes
32              productId
32              priceId
8               price
1               price_type
4               exponent
8               twap value
8               twac value
8               confidence
8               timestamp (based on Solana contract call time)
```

## Installation

Prepare all Node packages with:

```
npm install
```

## Deployment of Applications

Use the deployment tools in `tools` subdirectory.

* To deploy the VAA processor  and Pricekeeper V2 app to use with Wormhole, make sure you have Python environment running (preferably >=3.7.0), and `pyteal` installed with `pip3`.  
* The deployment program will:  generate all TEAL files from PyTEAL sources, deploy the VAA Processor application, deploy the Pricekeeper V2 contract, compile the stateless program and set the correct parameters for the contracts: authid, vphash in VAA Processor and vaapid in the Pricekeeper app.

For example, using `deploy-wh` with sample output: 

```
$ node tools\deploy-wh.js tools\gkeys.test 1000  OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU testnet keys\owner.key

Pricecaster v2 Apps Deployment Tool
Copyright (c) Randlabs Inc,  2021-22

Parameters for deployment:
From: OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU
Network: testnet
Guardian expiration time: 1000
Guardian Keys: (1) 13947Bd48b18E53fdAeEe77F3473391aC727C638

Enter YES to confirm parameters, anything else to abort. YES
Compiling programs ...

,VAA Processor Program, (c) 2021-22 Randlabs Inc.
Compiling approval program...
Written to teal/wormhole/build/vaa-processor-approval.teal
Compiling clear state program...
Written to teal/wormhole/build/vaa-processor-clear.teal
,
,Pricekeeper V2 Program, (c) 2021-22 Randlabs Inc.
Compiling approval program...
Written to teal/wormhole/build/pricekeeper-v2-approval.teal
Compiling clear state program...
Written to teal/wormhole/build/pricekeeper-v2-clear.teal
,
Creating VAA Processor...
txId: WS7GE5A6YAADHVNH5OU337MK7T325AE2GML5S3RWK2VTNCQ23HWA
Deployment App Id: 52438261
Creating Pricekeeper V2...
txId: FICS3HFALLJTMFGEVC65IQ67NCYRJATR32QWZS5VMKGEXHBJJUVA
Deployment App Id: 52438280
Setting VAA Processor authid parameter...
txId: 5NVJGG32DRWAURD3LUHPELJAZTFMM6HLAJPPGNPXNDC5FJFDNVUQ
Compiling verify VAA stateless code...
,VAA Verify Stateless Program, (c) 2021-22 Randlabs Inc.
Compiling...
Written to teal/wormhole/build/vaa-verify.teal
,
Stateless program address:  KRNYKVVWZDCNOPLL63ZHFOKG2IIY7REBYTPVR5TJLD67JR6FMRJXYW63TI
Setting VAA Processor stateless code...
txId: 5NVJGG32DRWAURD3LUHPELJAZTFMM6HLAJPPGNPXNDC5FJFDNVUQ
Writing deployment results file DEPLOY-1639769594911...
Writing stateless code binary file VAA-VERIFY-1639769594911.BIN...
Bye.
```

* To operate, the stateless contract address must be supplied with funds to pay fees when submitting transactions.
* Use the generated `DEPLOY-XXX` file to set values in the `settings-worm.ts` file (or your current one): app ids and stateless hash.  
* Copy the generated `VAA-VERIFY-xxx`  file as `vaa-verify.bin` under the `bin` directory.

## Backend Configuration

The backend will read configuration from a `settings.ts` file pointed by the `PRICECASTER_SETTINGS` environment variable.  

### Diagnosing failed transactions

If a transaction fails, a diagnostic system is available where the group TX is dumped in a directory. To use this, set the relevant settings file:

```
  algo: {
    ...
    dumpFailedTx: true,
    dumpFailedTxDirectory: './dump'
  },
```

The dump directory will be filled with files named `failed-xxxx.stxn`.  You can use this file and `goal clerk` to trigger the stateless logic checks:

```
root@47d99e4cfffc:~/testnetwork/Node# goal clerk dryrun -t failed-1641324602942.stxn
tx[0] trace:
  1 intcblock 1 8 0 32 66 20 => <empty stack>
  9 bytecblock 0x => <empty stack>
 12 txn Fee => (1000 0x3e8)
 14 pushint 1000 => (1000 0x3e8)
 .
 . 
 .
 47 txn ApplicationID => (622608992 0x251c4260)
 49 pushint 596576475 => (596576475 0x238f08db)
 55 == => (0 0x0)
 56 assert =>
 56 assert failed pc=56

REJECT
ERROR: assert failed pc=56
```

In this example output, this means the logic failed due to mismatched stateful application id.


For a stateful run, you must do a remote dryrun.  This is done by:

```
goal clerk dryrun -t failed-1641324602942.stxn  --dryrun-dump -o dump.dr
goal clerk dryrun-remote -D dump.dr -v

```

## Running the system

Check the `package.json` file for `npm run tart-xxx`  automated commands. 

## Tests

Tests can be run for the old `Pricekeeper` contract, and for the new set of Wormhole client contracts:

`npm run pkeeper-sc-test`

`npm run wormhole-sc-test`

Backend tests will come shortly.

## Appendix

### Common errors

**TransactionPool.Remember: transaction XMGXHGC4GVEHQD2T7MZDKTFJWFRY5TFXX2WECCXBWTOZVHC7QLAA: overspend, account X**

If account X is the stateless program address, this means that this account is without enough balance to pay the fees for each TX group.


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

