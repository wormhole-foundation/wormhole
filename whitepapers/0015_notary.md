# Notary

## Objective

Provide a generalized message evaluation system that can assess the validity of Wormhole messages and make decisions about their processing based on verification state, with the ability to delay suspicious messages for manual review or permanently block malicious ones.

## Background

The core Wormhole protocol ensures message authenticity through guardian signatures but it cannot inherently determine whether a message represents legitimate activity or is the result of an exploit, bug, or malicious behavior.

The Transfer Verifier system was developed to validate the legitimacy of token transfer transactions by analyzing on-chain data and transaction receipts. However, this validation occurs at the chain watcher level and produces verification states that need to be acted upon elsewhere in the codebase. Transfer Verifier only adds a status to a message, but does not alter how that message is processed on the way to becoming a VAA.

## Goals

The Notary system aims to:

* Provide a standardized, coordinated response to Transfer Verifier results across all guardians
* Enable temporary delays for suspicious messages to allow for manual review and investigation
* Permanently block messages that have been definitively identified as malicious or not well-formed
* Maintain a persistent record of delayed and blocked messages across guardian restarts
* Support the broader Wormhole security model by providing an additional layer of protection
* Operate as a general-purpose message evaluation framework that could be extended beyond token transfers

## Non-Goals

* Replace or duplicate the Transfer Verifier's analysis capabilities
* Modify or alter message content - the Notary only suggests processing decisions
* Provide automatic resolution of suspicious messages - manual intervention is required
* Support message types other than token transfers in the initial implementation
* Synchronize state between guardians - each guardian maintains its own Notary state
* Take action on hacks or illegitimate messages that originate from outside of the Wormhole network

## Overview

The Notary operates as a message evaluation layer within each guardian node, positioned between the message observation phase and the VAA signing phase. It receives messages from the watchers that have already been processed by the Transfer Verifier (if enabled) and makes decisions based on their verification state.

The system implements three possible verdicts:

1. **Approve** - Messages should proceed normally through the signing process
2. **Delay** - Messages should be temporarily held for manual review (default: 4 days)
3. **Blackhole** - Messages should be permanently blocked from processing

The Notary maintains persistent storage of delayed and blackholed messages, ensuring consistent behavior across guardian restarts and providing operators with the ability to manage problematic messages.


## Detailed Design

### Architecture

The Notary is implemented as a Go package (`node/pkg/notary`) that integrates with the guardian's message processing pipeline. It consists of:

* **Core Notary struct** - Main processing logic and state management
* **Database interface** - Persistent storage for delayed and blackholed messages  
* **Message queues** - In-memory management of pending messages

### Message Processing Flow

When a message reaches the Notary:

1. **Type Check** - Only token transfer messages are evaluated; all others are automatically approved
2. **Blackhole Check** - If the message is already blackholed, return Blackhole verdict immediately
3. **Verification State Evaluation** - Based on the Transfer Verifier's assessment:
   - `Valid`, `NotVerified`, `NotApplicable`, `CouldNotVerify` → **Approve**
   - `Anomalous` → **Delay** (4 days default)
   - `Rejected` → **Blackhole** (permanent)

Similar to the Governor and Accountant systems, the Notary maintains a list of messages which the Processor can consult to determine if a message should be processed into a VAA or not. It cannot directly block messages independently of the Processor.

### Verdict Types and Behavior

#### Approve Verdict
- Message proceeds immediately to VAA signing
- No database storage required
- Used for all non-token-transfer messages and verified transfers

#### Delay Verdict  
- Message is stored in both database and in-memory queue with release timestamp
- Default delay period: 4 days (configurable via `DelayFor` constant)
- Messages are automatically released after the delay period expires
- Delayed messages can be manually promoted to blackholed status if determined to be malicious

#### Blackhole Verdict
- Message is permanently blocked from processing
- Stored in database for persistence across restarts
- Cannot be automatically released - requires manual intervention
- Used for messages definitively identified as malicious by Transfer Verifier

### Database Schema

The Notary uses a BadgerDB-based storage system with two primary data types:

