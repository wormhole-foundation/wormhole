## Wormhole event BigTable schema

### Row Keys

Row keys contain the MessageID, delimited by colons, like so: `EmitterChain:EmitterAddress:Sequence`.

- `EmitterAddress` left padded with `0`s to 32 bytes, then hex encoded.

- `Sequence` left padded with `0`s to 16 characters, so rows are ordered in the sequence they occured. BigTable Rows are sorted lexicographically by row key.

BigTable can only be queried for data in the row key. Only row key data is indexed. You cannot query based on the value of a column; however you may filter based on column value.

### Column Families

BigTable requires that columns are within a "Column family". Families group columns that store related data. Grouping columns is useful for efficient reads, as you may specify which families you want returned.

The column families listed below represent data unique to a phase of the attestation lifecycle.

- `MessagePublication` holds data about a user's interaction with a Wormhole contract. Contains data from the Guardian's VAA struct.

- `QuorumState` stores the signed VAA once quorum is reached.

- `TokenTransferPayload` stores the decoded payload of transfer messages.

- `AssetMetaPayload` stores the decoded payload of asset metadata messages.

- `NFTTransferPayload` stores the decoded payload of NFT transfer messages.

- `TokenTransferDetails` stores information about the transfer.

- `ChainDetails` stores chain-native data supplimented from external source(s).

### Column Qualifiers

Each column qualifier below is prefixed with its column family.

#### MessagePublication
- `MessagePublication:Version` Version of the VAA schema.
- `MessagePublication:GuardianSetIndex` The index of the active Guardian set.
- `MessagePublication:Timestamp` Timestamp when the VAA was created by the Guardian.
- `MessagePublication:Nonce` Nonce of the user's transaction.
- `MessagePublication:Sequence` Sequence from the interaction with the Wormhole contract.
- `MessagePublication:EmitterChain` The chain the message was emitted on.
- `MessagePublication:EmitterAddress` The address of the contract that emitted the message.
- `MessagePublication:InitiatingTxID` The transaction identifier of the user's interaction with the contract.
- `MessagePublication:Payload` The payload of the user's message.

#### QuorumState
- `QuorumState:SignedVAA` the VAA with the signatures that contributed to quorum.

#### TokenTransferPayload
- `TokenTransferPayload:PayloadId` the payload identifier of the payload.
- `TokenTransferPayload:Amount` the amount of the transfer.
- `TokenTransferPayload:OriginAddress` the address the transfer originates from.
- `TokenTransferPayload:OriginChain` the chain identifier of the chain the transfer originates from.
- `TokenTransferPayload:TargetAddress` the destination address of the transfer.
- `TokenTransferPayload:TargetChain` the destination chain identifier of the transfer.

#### AssetMetaPayload
- `AssetMetaPayload:PayloadId` the payload identifier of the payload.
- `AssetMetaPayload:TokenAddress` the address of the token. left padded with `0`s to 32 bytes.
- `AssetMetaPayload:TokenChain`  the chain identifier of the chain the transfer originates from.
- `AssetMetaPayload:Decimals` the number of decimals of the token.
- `AssetMetaPayload:Symbol` the ticker symbol of the token.
- `AssetMetaPayload:Name` the name of the token.

#### NFTTransferPayload
- `NFTTransferPayload:PayloadId` the payload identifier of the payload.
- `NFTTransferPayload:OriginAddress` the address the transfer originates from.
- `NFTTransferPayload:OriginChain` the chain identifier of the chain the transfer originates from.
- `NFTTransferPayload:Symbol` the symbol of the nft.
- `NFTTransferPayload:Name` the name of the nft.
- `NFTTransferPayload:TokenId` the token identifier of the nft.
- `NFTTransferPayload:URI` the URI of the nft.
- `NFTTransferPayload:TargetAddress`  the destination address of the transfer.
- `NFTTransferPayload:TargetChain` the destination chain identifier of the transfer.

#### TokenTransferDetails
- `TokenTransferDetails:Amount` the amount transfered.
- `TokenTransferDetails:NotionalUSD` the notional value of the transfer in USD.
- `TokenTransferDetails:OriginSymbol` the symbol of the token sent to wormhole.
- `TokenTransferDetails:OriginName` the name of the token sent to wormhole.
- `TokenTransferDetails:OriginTokenAddress` the address of the token sent to wormhole.

#### ChainDetails
- `ChainDetails:SenderAddress` the native address that sent the message.
- `ChainDetails:ReceiverAddress` the native address that received the message.
