# Integrity Checkers

Wormchain Integrity Checkers are smart contracts deployed on Wormchain that xApp developers can use to validate the integrity of cross-chain messages based on a globally synchronized state. This document describes the motivation, goals, and technical design.

## Objectives

- Allow xApps to implement safety checks on Wormchain that have access to the global, synchronized, state of the xApp.
- Integrity Checkers on Wormchain should be trustless, meaning that they can block but not create or modify messages.

## Background

xApps generally have one of two architectures:

1. Hub-and-spoke: A core contract is deployed on a main chain and has the global view of the cross-chain application. On each connected chain, there is a smaller contract, allowing for interaction with the core contract. This architecture is simple, but the main advantages are increased latency and increased cost when you want interaction between two chains that are not the hub. Example: ICCO.
2. Distributed: Each contract is the peer of multiple contracts on other chains and directly receives and sends messages to/from them. The main advantages are that there is no reliance on a single chain, the xApp can utilize the strengths of each chain, and latency and transaction fees can be lower. The main downside is that it is difficult to synchronize state between all the chains. Example: Portal Token Bridge.

These two architectures drastically differ in their trust model: In the hub-and-spoke model, the contract on the hub chain has access to the global state of the xApp and can enforce global invariants of the protocol. For example, a token bridge implemented in the hub-and-spoke architecture could enforce that a chain cannot send more wrapped tokens that have been deposited there.

Conversely, the lack of synchronized state in the distributed xApp model often leads to the xApp trusting all connected chains. These chains may have different trust models and security properties, which could lead to the xApp relying on the weakest link. For example, a token bridge implemented in the decentralized architecture could be drained if a single chain has a fault.

## Goals

### Primary goals

- Motivate the concept of integrity checkers

### Out of scope (for now)

- Registration of Integrity Checkers: For now, Wormchain will be restricted to trusted Integrity Checkers.
- Gas cost: For now, executing the checkers on Wormchain will be free.

## Overview

After a Guardian makes an observation, it checks if there is a integrity checkers configured for the emitter. If there are, it will submit a pre-observation to the integrity-checker smart contract on Wormchain and save it to a local database.

The integrity-checker can do whatever it wants. For example, it could first wait for the message to reach quorum by multiple guardians, then perform some checks, etc.

Eventually, integrity-checker emits the wormhole message if it deems it to be valid.

The Guardian picks up the messages emitted from the integrity-checker on wormchain and only if they correspond to an actual observation the Guardian has made itself, it will sign it and broadcast the signature to the Guardian p2p network. Thereby, the integrity-checker is not able to modify or inject messages, it can only block messages.

## Detailed Design

### Pre-Observations

Pre-observations are essentially like observations, i.e. a Wormhole message signed by a single guardian, but they can be distinguished by their signature format and prefix. They follow the same signature format as signed gossip messages described in 0009_guardian_key.md with a unique signature prefix.

### Batching

Pre-observations are accumulated over a configurable period of time and submitted in a batch to Wormchain to save gas cost and computational overhead.

### Persistance

The Guardian persists the status of pre-observations to the local database.

### Retry

The Guardian periodically watches the Wormchain state and ensures that all local pre-observations have been submitted correctly and retries otherwise.

## APIs

Todo

## Caveats

This design is incomplete. See goals that are out of scope for now.