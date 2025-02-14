//@ts-nocheck
/* eslint-disable */
/* tslint:disable */
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

/**
* Event allows application developers to attach additional information to
ResponseBeginBlock, ResponseEndBlock, ResponseCheckTx and ResponseDeliverTx.
Later, transactions may be queried using these events.
*/
export interface AbciEvent {
  type?: string;
  attributes?: AbciEventAttribute[];
}

/**
 * EventAttribute is a single key-value pair, associated with an event.
 */
export interface AbciEventAttribute {
  key?: string;
  value?: string;

  /** nondeterministic */
  index?: boolean;
}

/**
 * Result is the union of ResponseFormat and ResponseCheckTx.
 */
export interface Abciv1Beta1Result {
  /**
   * Data is any data returned from message or handler execution. It MUST be
   * length prefixed in order to separate data from multiple message executions.
   * Deprecated. This field is still populated, but prefer msg_response instead
   * because it also contains the Msg response typeURL.
   * @format byte
   */
  data?: string;

  /** Log contains the log information from message or handler execution. */
  log?: string;

  /**
   * Events contains a slice of Event objects that were emitted during message
   * or handler execution.
   */
  events?: AbciEvent[];

  /**
   * msg_responses contains the Msg handler responses type packed in Anys.
   *
   * Since: cosmos-sdk 0.46
   */
  msg_responses?: ProtobufAny[];
}

export interface CryptoPublicKey {
  /** @format byte */
  ed25519?: string;

  /** @format byte */
  secp256k1?: string;
}

/**
* `Any` contains an arbitrary serialized protocol buffer message along with a
URL that describes the type of the serialized message.

Protobuf library provides support to pack/unpack Any values in the form
of utility functions or additional generated methods of the Any type.

Example 1: Pack and unpack a message in C++.

    Foo foo = ...;
    Any any;
    any.PackFrom(foo);
    ...
    if (any.UnpackTo(&foo)) {
      ...
    }

Example 2: Pack and unpack a message in Java.

    Foo foo = ...;
    Any any = Any.pack(foo);
    ...
    if (any.is(Foo.class)) {
      foo = any.unpack(Foo.class);
    }

 Example 3: Pack and unpack a message in Python.

    foo = Foo(...)
    any = Any()
    any.Pack(foo)
    ...
    if any.Is(Foo.DESCRIPTOR):
      any.Unpack(foo)
      ...

 Example 4: Pack and unpack a message in Go

     foo := &pb.Foo{...}
     any, err := anypb.New(foo)
     if err != nil {
       ...
     }
     ...
     foo := &pb.Foo{}
     if err := any.UnmarshalTo(foo); err != nil {
       ...
     }

The pack methods provided by protobuf library will by default use
'type.googleapis.com/full.type.name' as the type URL and the unpack
methods only use the fully qualified type name after the last '/'
in the type URL, for example "foo.bar.com/x/y.z" will yield type
name "y.z".


JSON
====
The JSON representation of an `Any` value uses the regular
representation of the deserialized, embedded message, with an
additional field `@type` which contains the type URL. Example:

    package google.profile;
    message Person {
      string first_name = 1;
      string last_name = 2;
    }

    {
      "@type": "type.googleapis.com/google.profile.Person",
      "firstName": <string>,
      "lastName": <string>
    }

If the embedded message type is well-known and has a custom JSON
representation, that representation will be embedded adding a field
`value` which holds the custom JSON in addition to the `@type`
field. Example (for message [google.protobuf.Duration][]):

    {
      "@type": "type.googleapis.com/google.protobuf.Duration",
      "value": "1.212s"
    }
*/
export interface ProtobufAny {
  /**
   * A URL/resource name that uniquely identifies the type of the serialized
   * protocol buffer message. This string must contain at least
   * one "/" character. The last segment of the URL's path must represent
   * the fully qualified name of the type (as in
   * `path/google.protobuf.Duration`). The name should be in a canonical form
   * (e.g., leading "." is not accepted).
   *
   * In practice, teams usually precompile into the binary all types that they
   * expect it to use in the context of Any. However, for URLs which use the
   * scheme `http`, `https`, or no scheme, one can optionally set up a type
   * server that maps type URLs to message definitions as follows:
   * * If no scheme is provided, `https` is assumed.
   * * An HTTP GET on the URL must yield a [google.protobuf.Type][]
   *   value in binary format, or produce an error.
   * * Applications are allowed to cache lookup results based on the
   *   URL, or have them precompiled into a binary to avoid any
   *   lookup. Therefore, binary compatibility needs to be preserved
   *   on changes to types. (Use versioned type names to manage
   *   breaking changes.)
   * Note: this functionality is not currently available in the official
   * protobuf release, and it is not used for type URLs beginning with
   * type.googleapis.com.
   * Schemes other than `http`, `https` (or the empty scheme) might be
   * used with implementation specific semantics.
   */
  "@type"?: string;
}

export interface RpcStatus {
  /** @format int32 */
  code?: number;
  message?: string;
  details?: ProtobufAny[];
}

export interface TenderminttypesData {
  /**
   * Txs that will be applied by state @ block.Height+1.
   * NOTE: not all txs here are valid.  We're just agreeing on the order first.
   * This means that block.AppHash does not include these txs.
   */
  txs?: string[];
}

export interface TenderminttypesValidator {
  /** @format byte */
  address?: string;
  pub_key?: CryptoPublicKey;

  /** @format int64 */
  voting_power?: string;

  /** @format int64 */
  proposer_priority?: string;
}

