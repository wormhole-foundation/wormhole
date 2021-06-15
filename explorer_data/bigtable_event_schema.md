## Wormhole event BigTable schema

### Row Keys

Row keys contain the MessageID, delimited by colons, like so: `EmitterChain:EmitterAddress:Sequence`.

BigTable can only be queried for data in the row key. Only row key data is indexed. You cannot query based on the value of a column; however you may filter based on column value.

### Column Families

BigTable requires that columns are within a "Column family". Families group columns that store related data. Grouping columns is useful for efficient reads, as you may specify which families you want returned.

The column families listed below represent data unique to a phase of the attestation lifecycle.

- `MessagePublication` holds data about a user's interaction with a Wormhole contract. Contains data from the Guardian's VAA struct.

- `Signatures` holds observed signatures from Guardians within a GuardianSet. Holds signatures independent of index within GuardianSet. This column family will provide an account of "which Guardians observed X transaction, and when?".

- `VAAState` records incremental updates toward Guardian consensus. The VAAState column family holds the progression of signatures of a GuardianSet. Each update to the Signatures list of the VAA struct is recorded. This column family will provide an account of "Which Guardians contributed to reaching quorum".

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

- `Signatures:{GuardianAddress}` This column qualifier will be the address of the Guardian, and the data stored here will be the signature broadcast by the Guardian. There will be a column in this family for each Guardian address that appears in a Guardian set. The column qualifier is a part of the data that is recorded here. See the [BigTable design docs](https://cloud.google.com/bigtable/docs/schema-design#columns) for the thought process behind this approach.

- `VAAState:Signatures:{GuardianSetIndex}` a list of objects containing Guardian signatures and the index of the Guardian within the GuardianSet. This is the Signatures list from the Guardian's VAA struct. Note that a BigTable column can store many values (aka "cells") for a single row, unique by timestamp. This column will hold cells containing a list of signatures, with each cell list containing one more signature than the previous cell. This will show the order that signatures accumulate.

- `QuorumState:SignedVAA` the VAA with the signatures that contributed to quorum.
