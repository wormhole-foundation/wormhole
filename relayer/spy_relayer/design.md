## Docker Images

VAA_Listener
Redis
Relayer

## Dependencies

- Guardian Spy
- Blockchain Nodes for all supported target chains

## High Level Workflow:

The VAA_Listener listens for Token Bridge SignedVAAs coming from both the guardian network (via a guardian spy), and end users (via a REST interface).

The VAA_Listener then passes this SignedVAA into a validate function, which determines if this VAA should be processed. If so, it enqueues the VAA in redis to be processed by the relayer.

Validation criteria:

- VAA must be token bridge signedVAA of type payload 1.
- VAA be for a supported target chain & origin asset.
- VAA must have a sufficiently high 'fee' field on it.
- VAA must not already be in the 'incoming', 'in-work', or 'pending confirmation' redis tables. (Optionally, also a max-retries exceeded table?)
- VAA must not be already redeemed.

# Redis

Four tables:

- Incoming: These are requests which have been queued by the listener, but have not yet been attempted by the relayer.
- In-Work: These are requests which have been popped off the 'incoming' stack, but have not yet been successfuly submitted on chain.
- Pending Confirmation: These are requests which have been successfully submitted on chain, and are waiting for a finality check to ensure they were not rolled back.
- Failed: These are requests which were removed from the In-Work table due to having exceeded their max number of retries.

All requests enter via the 'Incoming' table, and should eventually either be 'purged' once they successfully exit the Pending Confirmation table, or end in the "Failed" table. For data retention purposes, it may be worthwhile to have a "Completed" table, however, logging should be sufficient for this.

# Relayer

The relayer is responsible for monitoring redis and submitting transaction on chain.

The relayer spawns a worker for each combination of {targetChain + privateKey}, such that no two schedulers should collide on-chain.

Each worker perpetually attempts to submit items in the 'In-Work' table which are assigned to them. When they successfully process an In-Work item, they move it to the Pending Confirmation table. If they are not successful, they increment the failure-count on the In-Work item. If the failure-count exceeds MAX_RETRIES, the In-Work item is moved to the 'Failed' table.

If there are no eligible items in the In-Work table, the worker will scan the Incoming table, and move the Incoming item into the 'In-Work' table under their name. Workers are identified by a string which is their target chain + the public key of their wallet.

Prior to submitting a signedVAA, relayers should check that the VAA has not been redeemed, as other processes may 'scoop' a VAA.
