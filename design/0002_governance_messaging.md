# Cross-Chain Governance Decision Messaging

[TOC]

## Objective

Establish a protocol for wormhole core implementations and modules on different chains to communicate governance
decisions/instructions with each other.

## Goals

- Define a messaging protocol for global and chain-specific governance actions.
- A message should carry all required information required for an implementation to implement it.

## Non-Goals

- Define the governance processes itself (staking, voting etc.)

## Overview

Governance happens in a smart contract on Solana (to be specified). This contract passes VAAs with finalized decisions to the Wormhole.

Implementations on other chains have the address of that contract hardcoded and accept a set of VAAs for governance actions from that contract.
All governance VAAs follow the `GovernancePacket` structure.

### General Packet Structure

`Module` is the component this governance VAA is targeting. This could be the core bridge contract but any
program (e.g. a Wormhole extension module) can use the governance contract and governance messaging by picking a unique
identifier.

`Action` is a unique action ID that identify different governance message payloads of a module.

```go
GovernancePacket struct {
    // Module identifier (left-padded)
    Module [32]byte
    // Action index
    Action uint8
    // Chain index (0 for non-specific actions like guardian set changes)
    Chain uint16
    // Action-specific payload fields
    [...]
}
```

### Specified Governance VAAs

The following VAAs are example governance VAAs of the core Wormhole contract.

```go
// ContractUpgrade is a VAA that instructs an implementation on a specific chain to upgrade itself
ContractUpgrade struct {
    // Core Wormhole Module
    Module [32]byte = "Core"
    // Action index (1 for Contract Upgrade)
    Action uint8 = 1
    // Target chain ID
    Chain uint16
    // Address of the new Implementation
    NewContract [32]byte
}

// GuardianSetUpgrade is a VAA that instructs an implementation to upgrade the current guardian set
GuardianSetUpgrade struct {
    // Core Wormhole Module
    Module [32]byte = "Core"
    // Action index (2 for GuardianSet Upgrade)
    Action uint8 = 2
    // This update is chain independent
    Chain uint16 = 0

    // New GuardianSet
    NewGuardianSetIndex uint32
    // New GuardianSet
    NewGuardianSetLen u8
    NewGuardianSet []Guardian
}
```
