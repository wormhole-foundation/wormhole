# Stacks Watcher

This package implements the watcher for the Stacks blockchain.

Responsibility: Observe message publication events from the Wormhole Core contract on the Stacks blockchain.

## High-level architecture

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

## Re-observation Support

The watcher can reprocess specific transactions when requested. It:

1. Fetches the transaction details using the transaction ID
2. Processes the transaction to look for Wormhole events
3. Publishes any valid messages found

## Configuration

The watcher requires the following configuration:

- Stacks RPC URL: The URL of the Stacks blockchain API
- Contract Address: The address of the Wormhole Core contract on Stacks

## API Endpoints

The watcher interacts with the Stacks Node RPC API endpoints:

- `/v3/tenures/info` - Get current tenure information
- `/v3/tenures/blocks/{consensusHash}` - Get blocks for a tenure by consensus hash
- `/v3/tenures/blocks/height/{height}` - Get tenure blocks by Bitcoin block height
- `/v3/blocks/replay/{blockHash}` - Get block transactions with simulation results (includes `vm_error` field for failed transactions)
- `/v3/transaction/{txID}` - Get transaction details by ID

**Note:** The v3 block replay endpoint includes a `vm_error` field (added in stacks-core PR #6575) which contains the runtime error message for failed transactions. This field is null for successful transactions.

## Error Handling

The watcher handles the following error conditions, however generally errors are ignored and subsequent blocks/transactions are still processed (since there is always the option to reobserve missed transactions).

- RPC connection failures - Will retry on the next polling interval
- Block processing failures - Logs errors and continues with the next block
- Transaction processing failures - Logs errors and continues with the next transaction
- Event processing failures - Logs errors and continues with the next event
