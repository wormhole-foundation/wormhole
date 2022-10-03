# NEAR Watcher
This package implements the watcher for the NEAR blockchain.

Responsibility: Observe finalized `publishMessage` event emissions from the Wormhole Core account on the NEAR blockchain.

## High-level architecture
There are multiple supervised runners:
* *BlockPoll*: Polls the NEAR RPC API for finalized blocks and generates transactionProcessingJobs for every single transaction
* *ChunkFetcher*: Fetches chunks for each block in parallel
* *TxProcessor*: Processes `transactionProcessingJob`, going through all receipt outcomes, searching for Wormhole messages, and checking that they have been finalized. If there are Wormhole messages in any receipts from this transaction but those receipts are not in finalized blocks, the `transactionProcessingJob` will be put in the back of the queque.
* *ObsvReqProcessor*: Process observation requests. An observation request is a way of kindly asking the Guardian to go back in time and look at a particular transaction and try to identify wormhole events in it. Observation requests are received from other Guardians or injected through the admin API. Eventhough they are signed, they should not be trusted.


* chunkProcessingQueue gets new chunk hashes.
	These are processed once we think that the transactions in it are likely to be completed.
* transactionProcessingQueue is a priority queue and gets all the transactions from the chunks.
	These are constantly processed and put back to the end of the queque if processing fails.
	If processing a transaction fails, most likely because not all receipts have been finalized, it is retried with an exponential backoff and eventually it is dropped.
* multiple workers process both types of jobs from these two queques

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