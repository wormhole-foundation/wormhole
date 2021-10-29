## Wormhole event BigTable schema

### Row Keys

Row keys contain the MessageID, delimited by colons, like so: `EmitterChain:EmitterAddress:Sequence`.

BigTable can only be queried for data in the row key. Only row key data is indexed. You cannot query based on the value of a column; however you may filter based on column value.

### Column Families

BigTable requires that columns are within a "Column family". Families group columns that store related data. Grouping columns is useful for efficient reads, as you may specify which families you want returned.

The column families listed below represent data unique to a phase of the attestation lifecycle.

- `MessagePublication` holds data about a user's interaction with a Wormhole contract. Contains data from the Guardian's VAA struct.

- `QuorumState` stores the signed VAA once quorum is reached.

### Column Qualifiers

Each column qualifier below is prefixed with its column family.

- `MessagePublication:Version` Version of the VAA schema.
- `MessagePublication:GuardianSetIndex` The index of the active Guardian set.
- `MessagePublication:Timestamp` Timestamp when the VAA was created by the Guardian.
- `MessagePublication:Nonce` Nonce of the user's transaction.
- `MessagePublication:Sequence` Sequence from the interaction with the Wormhole contract.
- `MessagePublication:EmitterChain` The chain the message was emitted on.
- `MessagePublication:EmitterAddress` The address of the contract that emitted the message.
- `MessagePublication:InitiatingTxID` The transaction identifier of the user's interaction with the contract.
- `MessagePublication:Payload` The payload of the user's message.

- `QuorumState:SignedVAA` the VAA with the signatures that contributed to quorum.
