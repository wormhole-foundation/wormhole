# Wormhole Protocol

The Wormhole protocol is a way of transferring assets between a **root chain** and multiple **foreign chains**.
Therefor it makes use of decentralized oracles called **guardians** to relay transfer information about token transfers
between the chains.

## The role of guardians

Guardians are responsible for monitoring the root and foreign chains for token transfers to bridge *smart contracts*.
This can be done using full or light clients of the particular network.
They need to make sure to monitor finality of transactions (e.g. track number of confirmations) before relaying messages.

A guardian is identified by an **admin key** and **voter key**.

The **admin key** is supposed to be held in cold-storage and is used to manage rewards and assign a signer key.

The **signer key** is a hot-key that is used to confirm asset transfers between chains by reporting lockups of tokens
on a foreign chain on the root chain or the other way around.

## Protocol

The following section describes the protocol and design decisions made.

### Signature scheme

In order to implement a trustless bridge, there needs to be a consensus mechanism to measure whether there is a quorum
on a cross chain transfer to prevent a single malicious actor from unlocking or minting an infinite amount of assets.

There are multiple ways to measure whether enough validators have approved a decision:

#### Multiple signatures - MultiSig

The most simple solution is by using a *MultiSig* mechanism. This means that each guardian would sign a message 
and submit it to a smart contract on-chain with reference to a *decision* that the guardians need to make (e.g. a transfer).
Since a transaction itself is already signed, we can simplify this to using the transaction itself as proof.

Said smart contract will count the number of guardians that have submitted a transaction for a *decision*.
Once the consensus threshold has been reached, the contract will execute the action the guardians have agreed on.

The issue with this schema is that it requires at least `n=2/3*m+1` transactions for `m` validators. On Ethereum for
example one such transaction would cost `21k+20k+x` gas (base + `SSTORE` \[to track the tx] + additional compute).
With `n` txs and 20 guardians threshold (`2/3m+1`) the cost would be `n*(41k+x)` which is `820k+20x`.

At a gas price of `50 Gwei` this would mean total tx costs of `0.041 ETH` at `x=0`. At an ETH price of `300$` that
means costs of `12.3$`.

These prices will require the guardians to charge significant fees. If these fees are not covered by the user, bridge
transactions would stall and time out. 

There are a couple of other issues with this concept:

1. There is no way for the Solana Bridge program to verify whether the guardians have actually unlocked the tokens on
the foreign chain.
2. Users cannot cover gas costs themselves because transactions are not "portable". I.e. the require serialized nonces.
If a guardian submits a transaction with nonce 20 to the user but in the meantime issues another transaction with the 
same nonce, the user tx will be invalid even though the Solana program might successfully verify the tx (as it does not
know the state of ETH).

There is an alternative way by using portable ECDSA signatures that approve an action i.e. a transfer. The guardians
could submit all of those signatures to the lock proposal and the user or another participant in the network could relay
them to Ethereum.
That way the Solana program can verify that the signatures and signed action are valid, being sure that if there is a 
quorum (i.e. enough signatures), the user could use these signatures to trigger the execution of the signed action on
the foreign chain.

The downside here is that this makes tracking and synchronizing guardian changes highly complex and further increases
gas costs by about `(5k+5k)*n` (`ECRECOVER+GTXDATANONZERO*72`) for the additional `ecrecover` calls that need to be made.
However since all signatures can be aggregate into one tx, we'll save `(n-1)*21k` leading to an effective gas saving of
`~10k*n`. Still, transfers would be considerably expensive applying the aforementioned assumptions.

#### Threshold signatures

Most of the disadvantages of the MultiSig solution come down to the high gas costs of verifying multiple transactions
and tracking individual guardian key changes / set changes on other chains.

In order to prove a quorum on a single signature, there exist different mechanisms for so-called Threshold signatures.
A single signature is generated using a multi party computation process or aggregation of signatures from different
parties of a group and only valid if a previously specified quorum has participated in the generation of such signature.

This would essentially mean that such a signature could be published on the Solana chain and relayed by anyone to 
authorize an action on another chain, the same concept as described above but implemented with the cost of only 
sending and verifying one signature.

Since we target Ethereum as primary foreign chain, there are 3 viable options of threshold signatures:

**t-ECDSA**