```go
// Delayed messages with release timestamps
type PendingMessage struct {
    Msg         MessagePublication
    ReleaseTime time.Time
}

// Blackholed messages (permanent storage)
type MessagePublication struct {
    // Standard Wormhole message fields
    // Stored by VAAHash for efficient lookup
}
```

Storage keys are prefixed to distinguish between delayed and blackholed messages:
- Delayed: `notary:delayed:<hash>`
- Blackholed: `notary:blackholed:<hash>`

### Integration with Transfer Verifier

The Notary builds upon the Transfer Verifier system but serves a different purpose:

**Transfer Verifier**:
- Analyzes transaction receipts and on-chain data
- Determines if a transfer represents legitimate activity
- Sets verification state on messages (`Valid`, `Anomalous`, `Rejected`, etc.)
- Operates at the chain watcher level

**Notary**:
- Consumes Transfer Verifier results
- Makes coordinated processing decisions based on verification state
- Manages delayed and blackholed messages
- Operates at the guardian processor level

This separation allows the Transfer Verifier to focus on analysis while the Notary handles the operational response to that analysis.

### Relationship to Governor and Accountant

The Notary shares architectural similarities with the Governor and Accountant systems but serves a distinct purpose:

#### Common Patterns
All three systems:
- Operate as message filters in the guardian processing pipeline
- Can delay or block message processing based on their evaluation criteria
- Maintain persistent state across guardian restarts
- Provide manual override capabilities for operators
- Process messages serially in the order: Notary → Governor → Accountant

#### Key Differences

**Notary**:
- **Scope**: General message validation based on verification state
- **Criteria**: Transfer Verifier results (validity/legitimacy)
- **Action**: Approve, delay (4 days), or permanently blackhole
- **Focus**: Security and fraud prevention

**Governor**:
- **Scope**: Token transfer rate limiting
- **Criteria**: Transfer value and volume thresholds
- **Action**: Approve immediately or delay (24 hours) based on economic limits
- **Focus**: Economic protection and exploit impact limitation

**Accountant**:
- **Scope**: Cross-chain token accounting
- **Criteria**: Token balance consistency across chains
- **Action**: Approve or reject based on accounting rules
- **Focus**: Preventing unbacked token creation

### Message Release Mechanisms

The Notary provides multiple pathways for releasing delayed messages:

1. **Automatic Release** - Messages are automatically released after the delay period expires
2. **Manual Promotion** - Operators can promote delayed messages to blackholed status
3. **Manual Removal** - Operators can remove messages from delay or blackhole lists (via `forget` method)

### Operational Considerations

#### Modularity

- The Notary is designed to be modular with respect to the node. It can be disabled via a configuration flag.
- The Notary can be enabled or disabled independently of the Transfer Verifier and other security mechanisms.

#### Monitoring and Alerting
- Guardians should monitor Notary verdict distributions
- Unusual patterns in delayed or blackholed messages may indicate attacks
- Database growth should be limited, as delayed messages are automatically released and delete, and blackholed messages should be extremely rare.

#### Manual Intervention
- Delayed messages provide a window for manual review and investigation
- Operators can analyze suspicious transfers and make informed decisions
- Coordination between guardians may be necessary for consistent responses

## Security Considerations

### Trust Model
The Notary inherits the security properties of the Transfer Verifier system it depends on. If the Transfer Verifier is compromised or produces incorrect results, the Notary will make decisions based on that flawed information.

### Denial of Service
An attacker who can cause the Transfer Verifier to mark legitimate messages as "Anomalous" could cause widespread delays in message processing. However, messages are automatically released after the delay period, limiting the impact.
Guardians can also intervene manually to release delayed messages, or disable the Notary entirely which would prevent the delay from occurring.

### State Consistency
Each guardian maintains its own Notary state independently. While this provides resilience against single points of failure, it means guardians might have slightly different views of delayed/blackholed messages.

### Manual Override Risks
The ability to manually manage delayed and blackholed messages provides operational flexibility but also introduces the risk of human error or malicious operator behavior.

## Future Enhancements

### Extended Message Type Support
While currently limited to token transfers, the Notary architecture could be extended to support other message types with appropriate verification mechanisms.

### Configurable Delay Periods
The current 4-day delay period could be made configurable per message type or based on other criteria.
