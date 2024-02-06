# Guardian Key Usage

## Objective

- Describe how guardian keys are used and how message confusion is avoided.

## Background

Message confusion could occur when a Guardian signs a message and an attacker replays that message elsewhere where it is interpreted as a different message type, which could lead to unintended behavior.

## Overview

The Guardian Key is used to:

1. Sign gossip messages
   1. heartbeat
   2. governor config and governor status
   3. observation request
2. Sign Observations
   1. Version 1 VAAs

## Detailed Design

Signing of gossip messages:

1. Prepend the message type prefix to the payload
2. Compute Keccak256Hash of the payload.
3. Compute ethcrypto.Sign()

Signing of Observations:

- v1 VAA: `double-Keccak256(observation)`.

Rationale

- Gossip messages cannot be confused with other gossip messages because the message type prefix is prepended.
- Gossip messages cannot be confused with observations because observations utilize a double-Keccak256 and the payload is enforced to be `>=34` bytes.