Threshold ECDSA signatures generated using [GG20](https://eprint.iacr.org/2020/540.pdf).
This is a highly complex, cutting edge cryptographic protocol that requires significant amounts of compute to generate
signatures with larger quorums.

Still, it generates plain ECDSA signatures that can easily be verified on Ethereum (`5k gas`) or even be used for Bitcoin
transactions.

**BLS**

Boneh–Lynn–Shacham threshold signatures are very lightweight because they don't require a multi-round process and can
simply be aggregated from multiple individual signatures. This would eliminate the need for a p2p layer for MPC
communication.
However, verifying a BLS signature on Ethereum costs about 130k gas using the precompiled pairing functions over bn128.
Also there's very little prior work on this scheme especially in the context of Solidity.

**Schnorr-Threshold**

Schnorr threshold signatures require a multi-round computation and distributed key generation.
They can be verified on Ethereum extremely cheaply (https://blog.chain.link/threshold-signatures-in-chainlink/) and scale
well with more signing parties.
There's been significant prior work in the blockchain space, several implementations over different curves and a proposal
to implement support on Bitcoin (BIP340).

---

A great overview can be found [here](https://github.com/Turing-Chain/TSSKit-Threshold-Signature-Scheme-Toolkit)

#### Design choices

Most of the downsides of MultiSig are limited to foreign chains due to Solana being substantially faster and 
cheaper. We'll therefore use a multisig schema on Solana to verify transfers from foreign chains => Solana. This keeps
the disadvantage (1). Optionally we can add a feature to use the foreign chain for data availability on 
foreign chain => Solana transfers and allow the user to reclaim tokens if no VAA (which could be used to claim tokens
on Solana) is published by the guardians.

For transfers to foreign chain we'll implement a Schnorr-Threshold signature schema based on the implementation from 
Chainlink. We'll create a portable "action blob" with a threshold signature to allow anyone to relay action approvals
between chains. We call this structure: **VAA** (Validator Action Approval).

A validator action approval leads to information symmetry i.e. if the validators have submitted a VAA to a token lockup
on Solana, this VAA can be used to unlock the tokens on the specified foreign chain, it also proves to the Solana chain
that the lockup is not refundable as it can provably be claimed (as long as safety guarantees are not broken and except
for the case of a guardian set change which is discussed later).

### VAA - Validator Action Approval

Validator action approvals are used to approve the execution of a specified action on a chain.

They are structured as follows:

```
Header:
uint8               Version (0x01)
[72]uint8           signature(body)

body:
uint32              Validator set index
uint32              Unix seconds
uint8               Action
uint8               payload_size
[payload_size]uint8 payload
```


#### Actions

##### Guardian set update

ID: `0x01`

Size: `32 byte`

Payload:

```
[32]uint8 new_key
```

##### Solana (wrapped) -> Ethereum (native)

ID: `0x10`

Size: `72 byte`

Payload:

```
[20]uint8 target_address
[20]uint8 token_address
uint256 amount
```

##### Ethereum (native) -> Solana (wrapped)

ID: `0x12`

Size: `84 byte`

Payload:

```
[32]uint8 target_address
[20]uint8 token_address
uint256 amount
```

##### Solana (native) -> Ethereum (wrapped)

ID: `0x12`

Size: `84 byte`

Payload:

```
[20]uint8 target_address
[32]uint8 token_address
uint256 amount
```

##### Ethereum (wrapped) -> Solana (native)

ID: `0x13`

Size: `96 byte`

Payload:

```
[32]uint8 target_address
[32]uint8 token
uint256 amount
```

### Cross-Chain Transfers

#### Transfer of assets Foreign Chain -> Root Chain

The user sends a chain native asset to the bridge on the foreign chain using the `Lock` function.
The lock function takes a Solana `address` as parameter which is the TokenAccount that should receive the wrapped token.

Guardians will pick up the *Lock transaction* once it has enough confirmations on the foreign chain. The amount of 
confirmations required is a parameter that guardians can specify individually.

They check for the validity, parse it and will then send a `ConfirmForeignLockup` transaction to the Solana program
testifying that they have seen a foreign lockup. Once the quorum has been reached, a new wrapped asset will be minted or
released from custody. Custody is used for Solana-native tokens that have previously been transferred to a foreign 
chain, minting will be used to create new units of a wrapped foreign-chain asset.

If this is the first time a foreign asset is minted, a new **Mint** (token) will be created on quorum.

### Transfer of assets Root Chain -> Foreign Chain

The user sends a **Lock** or **LockNative** instruction to the *Bridge program*.

**Lock** has to be used for wrapped assets that should be transferred to a foreign chain. They will be burned on Solana.

**LockNative** has to be used for Solana-native assets that should be transferred to a foreign chain. They will be held
in a custody account until the tokens are transferred back from the foreign chain.

The lock function takes a `chain_id` which identifies the foreign chain the tokens should be sent to and a `foreign_address`
which is a left-zero-padded address on the foreign chain. This operation creates a **LockProposal** account
that tracks the status of the transfer.

Guardians will pick up the **LockProposal** once it has enough confirmations on the Solana network. It defaults to
full confirmation (i.e. the max lockup, currently 32 slots), but can be changed to a different commitment levels
on each guardian's discretion.

They check for the validity of the tx, parse it and will initiate an off-chain threshold signature ceremony which will
output a **VAA** that can be used with a foreign chain smart contract to reclaim an unwrapped local asset or mint a 
wrapped `spl-token`.

This VAA will be posted on Solana by one of the guardians using the `PostVAA` instruction and will be stored in the
`LockProposal`.

Depending on whether the fees are sufficient for **guardians** or **relayers** to cover the foreign chain fees, they
will also post the VAA on the foreign chain, completing the transfer.

If no fee or an insufficient fee is specified, the user can pick up the VAA from the `LockProposal` and submit it on the foreign chain themselves.

VAAs for conducting transfers to a foreign chain are submitted using `FinalizeTransfer`.

### Fees

TODO  \o/

### Config changes
#### Guardian set changes

Since we use a *TSS* (Threshold signature scheme) for VAAs, changes to the guardian list are finalized by setting a
new aggregate public key that's derived from a distributed key generation ("DKG") ceremony of the new guardian set.

This new public key is set via a VAA with the `UPDATE_GUARDIANS` action that is signed by the previous guardians.

The guardians need to make sure that the sets are synchronized between all chains.
If the guardian set is changed, the guardian must also be replaced on all foreign chains. Therefore we
conduct these changes via VAAs that are universally valid on all chains.

That way, if a change is made on the root chain, the same signatures can be used to trigger the same
update on the foreign chain. This allows all parties in the system to propagate bridge state changes across all
chains.

If all VAAs issued by the previous guardian set would immediately become invalid once a new guardian set takes over, that would
lead to some payments being "stuck". Therefore we track a list of previous guardian sets. VAAs issued by old 
guardian sets stay valid for one day from the time that the change happens.