export interface TypesBlock {
  /** Header defines the structure of a block header. */
  header?: TypesHeader;
  data?: TenderminttypesData;
  evidence?: TypesEvidenceList;

  /** Commit contains the evidence that a block was committed by a set of validators. */
  last_commit?: TypesCommit;
}

export interface TypesBlockID {
  /** @format byte */
  hash?: string;
  part_set_header?: TypesPartSetHeader;
}

export enum TypesBlockIDFlag {
  BLOCK_ID_FLAG_UNKNOWN = "BLOCK_ID_FLAG_UNKNOWN",
  BLOCK_ID_FLAG_ABSENT = "BLOCK_ID_FLAG_ABSENT",
  BLOCK_ID_FLAG_COMMIT = "BLOCK_ID_FLAG_COMMIT",
  BLOCK_ID_FLAG_NIL = "BLOCK_ID_FLAG_NIL",
}

/**
 * Commit contains the evidence that a block was committed by a set of validators.
 */
export interface TypesCommit {
  /** @format int64 */
  height?: string;

  /** @format int32 */
  round?: number;
  block_id?: TypesBlockID;
  signatures?: TypesCommitSig[];
}

/**
 * CommitSig is a part of the Vote included in a Commit.
 */
export interface TypesCommitSig {
  block_id_flag?: TypesBlockIDFlag;

  /** @format byte */
  validator_address?: string;

  /** @format date-time */
  timestamp?: string;

  /** @format byte */
  signature?: string;
}

/**
 * DuplicateVoteEvidence contains evidence of a validator signed two conflicting votes.
 */
export interface TypesDuplicateVoteEvidence {
  /**
   * Vote represents a prevote, precommit, or commit vote from validators for
   * consensus.
   */
  vote_a?: TypesVote;

  /**
   * Vote represents a prevote, precommit, or commit vote from validators for
   * consensus.
   */
  vote_b?: TypesVote;

  /** @format int64 */
  total_voting_power?: string;

  /** @format int64 */
  validator_power?: string;

  /** @format date-time */
  timestamp?: string;
}

export interface TypesEvidence {
  /** DuplicateVoteEvidence contains evidence of a validator signed two conflicting votes. */
  duplicate_vote_evidence?: TypesDuplicateVoteEvidence;

  /** LightClientAttackEvidence contains evidence of a set of validators attempting to mislead a light client. */
  light_client_attack_evidence?: TypesLightClientAttackEvidence;
}

export interface TypesEvidenceList {
  evidence?: TypesEvidence[];
}

/**
 * Header defines the structure of a block header.
 */
export interface TypesHeader {
  /**
   * basic block info
   * Consensus captures the consensus rules for processing a block in the blockchain,
   * including all blockchain data structures and the rules of the application's
   * state transition machine.
   */
  version?: VersionConsensus;
  chain_id?: string;

  /** @format int64 */
  height?: string;

  /** @format date-time */
  time?: string;

  /** prev block info */
  last_block_id?: TypesBlockID;

  /**
   * hashes of block data
   * commit from validators from the last block
   * @format byte
   */
  last_commit_hash?: string;

  /**
   * transactions
   * @format byte
   */
  data_hash?: string;

  /**
   * hashes from the app output from the prev block
   * validators for the current block
   * @format byte
   */
  validators_hash?: string;

  /**
   * validators for the next block
   * @format byte
   */
  next_validators_hash?: string;

  /**
   * consensus params for current block
   * @format byte
   */
  consensus_hash?: string;

  /**
   * state after txs from the previous block
   * @format byte
   */
  app_hash?: string;

  /**
   * root hash of all results from the txs from the previous block
   * @format byte
   */
  last_results_hash?: string;

  /**
   * consensus info
   * evidence included in the block
   * @format byte
   */
  evidence_hash?: string;

  /**
   * original proposer of the block
   * @format byte
   */
  proposer_address?: string;
}

export interface TypesLightBlock {
  signed_header?: TypesSignedHeader;
  validator_set?: TypesValidatorSet;
}

/**
 * LightClientAttackEvidence contains evidence of a set of validators attempting to mislead a light client.
 */
export interface TypesLightClientAttackEvidence {
  conflicting_block?: TypesLightBlock;

  /** @format int64 */
  common_height?: string;
  byzantine_validators?: TenderminttypesValidator[];

  /** @format int64 */
  total_voting_power?: string;

  /** @format date-time */
  timestamp?: string;
}

export interface TypesPartSetHeader {
  /** @format int64 */
  total?: number;

  /** @format byte */
  hash?: string;
}

export interface TypesSignedHeader {
  /** Header defines the structure of a block header. */
  header?: TypesHeader;

  /** Commit contains the evidence that a block was committed by a set of validators. */
  commit?: TypesCommit;
}

/**
* SignedMsgType is a type of signed message in the consensus.

 - SIGNED_MSG_TYPE_PREVOTE: Votes
 - SIGNED_MSG_TYPE_PROPOSAL: Proposals
*/
export enum TypesSignedMsgType {
  SIGNED_MSG_TYPE_UNKNOWN = "SIGNED_MSG_TYPE_UNKNOWN",
  SIGNED_MSG_TYPE_PREVOTE = "SIGNED_MSG_TYPE_PREVOTE",
  SIGNED_MSG_TYPE_PRECOMMIT = "SIGNED_MSG_TYPE_PRECOMMIT",
  SIGNED_MSG_TYPE_PROPOSAL = "SIGNED_MSG_TYPE_PROPOSAL",
}

