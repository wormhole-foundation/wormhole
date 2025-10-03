# Notary

## Objective

Provide a generalized message evaluation system that can assess the validity of Wormhole messages and make decisions about their processing based on verification state, with the ability to delay suspicious messages for manual review or permanently block malicious ones.

## Background

The core Wormhole protocol ensures message authenticity through Guardian signatures but it cannot inherently determine whether a message represents legitimate activity or is the result of an exploit, bug, or malicious behavior.

The Transfer Verifier system was developed to validate the invariants of wrapped token transfers by analyzing on-chain data and transaction receipts (or other on-chain artifacts). However, this validation occurs at the chain watcher level and produces verification states that need to be acted upon elsewhere in the codebase. Transfer Verifier only returns an accept/reject result when provided with a wrapped token transfer messages, but does not alter how that message is processed on the way to becoming a VAA.

## Goals

The Notary system aims to:

* Provide a standardized, coordinated response to Transfer Verifier results across all Guardians
* Enable temporary delays for suspicious messages to allow for manual review and investigation
* Maintain a list of messages that are confirmed to be malicious or malformed
* Maintain a persistent record of delayed and blocked messages across Guardian restarts
* Support the broader Wormhole security model by providing an additional layer of protection
* Operate as a general-purpose message evaluation framework that could be extended beyond Wrapped Token Transfers

## Non-Goals

* Replace or duplicate the Transfer Verifier's analysis capabilities
* Modify or alter message content - the Notary only suggests processing decisions
* Provide automatic resolution of suspicious messages - manual intervention is required
* Support message types other than Wrapped Token Transfers (in the initial implementation)
* Synchronize state between Guardians - each Guardian maintains its own Notary state
* Take action on hacks or illegitimate messages that originate from outside of the Wormhole network

## Overview

The Notary operates as a message evaluation layer within each Guardian node, positioned between the message observation phase and the VAA signing phase. It receives messages from the watchers that have already been processed by the Transfer Verifier (if enabled) and makes decisions based on their verification state.

The system implements three possible verdicts:

1. **Approve** - Messages should proceed normally through the signing process
2. **Delay** - Messages should be temporarily held for manual review (default: 4 days)
3. **Blackhole** - Messages should be permanently blocked from processing

The Notary maintains persistent storage of delayed and blackholed messages, ensuring consistent behavior across Guardian restarts and providing operators with the ability to manage problematic messages.


## Detailed Design

### Architecture

The Notary is implemented as a Go package (`node/pkg/notary`) that integrates with the Guardian's message processing pipeline. It consists of:

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

Similar to the Governor and Accountant systems, the Notary maintains a list of messages which the Processor can consult to determine if a message should be processed into a VAA or not. It cannot directly block or delay messages.

### Verdict Types and Behavior

_All three statuses are mutually-exclusive, and no message should be duplicated within or across the containing data structures._

#### Approve Verdict
- Message proceeds immediately to VAA signing
- No database storage required
- Used for all non-token-transfer messages and verified transfers

#### Delay Verdict  
- Message is stored in both database and in-memory queue with release timestamp
- Default delay period: 4 days
- Messages are automatically released after the delay period expires
- Delayed messages can be manually promoted to "blackholed" status if determined to be malicious
- Delayed messages can be manually removed from the delay queue if determined to be safe, and then processed immediately

#### Blackhole Verdict
- Message is permanently blocked from processing (though ultimately it is the Processor that actually takes this action)
- Stored in database for persistence across restarts
- Cannot be automatically released - requires manual intervention
- Currently only used for messages definitively identified as malicious by Transfer Verifier

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
- Analyzes transaction receipts and on-chain data for Wrapped Token Transfers
- Determines if a transfer is well-formed (i.e. has matching events on the Token and Core Bridges)
- Returns a corresponding verification state on messages (`Valid`, `Anomalous`, `Rejected`, etc.)
- Operates at the chain watcher level

