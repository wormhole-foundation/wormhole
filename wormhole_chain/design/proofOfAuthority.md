# Wormhole Chain PoA Architecture Design

The Wormhole Chain is intended to operate via the same PoA mechanism as the rest of the Wormhole ecosystem. This entails the following:

- Two thirds of the Consensus Guardian Set are required for consensus. (In this case, block production.)
- Guardian Sets are upgraded via processing Guardian Set Upgrade Governance VAAs.

As such, the intent is that the 19 guardians will validate for Wormhole Chain, and Wormhole Chain consensus will be achieved when 13 Guardians vote to approve a block (via Tendermint). This means that we will need to hand-roll a PoA mechanism in the Cosmos-SDK on top of Tendermint and the normal Cosmos Staking module.

## High-Level PoA Design Overview

At any given time in the Wormhole Network, there is an "Latest Guardian Set". This is defined as the highest index Guardian Set, and is relevant outside of Wormhole Chain as well. The 'Latest Guardian Set' is meant to be the group of Guardians which is currently signing VAAs.

Because the Guardian keys are meant to sign VAAs, and not to produce blocks on a Cosmos blockchain, the Guardians will have to separately host validators for Wormhole Chain, with different addresses, and then associate the addresses of their validator nodes to their Guardian public keys.

Once an association has been created between the validator and its Guardian Key, the validator will be considered a 'Guardian Validator', and will be awarded 1 consensus voting power. The total voting power is equal to the size of the Consensus Guardian Set, and at least two thirds of the total voting power must vote to create a block.

The Consensus Guardian Set is a distinct term from the Latest Guardian Set. This is because the Guardian Set Upgrade VAA is submitted via a normal Wormhole Chain transaction. When a new Latest Guardian Set is created, many of the Guardians in the new set may not have yet registered as Guardian Validators. Thus, the older Guardian Set must remain marked as the Consensus Set until enough Guardians from the new set have registered.

## Validator Registration:

First, validators must be able to join the Wormhole Chain Tendermint network. Validator registration is identical to the stock Cosmos-SDK design. Validators may bond and unbond as they would for any other Cosmos Chain. However, all validators have 0 consensus voting power, unless they are registered as a Guardian Validator, wherein they will have 1 voting power.

## Mapping Validators to Guardians:

Bonded Validators may register as Guardian Validators by submitting a transaction on-chain. This requires the following criteria:

- The validator must be bonded.
- The validator must hash their Validator Address (Operator Address), sign it with one of the Guardian Keys from the Latest Guardian Set (Note: Latest set, not necessarily Consensus Set.), and then submit this signature in a transaction to the RegisterValidatorAsGuardian function.
- The transaction must be signed/sent from the Validator Address.
- The validator must not have already registered as a different Guardian from the same set.

A Guardian Public Key may only be registered to a single validator at a time. If a new validator proof is received for an existing Guardian Validator, the previous entry is overwritten. As an optional defense mechanism, the registration proofs could be limited to only Guardian Keys in the Latest set.

## Guardian Set Upgrades

Guardian Set upgrades are the trickiest operation to handle. When processing the Guardian Set Upgrade, the following steps happen:

- The Latest Guardian Set is changed to the new Guardian Set.
- If all Guardian Keys in the new Latest Guardian Set are registered, the Latest Guardian Set automatically becomes the new Consensus Guardian Set. Otherwise, the Latest Guardian Set will not become the Consensus Guardian Set until this threshold is met.

## Benefits of this implementation:

- Adequately meets the requirement that Guardians are responsible for consensus and block production on Wormhole Chain.
- Relatively robust with regard to chain 'bricks'. If at any point in the life of Wormhole Chain less than 13 of the Guardians in the Consensus Set are registered, the network will deadlock. There will not be enough Guardians registered to produce a block, and because no blocks are being produced, no registrations can be completed. This design does not change the Consensus Set unless a sufficient amount of Guardians are registered.
- Can swap out a massive set of Guardians all at once. Many other (simpler) designs for Guardian set swaps limit the number of Guardians which can be changed at once to only 6 to avoid network deadlocks. This design does not have this problem.
- No modifications to Cosmos SDK validator bonding.

### Cons

- Moderate complexity. This is more complicated than the most straightforward implementations, but gains important features and protections to prevent deadlocks.
- Not 100% immune to deadlocks. If less than 13 Guardians have valid registrations, the chain will permanently halt. This is prohibitively difficult to prevent with on-chain mechanisms, and unlikely to occur. Performing a simple hard fork in the scenario of a maimed Guardian Validator set is likely the safer and simpler option.
- Avoids some DOS scenarios by only allowing validator registrations for known Guardian Keys.

## Terms & Phrases:

### Guardian

- One of the entities approved to sign VAAs on the Wormhole network. Guardians are identified by the public key which they use to sign VAAs.

### Guardian Set

- A collection of Guardians which at one time was approved by the Wormhole network to produce VAAs. These collections are identified by their sequential 'Set Index'.

### Latest Guardian Set

- The highest index Guardian Set.

### Consensus Guardian Set

- The Guardian Set which is currently being used to produce blocks on Wormhole Chain. May be different from the Latest Guardian Set.

### Guardian Set Upgrade VAA

- A Wormhole network VAA which specifies a new Guardian Set. Emitting a new Guardian Set Upgrade VAA is the mechanism which creates a new Guardian Set.

### Validator

- A node on Wormhole Chain which is connected to the Tendermint peer network.

### Guardian Validator

- A Validator which is currently registered against a Guardian Public Key in the Consensus Guardian Set