export interface TypesValidatorSet {
  validators?: TenderminttypesValidator[];
  proposer?: TenderminttypesValidator;

  /** @format int64 */
  total_voting_power?: string;
}

/**
* Vote represents a prevote, precommit, or commit vote from validators for
consensus.
*/
export interface TypesVote {
  /**
   * SignedMsgType is a type of signed message in the consensus.
   *
   *  - SIGNED_MSG_TYPE_PREVOTE: Votes
   *  - SIGNED_MSG_TYPE_PROPOSAL: Proposals
   */
  type?: TypesSignedMsgType;

  /** @format int64 */
  height?: string;

  /** @format int32 */
  round?: number;

  /** zero if vote is nil. */
  block_id?: TypesBlockID;

  /** @format date-time */
  timestamp?: string;

  /** @format byte */
  validator_address?: string;

  /** @format int32 */
  validator_index?: number;

  /** @format byte */
  signature?: string;
}

/**
 * ABCIMessageLog defines a structure containing an indexed tx ABCI message log.
 */
export interface V1Beta1ABCIMessageLog {
  /** @format int64 */
  msg_index?: number;
  log?: string;

  /**
   * Events contains a slice of Event objects that were emitted during some
   * execution.
   */
  events?: V1Beta1StringEvent[];
}

/**
* Attribute defines an attribute wrapper where the key and value are
strings instead of raw bytes.
*/
export interface V1Beta1Attribute {
  key?: string;
  value?: string;
}

/**
* AuthInfo describes the fee and signer modes that are used to sign a
transaction.
*/
export interface V1Beta1AuthInfo {
  /**
   * signer_infos defines the signing modes for the required signers. The number
   * and order of elements must match the required signers from TxBody's
   * messages. The first element is the primary signer and the one which pays
   * the fee.
   */
  signer_infos?: V1Beta1SignerInfo[];

  /**
   * Fee is the fee and gas limit for the transaction. The first signer is the
   * primary signer and the one which pays the fee. The fee can be calculated
   * based on the cost of evaluating the body and doing signature verification
   * of the signers. This can be estimated via simulation.
   */
  fee?: V1Beta1Fee;

  /**
   * Tip is the optional tip used for transactions fees paid in another denom.
   *
   * This field is ignored if the chain didn't enable tips, i.e. didn't add the
   * `TipDecorator` in its posthandler.
   * Since: cosmos-sdk 0.46
   */
  tip?: V1Beta1Tip;
}

/**
* BroadcastMode specifies the broadcast mode for the TxService.Broadcast RPC method.

 - BROADCAST_MODE_UNSPECIFIED: zero-value for mode ordering
 - BROADCAST_MODE_BLOCK: DEPRECATED: use BROADCAST_MODE_SYNC instead,
BROADCAST_MODE_BLOCK is not supported by the SDK from v0.47.x onwards.
 - BROADCAST_MODE_SYNC: BROADCAST_MODE_SYNC defines a tx broadcasting mode where the client waits for
a CheckTx execution response only.
 - BROADCAST_MODE_ASYNC: BROADCAST_MODE_ASYNC defines a tx broadcasting mode where the client returns
immediately.
*/
export enum V1Beta1BroadcastMode {
  BROADCAST_MODE_UNSPECIFIED = "BROADCAST_MODE_UNSPECIFIED",
  BROADCAST_MODE_BLOCK = "BROADCAST_MODE_BLOCK",
  BROADCAST_MODE_SYNC = "BROADCAST_MODE_SYNC",
  BROADCAST_MODE_ASYNC = "BROADCAST_MODE_ASYNC",
}

/**
* BroadcastTxRequest is the request type for the Service.BroadcastTxRequest
RPC method.
*/
export interface V1Beta1BroadcastTxRequest {
  /**
   * tx_bytes is the raw transaction.
   * @format byte
   */
  tx_bytes?: string;

  /**
   * BroadcastMode specifies the broadcast mode for the TxService.Broadcast RPC method.
   *
   *  - BROADCAST_MODE_UNSPECIFIED: zero-value for mode ordering
   *  - BROADCAST_MODE_BLOCK: DEPRECATED: use BROADCAST_MODE_SYNC instead,
   * BROADCAST_MODE_BLOCK is not supported by the SDK from v0.47.x onwards.
   *  - BROADCAST_MODE_SYNC: BROADCAST_MODE_SYNC defines a tx broadcasting mode where the client waits for
   * a CheckTx execution response only.
   *  - BROADCAST_MODE_ASYNC: BROADCAST_MODE_ASYNC defines a tx broadcasting mode where the client returns
   * immediately.
   */
  mode?: V1Beta1BroadcastMode;
}

/**
* BroadcastTxResponse is the response type for the
Service.BroadcastTx method.
*/
export interface V1Beta1BroadcastTxResponse {
  /** tx_response is the queried TxResponses. */
  tx_response?: V1Beta1TxResponse;
}

/**
* Coin defines a token with a denomination and an amount.

NOTE: The amount field is an Int which implements the custom method
signatures required by gogoproto.
*/
export interface V1Beta1Coin {
  denom?: string;
  amount?: string;
}

/**
* CompactBitArray is an implementation of a space efficient bit array.
This is used to ensure that the encoded data takes up a minimal amount of
space after proto encoding.
This is not thread safe, and is not intended for concurrent usage.
*/
export interface V1Beta1CompactBitArray {
  /** @format int64 */
  extra_bits_stored?: number;

  /** @format byte */
  elems?: string;
}

