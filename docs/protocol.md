# Wormhole Protocol

The Wormhole protocol is a way of transferring assets between a **root chain** and multiple **foreign chains**.
It makes use of decentralized oracles called **guardians** to relay transfer information about token transfers
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
and submit it via a P2P gossip network.

Once the consensus threshold has been reached, a guardian will aggregate all signatures into a VAA and execute/submit it
on the chain.

The downside here is that gas costs increase with larger guardian sets bringing verification costs to
 `(5k+5k)*n` (`ECRECOVER+GTXDATANONZERO*72`).
 
To prevent lagging and complex gas price handling by validators or relayers, we always submit VAAs to Solana where txs
are negligibly cheap. In the case of a Solana -> ETH transfer. Guardians would publish a signed VAA on Solana and a user
or independently paid relayer would publish said VAA on Ethereum, paying for gas costs. This mechanism is similar to a 
check issued by the guardians (a VAA) which can be used on another chain to claim assets.

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

For transfers we implement a simple MultiSig schema.
We'll create a portable "action blob" with a threshold signature to allow anyone to relay action approvals
between chains. We call this structure: **VAA** (Verifiable Action Approval).

A validator action approval guarantees eventual consistency across chains - if the validators have submitted a VAA to a token lockup
on Solana, this VAA can be used to unlock the tokens on the specified foreign chain.

While for the above mentioned transfers from Solana => foreign chain we use Solana for data availability of the VAAs, 
in the other direction data availability i.e. the guardians posting the VAA on the foreign chain (where the transfer
was initiated) is optional because in most cases it will be substantially cheaper for the guardians to directly submit
the VAA on Solana itself to unlock/mint the transferred tokens there.

### VAA - Verifiable Action Approval

Verifiable action approvals are used to approve the execution of a specified action on a chain.

They are structured as follows:

```
Header:
uint8               version (0x01)
uint32              guardian set index
uint8               len signatures

per signature:
uint8               index of the signer (in guardian keys)
[65]uint8           signature

body:
uint32              unix seconds
uint8               action
[payload_size]uint8 payload
```

The `guardian set index` does not need to be in the signed body since it is verifiable using the signature itself which
is created using the guardian set's key.
It is a monotonically number that's increased every time a validator set update happens and tracks the public key of the
set.

#### Actions

##### Guardian set update

ID: `0x01`

Payload:

```
uint32 new_index
uint8 len(keys)
[][20]uint8 guardian addresses
```

The `new_index` must be monotonically increasing and is manually specified here to fix a potential guardian_set index 
desynchronization between the any of the chains in the system.

##### Transfer

ID: `0x10`

Payload:

```
uint32 nonce
uint8 source_chain
uint8 target_chain
[32]uint8 source_address
[32]uint8 target_address
uint8 token_chain
[32]uint8 token_address
uint256 amount
```

### Cross-Chain Transfers

#### Transfer of assets Foreign Chain -> Root Chain

If this is the first time the asset is transferred to the root chain, the user inititates a `CreateWrapped` instruction
on the root chain to initialize the wrapped asset. 

The user creates a token account for the wrapped asset on the root chain.

The user sends a chain native asset to the bridge on the foreign chain using the `Lock` function.
The lock function takes a Solana `address` as parameter which is the TokenAccount that should receive the wrapped token.

Guardians will pick up the *Lock transaction* once it has enough confirmations on the foreign chain. The amount of 
confirmations required is a parameter that guardians can specify individually.

They check for the validity, parse it and will then initiate a threshold signature ceremony on a deterministically 
produced VAA (`Transfer`) testifying that they have seen a foreign lockup. They will post this VAA on the root chain
using the `SubmitVAA` instruction.
 
This instruction will either mint a new wrapped asset or release tokens from custody. 
Custody is used for Solana-native tokens that have previously been transferred to a foreign chain, minting will be used
 to create new units of a wrapped foreign-chain asset.

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

They check for the validity of the tx, parse it and will initiate an off-chain signature aggregation ceremony which will
output a **VAA** that can be used with a foreign chain smart contract to reclaim an unwrapped local asset or mint a 
wrapped `spl-token`.

This VAA will be posted on Solana by one of the guardians using the `SubmitVAA` instruction and will be stored in the
`LockProposal`.

The user can then get the VAA from the `LockProposal` and submit it on the foreign chain.

### Fees

TODO  \o/

### Config changes
#### Guardian set changes

The guardians need to make sure that the sets are synchronized between all chains.
If the guardian set is changed, the guardian must also be replaced on all foreign chains. Therefore we
conduct these changes via VAAs that are universally valid on all chains.

That way, if a change is made on the root chain, the same signatures can be used to trigger the same
update on the foreign chain. This allows all parties in the system to propagate bridge state changes across all
chains.

If all VAAs issued by the previous guardian set would immediately become invalid once a new guardian set takes over, that would
lead to some payments being "stuck". Therefore we track a list of previous guardian sets. VAAs issued by old 
guardian sets stay valid for one day from the time that the change happens in the default configuration.