**Notary**:
- Consumes Transfer Verifier results
- Makes recommendations based on the verification state of the messages
- Manages its own list of delayed and blackholed messages
- Operates at the Guardian processor level

This separation allows a separation of concerns:
- the Transfer Verifier focuses on analysis
- the Notary handles the interpretation of the results
- the Processor takes action based on the Notary's verdict (as well as the Governor's and Accountant's)

### Relationship to Governor and Accountant

The Notary shares architectural similarities with the Governor and Accountant systems but serves a distinct purpose:

#### Common Patterns
All three systems:
- Operate as message filters in the Guardian processing pipeline
- Can instruct the processor to delay or block message processing based on their evaluation criteria
- Maintain persistent state across Guardian restarts
- Provide manual override capabilities for operators

#### Key Differences

**Notary**:
- **Scope**: General message validation based on verification state
- **Criteria**: Transfer Verifier results (validity/legitimacy)
- **Action**: Approve, delay (4 days), or permanently reject ("blackhole")
- **Focus**: Security and fraud prevention

**Governor**:
- **Scope**: Wrapped Token Transfer rate limiting
- **Criteria**: Transfer value and volume thresholds
- **Action**: Approve immediately or delay (24 hours) based on notional value of transfers
- **Focus**: Limitation of impact for software errors in the Guardian watchers

**Accountant**:
- **Scope**: Cross-chain token accounting
- **Criteria**: Token balance consistency across chains
- **Action**: Approve or reject based on accounting rules
- **Focus**: Prevention of erroneous token unlocks, reducing the effectiveness of cross-chain exploits

### Operational Considerations

#### Modularity

- The Notary is designed to be modular with respect to the node. It can be disabled via a configuration flag.
- The Notary can be enabled or disabled independently of the Transfer Verifier and other security mechanisms.

_The Notary's initial implementation only acts on results from the Transfer Verifier; if Transfer Verifier is not enabled
the Notary will approve all messages. (This is because the Notary works based on the Verification State of a Message
Publication, and the only place this field is used currently is within watchers with Transfer Verification implementations)._

#### Monitoring and Alerting

- Guardians should monitor Notary verdict distributions
- Unusual patterns in delayed or blackholed messages may indicate attacks
- Database growth should be limited, as delayed messages are automatically released and deleted, and blackholed messages should be extremely rare.

#### Manual Intervention

Guardians are able to use administrator commands to change the status of delayed or rejected messages.
This provides flexibility in case the system has false positives or other software bugs.

Possible actions include:
- Releasing a delayed message immediately
- Extending the release time of a delayed message
- Marking a delayed message as blackholed
- Marking a blackholed message as delayed

## Security Considerations

### Trust Model
The Notary inherits the security properties of the Transfer Verifier system it depends on. If the Transfer Verifier is compromised or produces incorrect results, the Notary will make decisions based on that flawed information.

### Denial of Service
An attacker who can cause the Transfer Verifier to mark legitimate messages as "Anomalous" could cause widespread delays in message processing for Wrapped Token Transfers. However, messages are automatically released after the delay period, limiting the impact.
Guardians can also intervene manually to release delayed messages, or disable the Notary entirely which would prevent the delay from occurring even if the Transfer Verifier continues to yield incorrect verification states.

### State Consistency
Each Guardian maintains its own Notary state independently. While this provides resilience against single points of failure, it means Guardians might have slightly different views of delayed/blackholed messages.

### Manual Override Risks
The ability to manually manage delayed and blackholed messages provides operational flexibility but also introduces the risk of human error or malicious operator behavior.
However, a VAA will only be crated for a message if supermajority of Guardians decide to process the message, as usual.

## Future Enhancements

### Extended Message Type Support
The operation of the Notary is abstract and can be generalized to any Message Publication and verification state that the Guardian software uses.
While currently the Notary only operations in conjunction with Wrapped Token Transfers and the Transfer Verifier, the Notary architecture could be extended to support other message types with appropriate verification mechanisms.

### Configurable Delay Periods
The current 4-day delay period could be made configurable per message type or based on other criteria.