/**
* Fee includes the amount of coins paid in fees and the maximum
gas to be used by the transaction. The ratio yields an effective "gasprice",
which must be above some miminum to be accepted into the mempool.
*/
export interface V1Beta1Fee {
  /** amount is the amount of coins to be paid as a fee */
  amount?: V1Beta1Coin[];

  /**
   * gas_limit is the maximum gas that can be used in transaction processing
   * before an out of gas error occurs
   * @format uint64
   */
  gas_limit?: string;

  /**
   * if unset, the first signer is responsible for paying the fees. If set, the specified account must pay the fees.
   * the payer must be a tx signer (and thus have signed this field in AuthInfo).
   * setting this field does *not* change the ordering of required signers for the transaction.
   */
  payer?: string;

  /**
   * if set, the fee payer (either the first signer or the value of the payer field) requests that a fee grant be used
   * to pay fees instead of the fee payer's own balance. If an appropriate fee grant does not exist or the chain does
   * not support fee grants, this will fail
   */
  granter?: string;
}

/**
 * GasInfo defines tx execution gas context.
 */
export interface V1Beta1GasInfo {
  /**
   * GasWanted is the maximum units of work we allow this tx to perform.
   * @format uint64
   */
  gas_wanted?: string;

  /**
   * GasUsed is the amount of gas actually consumed.
   * @format uint64
   */
  gas_used?: string;
}

/**
* GetBlockWithTxsResponse is the response type for the Service.GetBlockWithTxs method.

Since: cosmos-sdk 0.45.2
*/
export interface V1Beta1GetBlockWithTxsResponse {
  /** txs are the transactions in the block. */
  txs?: V1Beta1Tx[];
  block_id?: TypesBlockID;
  block?: TypesBlock;

  /** pagination defines a pagination for the response. */
  pagination?: V1Beta1PageResponse;
}

/**
 * GetTxResponse is the response type for the Service.GetTx method.
 */
export interface V1Beta1GetTxResponse {
  /** tx is the queried transaction. */
  tx?: V1Beta1Tx;

  /** tx_response is the queried TxResponses. */
  tx_response?: V1Beta1TxResponse;
}

/**
* GetTxsEventResponse is the response type for the Service.TxsByEvents
RPC method.
*/
export interface V1Beta1GetTxsEventResponse {
  /** txs is the list of queried transactions. */
  txs?: V1Beta1Tx[];

  /** tx_responses is the list of queried TxResponses. */
  tx_responses?: V1Beta1TxResponse[];

  /**
   * pagination defines a pagination for the response.
   * Deprecated post v0.46.x: use total instead.
   */
  pagination?: V1Beta1PageResponse;

  /**
   * total is total number of results available
   * @format uint64
   */
  total?: string;
}

/**
 * ModeInfo describes the signing mode of a single or nested multisig signer.
 */
export interface V1Beta1ModeInfo {
  /** single represents a single signer */
  single?: V1Beta1ModeInfoSingle;

  /** multi represents a nested multisig signer */
  multi?: V1Beta1ModeInfoMulti;
}

export interface V1Beta1ModeInfoMulti {
  /**
   * bitarray specifies which keys within the multisig are signing
   * CompactBitArray is an implementation of a space efficient bit array.
   * This is used to ensure that the encoded data takes up a minimal amount of
   * space after proto encoding.
   * This is not thread safe, and is not intended for concurrent usage.
   */
  bitarray?: V1Beta1CompactBitArray;

  /**
   * mode_infos is the corresponding modes of the signers of the multisig
   * which could include nested multisig public keys
   */
  mode_infos?: V1Beta1ModeInfo[];
}

export interface V1Beta1ModeInfoSingle {
  /**
   * mode is the signing mode of the single signer
   * SignMode represents a signing mode with its own security guarantees.
   *
   * This enum should be considered a registry of all known sign modes
   * in the Cosmos ecosystem. Apps are not expected to support all known
   * sign modes. Apps that would like to support custom  sign modes are
   * encouraged to open a small PR against this file to add a new case
   * to this SignMode enum describing their sign mode so that different
   * apps have a consistent version of this enum.
   *  - SIGN_MODE_UNSPECIFIED: SIGN_MODE_UNSPECIFIED specifies an unknown signing mode and will be
   * rejected.
   *  - SIGN_MODE_DIRECT: SIGN_MODE_DIRECT specifies a signing mode which uses SignDoc and is
   * verified with raw bytes from Tx.
   *  - SIGN_MODE_TEXTUAL: SIGN_MODE_TEXTUAL is a future signing mode that will verify some
   * human-readable textual representation on top of the binary representation
   * from SIGN_MODE_DIRECT. It is currently not supported.
   *  - SIGN_MODE_DIRECT_AUX: SIGN_MODE_DIRECT_AUX specifies a signing mode which uses
   * SignDocDirectAux. As opposed to SIGN_MODE_DIRECT, this sign mode does not
   * require signers signing over other signers' `signer_info`. It also allows
   * for adding Tips in transactions.
   * Since: cosmos-sdk 0.46
   *  - SIGN_MODE_LEGACY_AMINO_JSON: SIGN_MODE_LEGACY_AMINO_JSON is a backwards compatibility mode which uses
   * Amino JSON and will be removed in the future.
   *  - SIGN_MODE_EIP_191: SIGN_MODE_EIP_191 specifies the sign mode for EIP 191 signing on the Cosmos
   * SDK. Ref: https://eips.ethereum.org/EIPS/eip-191
   * Currently, SIGN_MODE_EIP_191 is registered as a SignMode enum variant,
   * but is not implemented on the SDK by default. To enable EIP-191, you need
   * to pass a custom `TxConfig` that has an implementation of
   * `SignModeHandler` for EIP-191. The SDK may decide to fully support
   * EIP-191 in the future.
   * Since: cosmos-sdk 0.45.2
   */
  mode?: V1Beta1SignMode;
}

