# Integrity Checkers

Wormchain Integrity Checkers are smart contracts deployed on Wormchain. They allow  xApp developers to validate the integrity of cross-chain messages based on a globally synchronized state. This document describes the motivation, goals, and technical design.

## Objectives

- Allow xApps to implement safety checks on Wormchain that have access to the global, synchronized, state of the xApp.
- Integrity Checkers on Wormchain should be trustless. This means they can block but not create or modify messages.

## Background

xApps generally have one of two architectures:

1. Hub-and-spoke: A core contract is deployed on a main chain and has the global view of the cross-chain application. On each connected chain, there is a smaller contract, allowing for interaction with the core contract. This architecture is simple, but the main disadvantages are increased latency and increased cost when interacting between two chains that are not the hub. Example: ICCO.
2. Distributed: Each contract is the peer of multiple contracts on other chains and directly sends/receives messages between them. The main advantages are no reliance on a single chain, lower latency, and transaction fees can be lower. The main downside is the difficulty of synchronizing state between all chains. Example: Portal Token Bridge.

These two architectures drastically differ in their trust model: In the hub-and-spoke model, the contract on the hub chain has access to the global state of the xApp and can enforce global invariants. For example, a token bridge implemented in the hub-and-spoke architecture could enforce that a chain cannot send more wrapped tokens than have been deposited.

Conversely, the lack of synchronized state in the distributed xApp model often leads to the xApp trusting all connected chains. These chains may have different trust models and security properties, which could lead to the xApp relying on the weakest link. For example, a token bridge implemented in the decentralized architecture could be drained if a single chain has a fault.

## Goals

### Primary goals

- Motivate the concept of integrity checkers

### Out of scope (for now)

- Registration of Integrity Checkers: For now, Wormchain will be restricted to trusted Integrity Checkers.
- Gas cost: For now, executing the checkers on Wormchain will be free.

## Overview

After making an observation, a Guardian checks if there are integrity checkers configured for the emitter. If there are, it submits a pre-observation to the integrity-checker smart contract on Wormchain. It then saves the pre-observation to a local database.

After its conditions are met, an integrity-checker approves the wormhole message it deems to be valid.

The Guardian picks up the messages approved from the integrity-checkers on Wormchain when they correspond to an observation the Guardian has made itself. It signs and broadcasts the signature to the Guardian peer to peer network. Thereby, an integrity-checker is unable to modify or inject messages and can only block messages.

## Terminology

* `Pre-Observation` - used to designate an observation being submitted to the integrity checker contract by a Wormhole Guardian. They are similar to observations, i.e. a Wormhole message signed by a single guardian, but can be distinguished by their prefix and signature format. They follow the same signature format as signed gossip messages described in [guardian key usage](0009_guardian_key.md) with a unique signature prefix.  Signed pre-observations therefore cannot be used like signed observations to create a VAA.
* `Batching` - pre-observations accumulate over a configurable time period and are batch submitted to Wormchain. This both saves on gas costs and results in less computational overhead.
* `Persistence` - the Guardian persists the status of pre-observations to the local database.
* `Retry` - the Guardian periodically watches the Wormchain state and ensures that all local pre-observations have been submitted correctly. In case of an error, it retries the submission.
