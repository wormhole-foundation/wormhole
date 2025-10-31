# Guardian Notary Admin Commands

Below are admin controls surfaced to the Guardians for the Notary system. The Notary is responsible for evaluating message publications and determining how they should be processed, providing three possible verdicts: Approve, Delay, or Blackhole (Reject).

## Default Behavior / Enabling the Notary

The Notary feature is disabled by default. Guardians can enable it by passing the following flag to the `guardiand` command when starting it up:

```bash
--notaryEnabled=true
```

When enabled, the Notary will evaluate all incoming message publications and apply the appropriate verdict based on the message status and content.

_NOTE: Currently, the Notary has no effect on message publication processing unless Transfer Verifier is also enabled._

## Notary Overview

The Notary evaluates incoming message publications and categorizes them:

1. **Approve** - Messages pass through normally (non-error status)
2. **Delay** - Messages are temporarily held for manual inspection (anomalous messages)  
3. **Blackhole** - Messages are permanently blocked from publication (rejected status)

Delayed messages are stored with timestamps indicating when they should be released. Blackholed messages are stored permanently in the database to prevent future processing.

## Admin Commands

### Query Commands

#### Get Delayed Message Details

To retrieve detailed information about a specific delayed message, Guardians can run the `notary-get-delayed-message` admin command:

```bash
guardiand admin notary-get-delayed-message "chain_id/emitter_address/sequence_number" --socket /path/to/admin.sock
```

This command displays the message ID, release time, and other details for the specified delayed message.

#### Get Blackholed Message Details

To retrieve detailed information about a specific blackholed message, Guardians can run the `notary-get-blackholed-message` admin command:

```bash
guardiand admin notary-get-blackholed-message "chain_id/emitter_address/sequence_number" --socket /path/to/admin.sock
```

This command displays the message ID and details for the specified blackholed message.

#### List All Delayed Messages

To list all currently delayed messages, Guardians can run the `notary-list-delayed-messages` admin command:

```bash
guardiand admin notary-list-delayed-messages --socket /path/to/admin.sock
```

This command displays a list of all message IDs currently in the delayed queue.

#### List All Blackholed Messages

To list all currently blackholed messages, Guardians can run the `notary-list-blackholed-messages` admin command:

```bash
guardiand admin notary-list-blackholed-messages --socket /path/to/admin.sock
```

This command displays a list of all message IDs currently in the blackholed list.

### Management Commands

#### Blackholing Delayed Messages

To move a delayed VAA to the blackholed list (permanently blocking it), Guardians can run the `notary-blackhole-delayed-message` admin command:

```bash
guardiand admin notary-blackhole-delayed-message "chain_id/emitter_address/sequence_number" --socket /path/to/admin.sock
```

This command moves the specified VAA from the Notary's delayed list to the blackholed list, preventing it from ever being published.

**Warning:** *Blackholing a VAA should only be used when there is a confirmed hack that directly affects the security of the Wormhole network. This action permanently blocks the message from being processed.*

#### Releasing Delayed Messages

To immediately release a delayed VAA and publish it, Guardians can run the `notary-release-delayed-message` admin command:

```bash
guardiand admin notary-release-delayed-message "chain_id/emitter_address/sequence_number" --socket /path/to/admin.sock
```

This command releases the specified VAA from the Notary's delayed list and publishes it immediately, bypassing the remaining delay period.

**Warning:** *Releasing a VAA manually should rarely occur. Guardians should only release VAAs early if they are confident the message is legitimate and does not result from an exploit.*

#### Removing Blackholed Messages

To remove a VAA from the blackholed list and restore it to the delayed list, Guardians can run the `notary-remove-blackholed-message` admin command:

```bash
guardiand admin notary-remove-blackholed-message "chain_id/emitter_address/sequence_number" --socket /path/to/admin.sock
```

This command removes the specified VAA from the blackholed list and adds it back to the delayed list with zero delay, so it will be published on the next processing cycle.

**Warning:** *Removing a blackholed message should only be done if the previous blackholing was determined to be in error. This action should be used with extreme caution.*

#### Resetting Release Timer

To reset the release timer for a delayed message to a specified number of days, Guardians can run the `notary-reset-release-timer` admin command:

```bash
guardiand admin notary-reset-release-timer "chain_id/emitter_address/sequence_number" "number_of_days" --socket /path/to/admin.sock
```

The number of days must be between 0 and 30. This command resets the release timer for the specified VAA to the given number of days from the current time.

**Warning:** *Resetting the release timer should only be used when more time is needed to investigate potential fraud or security issues. Use this command judiciously to avoid unnecessary delays in legitimate message processing.*



## Message ID Format

All Notary admin commands use a consistent Message ID format:

```
chain_id/emitter_address/sequence_number
```

For example:
```
1/0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585/12345
```

Where:
- `1` is the chain ID (Ethereum mainnet)
- `0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585` is the emitter address  
- `12345` is the sequence number

## Default Delay Timing

The Notary system uses a default delay of 4 days (96 hours) for anomalous messages that require manual review. The maximum delay that can be set is 30 days.

Messages are automatically released after their delay period expires, unless they have been manually blackholed by the Guardians.