/**
* - ORDER_BY_UNSPECIFIED: ORDER_BY_UNSPECIFIED specifies an unknown sorting order. OrderBy defaults to ASC in this case.
 - ORDER_BY_ASC: ORDER_BY_ASC defines ascending order
 - ORDER_BY_DESC: ORDER_BY_DESC defines descending order
*/
export enum V1Beta1OrderBy {
  ORDER_BY_UNSPECIFIED = "ORDER_BY_UNSPECIFIED",
  ORDER_BY_ASC = "ORDER_BY_ASC",
  ORDER_BY_DESC = "ORDER_BY_DESC",
}

/**
* message SomeRequest {
         Foo some_parameter = 1;
         PageRequest pagination = 2;
 }
*/
export interface V1Beta1PageRequest {
  /**
   * key is a value returned in PageResponse.next_key to begin
   * querying the next page most efficiently. Only one of offset or key
   * should be set.
   * @format byte
   */
  key?: string;

  /**
   * offset is a numeric offset that can be used when key is unavailable.
   * It is less efficient than using key. Only one of offset or key should
   * be set.
   * @format uint64
   */
  offset?: string;

  /**
   * limit is the total number of results to be returned in the result page.
   * If left empty it will default to a value to be set by each app.
   * @format uint64
   */
  limit?: string;

  /**
   * count_total is set to true  to indicate that the result set should include
   * a count of the total number of items available for pagination in UIs.
   * count_total is only respected when offset is used. It is ignored when key
   * is set.
   */
  count_total?: boolean;

  /**
   * reverse is set to true if results are to be returned in the descending order.
   *
   * Since: cosmos-sdk 0.43
   */
  reverse?: boolean;
}

/**
* PageResponse is to be embedded in gRPC response messages where the
corresponding request message has used PageRequest.

 message SomeResponse {
         repeated Bar results = 1;
         PageResponse page = 2;
 }
*/
export interface V1Beta1PageResponse {
  /**
   * next_key is the key to be passed to PageRequest.key to
   * query the next page most efficiently. It will be empty if
   * there are no more results.
   * @format byte
   */
  next_key?: string;

  /**
   * total is total number of results available if PageRequest.count_total
   * was set, its value is undefined otherwise
   * @format uint64
   */
  total?: string;
}

/**
* SignMode represents a signing mode with its own security guarantees.

This enum should be considered a registry of all known sign modes
in the Cosmos ecosystem. Apps are not expected to support all known
sign modes. Apps that would like to support custom  sign modes are
encouraged to open a small PR against this file to add a new case
to this SignMode enum describing their sign mode so that different
apps have a consistent version of this enum.

 - SIGN_MODE_UNSPECIFIED: SIGN_MODE_UNSPECIFIED specifies an unknown signing mode and will be
rejected.
 - SIGN_MODE_DIRECT: SIGN_MODE_DIRECT specifies a signing mode which uses SignDoc and is
verified with raw bytes from Tx.
 - SIGN_MODE_TEXTUAL: SIGN_MODE_TEXTUAL is a future signing mode that will verify some
human-readable textual representation on top of the binary representation
from SIGN_MODE_DIRECT. It is currently not supported.
 - SIGN_MODE_DIRECT_AUX: SIGN_MODE_DIRECT_AUX specifies a signing mode which uses
SignDocDirectAux. As opposed to SIGN_MODE_DIRECT, this sign mode does not
require signers signing over other signers' `signer_info`. It also allows
for adding Tips in transactions.

Since: cosmos-sdk 0.46
 - SIGN_MODE_LEGACY_AMINO_JSON: SIGN_MODE_LEGACY_AMINO_JSON is a backwards compatibility mode which uses
Amino JSON and will be removed in the future.
 - SIGN_MODE_EIP_191: SIGN_MODE_EIP_191 specifies the sign mode for EIP 191 signing on the Cosmos
SDK. Ref: https://eips.ethereum.org/EIPS/eip-191

Currently, SIGN_MODE_EIP_191 is registered as a SignMode enum variant,
but is not implemented on the SDK by default. To enable EIP-191, you need
to pass a custom `TxConfig` that has an implementation of
`SignModeHandler` for EIP-191. The SDK may decide to fully support
EIP-191 in the future.

Since: cosmos-sdk 0.45.2
*/
export enum V1Beta1SignMode {
  SIGN_MODE_UNSPECIFIED = "SIGN_MODE_UNSPECIFIED",
  SIGN_MODE_DIRECT = "SIGN_MODE_DIRECT",
  SIGN_MODE_TEXTUAL = "SIGN_MODE_TEXTUAL",
  SIGN_MODE_DIRECT_AUX = "SIGN_MODE_DIRECT_AUX",
  SIGN_MODE_LEGACY_AMINO_JSON = "SIGN_MODE_LEGACY_AMINO_JSON",
  SIGNMODEEIP191 = "SIGN_MODE_EIP_191",
}

/**
* SignerInfo describes the public key and signing mode of a single top-level
signer.
*/
export interface V1Beta1SignerInfo {
  /**
   * public_key is the public key of the signer. It is optional for accounts
   * that already exist in state. If unset, the verifier can use the required \
   * signer address for this position and lookup the public key.
   */
  public_key?: ProtobufAny;

