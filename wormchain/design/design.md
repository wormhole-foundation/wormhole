# Table of Contents

1.  [Inbox](#org155bf00)
    1.  [Bootstrap chain](#org819971b)
    2.  [Onboarding guardians](#org60d7dc9)

<a id="org155bf00"></a>

# Inbox

<a id="org819971b"></a>

## TODO Bootstrap chain

The native token of the Wormhole chain is $WORM. This token is used both for
staking (governance) and fees. These tokens are already minted on Solana, and
they won't be available initially at the genesis of the chain. This presents
a number of difficulties around bootstrapping.

At genesis, the blockchain will be set up in the following way

1.  The staking denom is set to the $WORM token (of which 0 exist on this chain at this moment)
2.  Producing blocks uses Proof of Authority (PoA) consensus (i.e. no tokens are required to produce blocks)
3.  Fees are set to 0

Then, the $WORM tokens can be transferred over from Solana, and staking (with
delegation) can be done. At this stage, two different consensus mechanisms will
be in place simultaneously: block validation and guardian set election will
still use PoA, with each guardian having a singular vote. All other governance
votes will reach consensus with DPoS by staking $WORM tokens.

<a id="org60d7dc9"></a>

## TODO Onboarding guardians

The validators of wormhole chain are going to be the 19 guardians. We need a
way to connect their existing guardian public keys with their wormhole chain
addresses. We will have a registration process where a validator can register a
guardian public key to their validator address. This will entail
signing their wormhole address with their guardian private key, and sending
that signature from their wormhole address. At this point, if the signature
matches, the wormhole address becomes associated with the guardian public key.

After this, the guardian is eligible to become a validator.

Wormhole chain uses the ECDSA secp256k1 signature scheme, which is the same as what
the guardian signatures use, so we could directly derive a wormhole account for
them, but we choose not to do this in order to allow guardian key rotation.

    priv = ... // guardian private key
    addr = sdk.AccAddress(priv.PubKey().Address())

In theory it is possible to have multiple active guardian sets simultaneously
(e.g. during the expiration period of the previous set). We only want one set of
guardians to be able to produce blocks, so we store the latest validator set
(which should typically by a pointer to the most recent guardian set). We have to
be careful here, because if we update the guardian set to a new set where a
superminority of guardians are not online yet, they won't be able to register
themselves after the switch, since block production will come to a halt, and the
chain becomes deadlocked.

Thus we must only change over block production due to a guardian set update if a supermajority of guardians
in the new guardian set are already registered.

At present, Guardian Set upgrade VAAs are signed by the Guardians off-chain. This can stay off-chain for as long as needed, but should eventually be moved on-chain.

## TODO Bootstrapping the PoA Network

At time of writing, the Guardian Network is currently at Guardian Set 2, but will possibly be at set 3 or 4 by the time of launch.

It is likely not feasible to launch the chain with all 19 Guardians of the network hardcoded in the genesis block, as this would require the Guardians to determine their addresses off-chain, and have their information encoded in the genesis block.

As such, it is likely simpler to launch Wormhole Chain with a single validator (The guardian from Guardian Set 1), then have all the other Guardians perform real on-chain registrations for themselves, and then perform a Guardian Set upgrade directly to the current Guardian set.
