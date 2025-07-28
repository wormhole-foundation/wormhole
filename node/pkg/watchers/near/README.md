# NEAR Watcher
This package implements the watcher for the NEAR blockchain.

Responsibility: Observe finalized `publishMessage` event emissions from the Wormhole Core account on the NEAR blockchain.

## High-level architecture
There are multiple supervised runners:
* *BlockPoll*: Polls the NEAR RPC API for finalized blocks and generates transactionProcessingJobs for every single transaction
* *ChunkFetcher*: Fetches chunks for each block in parallel
* *TxProcessor*: Processes `transactionProcessingJob`, going through all receipt outcomes, searching for Wormhole messages, and checking that they have been finalized. If there are Wormhole messages in any receipts from this transaction but those receipts are not in finalized blocks, the `transactionProcessingJob` will be put in the back of the queue.
* *ObsvReqProcessor*: Process observation requests. An observation request is a way of kindly asking the Guardian to go back in time and look at a particular transaction and try to identify wormhole events in it. Observation requests are received from other Guardians or injected through the admin API. Even though they are signed, they should not be trusted.


* chunkProcessingQueue gets new chunk hashes.
	These are processed once we think that the transactions in it are likely to be completed.
* transactionProcessingQueue is a priority queue and gets all the transactions from the chunks.
	These are constantly processed and put back to the end of the queue if processing fails.
	If processing a transaction fails, most likely because not all receipts have been finalized, it is retried with an exponential backoff and eventually it is dropped.
* multiple workers process both types of jobs from these two queues

Determining finality of blocks:
* There is a lru cache, finalizedBlocksCache, that keeps track of blocks hashes of blocks that are known to be finalized
* Blocks that are returned from NEAR RPC API as the last finalized block are added to the cache
* As we walk back the prev_block hashes of final blocks, their block hashes are added to the cache as well
* If we are wondering if a block is finalized but it's not in the cache (this situation is most likely to occur
	during the processing of reobservation requests), then we query a couple of blocks ahead and check if the block in
	question shows up as "last_final_block" in any of the blocks ahead.

## FAQ
*Do we really have to parse every single NEAR transaction?*

Unfortunately NEAR does not (yet?) provide a mechanism to subscribe to a particular contract.
There is a RPC API EXPERIMENTAL_changes_in_block which would tell you if the block contained any receipts that touch your account, but there is no way of knowing in which block the transaction started. Even if there was, you'd still need to process all transactions in the block and we are expecting there to be a Wormhole transaction in most blocks.

## Logging
* log_msg_type (enum)
	* tx_processing_retry: A transaction was not successfully processed and will be retried.
	* tx_processing_retries_exceeded: A transaction was not successfully processed even after `txProcRetry` retries and was dropped
	* tx_processing_error: Transaction processing failed
	* startup_error: Watcher was unable to start up.
	* block_poll_error: Generic error when polling for new blocks
	* chunk_processing_failed
	* tx_proc_queue_full: The transaction processing queue is full but there are new transaction. This means that the Guardian is not able to catch up with block production on NEAR. This is a critical error that needs to be investigated. `chunk_id` is the ID of the chunk from which all or some transactions have been dropped.
	* obsv_req_received: Observation request received
	* info_process_tx: Transaction processing is being attempted. This is done for all transactions on NEAR, so we only log this with debugging level. Log fields: `tx_hash`.
	* wormhole_event: A Wormhole event is being processed
	* wormhole_event_success: A Wormhole event has been successfully processed
	* watcher_behind: The NEAR watcher fell behind too much and skipped blocks.
	* height: The highest height of a block from which at least one transaction has been processed. Includes `height`.
	* block_poll: A block has been successfully polled. Includes `height`.
	* polling_attempt: There are new final blocks available and the watcher is starting to poll them. Includes `previous_height` (the height of the previously highest block that we polled) and `newest_final_height` (the height of the latest block that is final).
* tx_hash: Transaction hash
* error_type (enum)
	* invalid_hash: Program encountered a hash that is not well-formed, i.e. not 32 bytes long.
	* nearapi_inconsistent: NEAR RPC returned data that doesn't make sense.<!-- cspell:disable-line -->
	* malformed_wormhole_event: The wormhole log emission is malformed. This should never happen. If it does, that'd be indicative of a big problem.
	* startup_fail: Something went wrong during watcher startup.


## Assumptions
* We assume that transactions containing Wormhole messages are finalized on the NEAR blockchain in `initialTxProcDelay ^ (txProcRetry+2)` time after the block containing the start of the transaction has been observed. Otherwise they will be missed. Strong network congestion or gaps/delays in block production could violate this.

## Testing and Debugging

### Unit tests
The testing strategy is to run the watcher with a mock RPC server. The mock RPC server mostly forwards requests to the mainnet RPC and caches them. Cached response are committed to the repository such that the tests don't actually depend on the mainnet RPC server.
For negative tests, there are folders with synthetic RPC responses. The synthetic data is generated with a bash script: [createDerivatives.sh](nearapi/mock/createDeriviates.sh).

### Integration tests
Run tilt without optional networks:
```sh
tilt up -- --evm2=false --solana=false --terra_classic=false --terra2=false
```

If you have everything built and setup:
```sh
cd wormhole/near/
npm ci
ts-node test/sdk.ts
```

If it doesn't work, this dockerfile may be a good starting point:
```docker
RUN dnf update && dnf install -y python3 npm curl
RUN dnf install -y gcc gcc-c++ make git
RUN npm install -g typescript ts-node n
RUN n stable

RUN git clone https://github.com/wormhole-foundation/wormhole.git

WORKDIR /wormhole/ethereum
RUN npm ci
RUN npm run build

WORKDIR /wormhole/sdk/js
RUN npm ci
RUN npm run build-all

WORKDIR /wormhole/near
RUN npm ci
RUN ts-node test/sdk.ts
```