  /**
   * mode_info describes the signing mode of the signer and is a nested
   * structure to support nested multisig pubkey's
   * ModeInfo describes the signing mode of a single or nested multisig signer.
   */
  mode_info?: V1Beta1ModeInfo;

  /**
   * sequence is the sequence of the account, which describes the
   * number of committed transactions signed by a given address. It is used to
   * prevent replay attacks.
   * @format uint64
   */
  sequence?: string;
}

/**
* SimulateRequest is the request type for the Service.Simulate
RPC method.
*/
export interface V1Beta1SimulateRequest {
  /**
   * tx is the transaction to simulate.
   * Deprecated. Send raw tx bytes instead.
   */
  tx?: V1Beta1Tx;

  /**
   * tx_bytes is the raw transaction.
   *
   * Since: cosmos-sdk 0.43
   * @format byte
   */
  tx_bytes?: string;
}

/**
* SimulateResponse is the response type for the
Service.SimulateRPC method.
*/
export interface V1Beta1SimulateResponse {
  /** gas_info is the information about gas used in the simulation. */
  gas_info?: V1Beta1GasInfo;

  /** result is the result of the simulation. */
  result?: Abciv1Beta1Result;
}

/**
* StringEvent defines en Event object wrapper where all the attributes
contain key/value pairs that are strings instead of raw bytes.
*/
export interface V1Beta1StringEvent {
  type?: string;
  attributes?: V1Beta1Attribute[];
}

/**
* Tip is the tip used for meta-transactions.

Since: cosmos-sdk 0.46
*/
export interface V1Beta1Tip {
  /** amount is the amount of the tip */
  amount?: V1Beta1Coin[];

  /** tipper is the address of the account paying for the tip */
  tipper?: string;
}

/**
 * Tx is the standard type used for broadcasting transactions.
 */
export interface V1Beta1Tx {
  /**
   * body is the processable content of the transaction
   * TxBody is the body of a transaction that all signers sign over.
   */
  body?: V1Beta1TxBody;

  /**
   * auth_info is the authorization related content of the transaction,
   * specifically signers, signer modes and fee
   * AuthInfo describes the fee and signer modes that are used to sign a
   * transaction.
   */
  auth_info?: V1Beta1AuthInfo;

  /**
   * signatures is a list of signatures that matches the length and order of
   * AuthInfo's signer_infos to allow connecting signature meta information like
   * public key and signing mode by position.
   */
  signatures?: string[];
}

/**
 * TxBody is the body of a transaction that all signers sign over.
 */
export interface V1Beta1TxBody {
  /**
   * messages is a list of messages to be executed. The required signers of
   * those messages define the number and order of elements in AuthInfo's
   * signer_infos and Tx's signatures. Each required signer address is added to
   * the list only the first time it occurs.
   * By convention, the first required signer (usually from the first message)
   * is referred to as the primary signer and pays the fee for the whole
   * transaction.
   */
  messages?: ProtobufAny[];

  /**
   * memo is any arbitrary note/comment to be added to the transaction.
   * WARNING: in clients, any publicly exposed text should not be called memo,
   * but should be called `note` instead (see https://github.com/cosmos/cosmos-sdk/issues/9122).
   */
  memo?: string;

  /**
   * timeout is the block height after which this transaction will not
   * be processed by the chain
   * @format uint64
   */
  timeout_height?: string;

  /**
   * extension_options are arbitrary options that can be added by chains
   * when the default options are not sufficient. If any of these are present
   * and can't be handled, the transaction will be rejected
   */
  extension_options?: ProtobufAny[];

  /**
   * extension_options are arbitrary options that can be added by chains
   * when the default options are not sufficient. If any of these are present
   * and can't be handled, they will be ignored
   */
  non_critical_extension_options?: ProtobufAny[];
}

/**
* TxDecodeAminoRequest is the request type for the Service.TxDecodeAmino
RPC method.

Since: cosmos-sdk 0.47
*/
export interface V1Beta1TxDecodeAminoRequest {
  /** @format byte */
  amino_binary?: string;
}

/**
* TxDecodeAminoResponse is the response type for the Service.TxDecodeAmino
RPC method.

Since: cosmos-sdk 0.47
*/
export interface V1Beta1TxDecodeAminoResponse {
  amino_json?: string;
}

/**
* TxDecodeRequest is the request type for the Service.TxDecode
RPC method.

Since: cosmos-sdk 0.47
*/
export interface V1Beta1TxDecodeRequest {
  /**
   * tx_bytes is the raw transaction.
   * @format byte
   */
  tx_bytes?: string;
}

/**
* TxDecodeResponse is the response type for the
Service.TxDecode method.

Since: cosmos-sdk 0.47
*/
export interface V1Beta1TxDecodeResponse {
  /** tx is the decoded transaction. */
  tx?: V1Beta1Tx;
}

/**
* TxEncodeAminoRequest is the request type for the Service.TxEncodeAmino
RPC method.

Since: cosmos-sdk 0.47
*/
export interface V1Beta1TxEncodeAminoRequest {
  amino_json?: string;
}

/**
* TxEncodeAminoResponse is the response type for the Service.TxEncodeAmino
RPC method.

Since: cosmos-sdk 0.47
*/
export interface V1Beta1TxEncodeAminoResponse {
  /** @format byte */
  amino_binary?: string;
}

/**
* TxEncodeRequest is the request type for the Service.TxEncode
RPC method.

Since: cosmos-sdk 0.47
*/
export interface V1Beta1TxEncodeRequest {
  /** tx is the transaction to encode. */
  tx?: V1Beta1Tx;
}

