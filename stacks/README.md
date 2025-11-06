# Stacks

## High-level architecture

The Stacks watcher is a component that monitors the Stacks blockchain and processes transactions to find Wormhole message publication events.

The Stacks watcher uses a polling-based approach to monitor the Stacks blockchain:

- **Bitcoin Block Anchoring**: Uses Bitcoin blocks (burn blocks) as the anchor point for confirmation.
- **BlockPoller**: Polls the Stacks RPC API for new Bitcoin blocks every 2 seconds.
- **Confirmation**: Considers a block final after 6 Bitcoin block confirmations.
- **ObsvReqProcessor**: Processes observation requests to re-observe specific transactions.

## Processing Flow

The watcher maintains two key tracking points:

- Latest Bitcoin block height seen
- Last processed Bitcoin block height

When new blocks are found, it:

1. Polls for new Bitcoin (burn) blocks
2. Processes Bitcoin blocks that have reached sufficient confirmation (6 blocks)
3. Fetches all Stacks blocks anchored to those Bitcoin blocks
4. Processes transactions in those Stacks blocks
5. Examines events in those transactions to find Wormhole message publication events
6. Extracts message data and creates MessagePublication objects