/**
* TxEncodeResponse is the response type for the
Service.TxEncode method.

Since: cosmos-sdk 0.47
*/
export interface V1Beta1TxEncodeResponse {
  /**
   * tx_bytes is the encoded transaction bytes.
   * @format byte
   */
  tx_bytes?: string;
}

/**
* TxResponse defines a structure containing relevant tx data and metadata. The
tags are stringified and the log is JSON decoded.
*/
export interface V1Beta1TxResponse {
  /**
   * The block height
   * @format int64
   */
  height?: string;

  /** The transaction hash. */
  txhash?: string;

  /** Namespace for the Code */
  codespace?: string;

  /**
   * Response code.
   * @format int64
   */
  code?: number;

  /** Result bytes, if any. */
  data?: string;

  /**
   * The output of the application's logger (raw string). May be
   * non-deterministic.
   */
  raw_log?: string;

  /** The output of the application's logger (typed). May be non-deterministic. */
  logs?: V1Beta1ABCIMessageLog[];

  /** Additional information. May be non-deterministic. */
  info?: string;

  /**
   * Amount of gas requested for transaction.
   * @format int64
   */
  gas_wanted?: string;

  /**
   * Amount of gas consumed by transaction.
   * @format int64
   */
  gas_used?: string;

  /** The request transaction bytes. */
  tx?: ProtobufAny;

  /**
   * Time of the previous block. For heights > 1, it's the weighted median of
   * the timestamps of the valid votes in the block.LastCommit. For height == 1,
   * it's genesis time.
   */
  timestamp?: string;

  /**
   * Events defines all the events emitted by processing a transaction. Note,
   * these events include those emitted by processing all the messages and those
   * emitted from the ante. Whereas Logs contains the events, with
   * additional metadata, emitted only by processing the messages.
   *
   * Since: cosmos-sdk 0.42.11, 0.44.5, 0.45
   */
  events?: AbciEvent[];
}

/**
* Consensus captures the consensus rules for processing a block in the blockchain,
including all blockchain data structures and the rules of the application's
state transition machine.
*/
export interface VersionConsensus {
  /** @format uint64 */
  block?: string;

  /** @format uint64 */
  app?: string;
}

import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse, ResponseType } from "axios";

export type QueryParamsType = Record<string | number, any>;

export interface FullRequestParams extends Omit<AxiosRequestConfig, "data" | "params" | "url" | "responseType"> {
  /** set parameter to `true` for call `securityWorker` for this request */
  secure?: boolean;
  /** request path */
  path: string;
  /** content type of request body */
  type?: ContentType;
  /** query params */
  query?: QueryParamsType;
  /** format of response (i.e. response.json() -> format: "json") */
  format?: ResponseType;
  /** request body */
  body?: unknown;
}

export type RequestParams = Omit<FullRequestParams, "body" | "method" | "query" | "path">;

export interface ApiConfig<SecurityDataType = unknown> extends Omit<AxiosRequestConfig, "data" | "cancelToken"> {
  securityWorker?: (
    securityData: SecurityDataType | null,
  ) => Promise<AxiosRequestConfig | void> | AxiosRequestConfig | void;
  secure?: boolean;
  format?: ResponseType;
}

export enum ContentType {
  Json = "application/json",
  FormData = "multipart/form-data",
  UrlEncoded = "application/x-www-form-urlencoded",
}

export class HttpClient<SecurityDataType = unknown> {
  public instance: AxiosInstance;
  private securityData: SecurityDataType | null = null;
  private securityWorker?: ApiConfig<SecurityDataType>["securityWorker"];
  private secure?: boolean;
  private format?: ResponseType;

  constructor({ securityWorker, secure, format, ...axiosConfig }: ApiConfig<SecurityDataType> = {}) {
    this.instance = axios.create({ ...axiosConfig, baseURL: axiosConfig.baseURL || "" });
    this.secure = secure;
    this.format = format;
    this.securityWorker = securityWorker;
  }

  public setSecurityData = (data: SecurityDataType | null) => {
    this.securityData = data;
  };

  private mergeRequestParams(params1: AxiosRequestConfig, params2?: AxiosRequestConfig): AxiosRequestConfig {
    return {
      ...this.instance.defaults,
      ...params1,
      ...(params2 || {}),
      headers: {
        ...(this.instance.defaults.headers || {}),
        ...(params1.headers || {}),
        ...((params2 && params2.headers) || {}),
      },
    };
  }

  private createFormData(input: Record<string, unknown>): FormData {
    return Object.keys(input || {}).reduce((formData, key) => {
      const property = input[key];
      formData.append(
        key,
        property instanceof Blob
          ? property
          : typeof property === "object" && property !== null
          ? JSON.stringify(property)
          : `${property}`,
      );
      return formData;
    }, new FormData());
  }

  public request = async <T = any, _E = any>({
    secure,
    path,
    type,
    query,
    format,
    body,
    ...params
  }: FullRequestParams): Promise<AxiosResponse<T>> => {
    const secureParams =
      ((typeof secure === "boolean" ? secure : this.secure) &&
        this.securityWorker &&
        (await this.securityWorker(this.securityData))) ||
      {};
    const requestParams = this.mergeRequestParams(params, secureParams);
    const responseFormat = (format && this.format) || void 0;

    if (type === ContentType.FormData && body && body !== null && typeof body === "object") {
      requestParams.headers.common = { Accept: "*/*" };
      requestParams.headers.post = {};
      requestParams.headers.put = {};

      body = this.createFormData(body as Record<string, unknown>);
    }

    return this.instance.request({
      ...requestParams,
      headers: {
        ...(type && type !== ContentType.FormData ? { "Content-Type": type } : {}),
        ...(requestParams.headers || {}),
      },
      params: query,
      responseType: responseFormat,
      data: body,
      url: path,
    });
  };
}

/**
 * @title cosmos/tx/v1beta1/service.proto
 * @version version not set
 */
export class Api<SecurityDataType extends unknown> extends HttpClient<SecurityDataType> {
  /**
   * @description Since: cosmos-sdk 0.47
   *
   * @tags Service
   * @name ServiceTxDecode
   * @summary TxDecode decodes the transaction.
   * @request POST:/cosmos/tx/v1beta1/decode
   */
  serviceTxDecode = (body: V1Beta1TxDecodeRequest, params: RequestParams = {}) =>
    this.request<V1Beta1TxDecodeResponse, RpcStatus>({
      path: `/cosmos/tx/v1beta1/decode`,
      method: "POST",
      body: body,
      type: ContentType.Json,
      format: "json",
      ...params,
    });

  /**
   * @description Since: cosmos-sdk 0.47
   *
   * @tags Service
   * @name ServiceTxDecodeAmino
   * @summary TxDecodeAmino decodes an Amino transaction from encoded bytes to JSON.
   * @request POST:/cosmos/tx/v1beta1/decode/amino
   */
  serviceTxDecodeAmino = (body: V1Beta1TxDecodeAminoRequest, params: RequestParams = {}) =>
    this.request<V1Beta1TxDecodeAminoResponse, RpcStatus>({
      path: `/cosmos/tx/v1beta1/decode/amino`,
      method: "POST",
      body: body,
      type: ContentType.Json,
      format: "json",
      ...params,
    });

  /**
   * @description Since: cosmos-sdk 0.47
   *
   * @tags Service
   * @name ServiceTxEncode
   * @summary TxEncode encodes the transaction.
   * @request POST:/cosmos/tx/v1beta1/encode
   */
  serviceTxEncode = (body: V1Beta1TxEncodeRequest, params: RequestParams = {}) =>
    this.request<V1Beta1TxEncodeResponse, RpcStatus>({
      path: `/cosmos/tx/v1beta1/encode`,
      method: "POST",
      body: body,
      type: ContentType.Json,
      format: "json",
      ...params,
    });

  /**
   * @description Since: cosmos-sdk 0.47
   *
   * @tags Service
   * @name ServiceTxEncodeAmino
   * @summary TxEncodeAmino encodes an Amino transaction from JSON to encoded bytes.
   * @request POST:/cosmos/tx/v1beta1/encode/amino
   */
  serviceTxEncodeAmino = (body: V1Beta1TxEncodeAminoRequest, params: RequestParams = {}) =>
    this.request<V1Beta1TxEncodeAminoResponse, RpcStatus>({
      path: `/cosmos/tx/v1beta1/encode/amino`,
      method: "POST",
      body: body,
      type: ContentType.Json,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Service
   * @name ServiceSimulate
   * @summary Simulate simulates executing a transaction for estimating gas usage.
   * @request POST:/cosmos/tx/v1beta1/simulate
   */
  serviceSimulate = (body: V1Beta1SimulateRequest, params: RequestParams = {}) =>
    this.request<V1Beta1SimulateResponse, RpcStatus>({
      path: `/cosmos/tx/v1beta1/simulate`,
      method: "POST",
      body: body,
      type: ContentType.Json,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Service
   * @name ServiceGetTxsEvent
   * @summary GetTxsEvent fetches txs by event.
   * @request GET:/cosmos/tx/v1beta1/txs
   */
  serviceGetTxsEvent = (
    query?: {
      events?: string[];
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
      order_by?: "ORDER_BY_UNSPECIFIED" | "ORDER_BY_ASC" | "ORDER_BY_DESC";
      page?: string;
      limit?: string;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1Beta1GetTxsEventResponse, RpcStatus>({
      path: `/cosmos/tx/v1beta1/txs`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Service
   * @name ServiceBroadcastTx
   * @summary BroadcastTx broadcast transaction.
   * @request POST:/cosmos/tx/v1beta1/txs
   */
  serviceBroadcastTx = (body: V1Beta1BroadcastTxRequest, params: RequestParams = {}) =>
    this.request<V1Beta1BroadcastTxResponse, RpcStatus>({
      path: `/cosmos/tx/v1beta1/txs`,
      method: "POST",
      body: body,
      type: ContentType.Json,
      format: "json",
      ...params,
    });

  /**
   * @description Since: cosmos-sdk 0.45.2
   *
   * @tags Service
   * @name ServiceGetBlockWithTxs
   * @summary GetBlockWithTxs fetches a block with decoded txs.
   * @request GET:/cosmos/tx/v1beta1/txs/block/{height}
   */
  serviceGetBlockWithTxs = (
    height: string,
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1Beta1GetBlockWithTxsResponse, RpcStatus>({
      path: `/cosmos/tx/v1beta1/txs/block/${height}`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Service
   * @name ServiceGetTx
   * @summary GetTx fetches a tx by hash.
   * @request GET:/cosmos/tx/v1beta1/txs/{hash}
   */
  serviceGetTx = (hash: string, params: RequestParams = {}) =>
    this.request<V1Beta1GetTxResponse, RpcStatus>({
      path: `/cosmos/tx/v1beta1/txs/${hash}`,
      method: "GET",
      format: "json",
      ...params,
    });
}
