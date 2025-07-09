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
* Channel defines pipeline for exactly-once packet delivery between specific
modules on separate blockchains, which has at least one end capable of
sending packets and one end capable of receiving packets.
*/
export interface Channelv1Channel {
  /**
   * current state of the channel end
   * State defines if a channel is in one of the following states:
   * CLOSED, INIT, TRYOPEN, OPEN or UNINITIALIZED.
   *
   *  - STATE_UNINITIALIZED_UNSPECIFIED: Default State
   *  - STATE_INIT: A channel has just started the opening handshake.
   *  - STATE_TRYOPEN: A channel has acknowledged the handshake step on the counterparty chain.
   *  - STATE_OPEN: A channel has completed the handshake. Open channels are
   * ready to send and receive packets.
   *  - STATE_CLOSED: A channel has been closed and can no longer be used to send or receive
   * packets.
   */
  state?: V1State;

  /**
   * whether the channel is ordered or unordered
   * - ORDER_NONE_UNSPECIFIED: zero-value for channel ordering
   *  - ORDER_UNORDERED: packets can be delivered in any order, which may differ from the order in
   * which they were sent.
   *  - ORDER_ORDERED: packets are delivered exactly in the order which they were sent
   */
  ordering?: V1Order;

  /** counterparty channel end */
  counterparty?: V1Counterparty;

  /**
   * list of connection identifiers, in order, along which packets sent on
   * this channel will travel
   */
  connection_hops?: string[];

  /** opaque channel version, which is agreed upon during the handshake */
  version?: string;
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

export interface V1Counterparty {
  /** port on the counterparty chain which owns the other end of the channel. */
  port_id?: string;

  /** channel end on the counterparty chain */
  channel_id?: string;
}

/**
* Normally the RevisionHeight is incremented at each height while keeping
RevisionNumber the same. However some consensus algorithms may choose to
reset the height in certain conditions e.g. hard forks, state-machine
breaking changes In these cases, the RevisionNumber is incremented so that
height continues to be monitonically increasing even as the RevisionHeight
gets reset
*/
export interface V1Height {
  /**
   * the revision that the client is currently on
   * @format uint64
   */
  revision_number?: string;

  /**
   * the height within the given revision
   * @format uint64
   */
  revision_height?: string;
}

/**
* IdentifiedChannel defines a channel with additional port and channel
identifier fields.
*/
export interface V1IdentifiedChannel {
  /**
   * current state of the channel end
   * State defines if a channel is in one of the following states:
   * CLOSED, INIT, TRYOPEN, OPEN or UNINITIALIZED.
   *
   *  - STATE_UNINITIALIZED_UNSPECIFIED: Default State
   *  - STATE_INIT: A channel has just started the opening handshake.
   *  - STATE_TRYOPEN: A channel has acknowledged the handshake step on the counterparty chain.
   *  - STATE_OPEN: A channel has completed the handshake. Open channels are
   * ready to send and receive packets.
   *  - STATE_CLOSED: A channel has been closed and can no longer be used to send or receive
   * packets.
   */
  state?: V1State;

  /**
   * whether the channel is ordered or unordered
   * - ORDER_NONE_UNSPECIFIED: zero-value for channel ordering
   *  - ORDER_UNORDERED: packets can be delivered in any order, which may differ from the order in
   * which they were sent.
   *  - ORDER_ORDERED: packets are delivered exactly in the order which they were sent
   */
  ordering?: V1Order;

  /** counterparty channel end */
  counterparty?: V1Counterparty;

  /**
   * list of connection identifiers, in order, along which packets sent on
   * this channel will travel
   */
  connection_hops?: string[];

  /** opaque channel version, which is agreed upon during the handshake */
  version?: string;

  /** port identifier */
  port_id?: string;

  /** channel identifier */
  channel_id?: string;
}

/**
* IdentifiedClientState defines a client state with an additional client
identifier field.
*/
export interface V1IdentifiedClientState {
  /** client identifier */
  client_id?: string;

  /**
   * client state
   * `Any` contains an arbitrary serialized protocol buffer message along with a
   * URL that describes the type of the serialized message.
   *
   * Protobuf library provides support to pack/unpack Any values in the form
   * of utility functions or additional generated methods of the Any type.
   * Example 1: Pack and unpack a message in C++.
   *     Foo foo = ...;
   *     Any any;
   *     any.PackFrom(foo);
   *     ...
   *     if (any.UnpackTo(&foo)) {
   *       ...
   *     }
   * Example 2: Pack and unpack a message in Java.
   *     Any any = Any.pack(foo);
   *     if (any.is(Foo.class)) {
   *       foo = any.unpack(Foo.class);
   *  Example 3: Pack and unpack a message in Python.
   *     foo = Foo(...)
   *     any = Any()
   *     any.Pack(foo)
   *     if any.Is(Foo.DESCRIPTOR):
   *       any.Unpack(foo)
   *  Example 4: Pack and unpack a message in Go
   *      foo := &pb.Foo{...}
   *      any, err := anypb.New(foo)
   *      if err != nil {
   *        ...
   *      }
   *      ...
   *      foo := &pb.Foo{}
   *      if err := any.UnmarshalTo(foo); err != nil {
   * The pack methods provided by protobuf library will by default use
   * 'type.googleapis.com/full.type.name' as the type URL and the unpack
   * methods only use the fully qualified type name after the last '/'
   * in the type URL, for example "foo.bar.com/x/y.z" will yield type
   * name "y.z".
   * JSON
   * ====
   * The JSON representation of an `Any` value uses the regular
   * representation of the deserialized, embedded message, with an
   * additional field `@type` which contains the type URL. Example:
   *     package google.profile;
   *     message Person {
   *       string first_name = 1;
   *       string last_name = 2;
   *     {
   *       "@type": "type.googleapis.com/google.profile.Person",
   *       "firstName": <string>,
   *       "lastName": <string>
   * If the embedded message type is well-known and has a custom JSON
   * representation, that representation will be embedded adding a field
   * `value` which holds the custom JSON in addition to the `@type`
   * field. Example (for message [google.protobuf.Duration][]):
   *       "@type": "type.googleapis.com/google.protobuf.Duration",
   *       "value": "1.212s"
   */
  client_state?: ProtobufAny;
}

/**
 * MsgAcknowledgementResponse defines the Msg/Acknowledgement response type.
 */
export interface V1MsgAcknowledgementResponse {
  /**
   * - RESPONSE_RESULT_TYPE_UNSPECIFIED: Default zero value enumeration
   *  - RESPONSE_RESULT_TYPE_NOOP: The message did not call the IBC application callbacks (because, for example, the packet had already been relayed)
   *  - RESPONSE_RESULT_TYPE_SUCCESS: The message was executed successfully
   */
  result?: V1ResponseResultType;
}

/**
* MsgChannelCloseConfirmResponse defines the Msg/ChannelCloseConfirm response
type.
*/
export type V1MsgChannelCloseConfirmResponse = object;

/**
 * MsgChannelCloseInitResponse defines the Msg/ChannelCloseInit response type.
 */
export type V1MsgChannelCloseInitResponse = object;

/**
 * MsgChannelOpenAckResponse defines the Msg/ChannelOpenAck response type.
 */
export type V1MsgChannelOpenAckResponse = object;

/**
* MsgChannelOpenConfirmResponse defines the Msg/ChannelOpenConfirm response
type.
*/
export type V1MsgChannelOpenConfirmResponse = object;

/**
 * MsgChannelOpenInitResponse defines the Msg/ChannelOpenInit response type.
 */
export interface V1MsgChannelOpenInitResponse {
  channel_id?: string;
  version?: string;
}

/**
 * MsgChannelOpenTryResponse defines the Msg/ChannelOpenTry response type.
 */
export interface V1MsgChannelOpenTryResponse {
  version?: string;
  channel_id?: string;
}

/**
 * MsgRecvPacketResponse defines the Msg/RecvPacket response type.
 */
export interface V1MsgRecvPacketResponse {
  /**
   * - RESPONSE_RESULT_TYPE_UNSPECIFIED: Default zero value enumeration
   *  - RESPONSE_RESULT_TYPE_NOOP: The message did not call the IBC application callbacks (because, for example, the packet had already been relayed)
   *  - RESPONSE_RESULT_TYPE_SUCCESS: The message was executed successfully
   */
  result?: V1ResponseResultType;
}

/**
 * MsgTimeoutOnCloseResponse defines the Msg/TimeoutOnClose response type.
 */
export interface V1MsgTimeoutOnCloseResponse {
  /**
   * - RESPONSE_RESULT_TYPE_UNSPECIFIED: Default zero value enumeration
   *  - RESPONSE_RESULT_TYPE_NOOP: The message did not call the IBC application callbacks (because, for example, the packet had already been relayed)
   *  - RESPONSE_RESULT_TYPE_SUCCESS: The message was executed successfully
   */
  result?: V1ResponseResultType;
}

/**
 * MsgTimeoutResponse defines the Msg/Timeout response type.
 */
export interface V1MsgTimeoutResponse {
  /**
   * - RESPONSE_RESULT_TYPE_UNSPECIFIED: Default zero value enumeration
   *  - RESPONSE_RESULT_TYPE_NOOP: The message did not call the IBC application callbacks (because, for example, the packet had already been relayed)
   *  - RESPONSE_RESULT_TYPE_SUCCESS: The message was executed successfully
   */
  result?: V1ResponseResultType;
}

/**
* - ORDER_NONE_UNSPECIFIED: zero-value for channel ordering
 - ORDER_UNORDERED: packets can be delivered in any order, which may differ from the order in
which they were sent.
 - ORDER_ORDERED: packets are delivered exactly in the order which they were sent
*/
export enum V1Order {
  ORDER_NONE_UNSPECIFIED = "ORDER_NONE_UNSPECIFIED",
  ORDER_UNORDERED = "ORDER_UNORDERED",
  ORDER_ORDERED = "ORDER_ORDERED",
}

export interface V1Packet {
  /**
   * number corresponds to the order of sends and receives, where a Packet
   * with an earlier sequence number must be sent and received before a Packet
   * with a later sequence number.
   * @format uint64
   */
  sequence?: string;

  /** identifies the port on the sending chain. */
  source_port?: string;

  /** identifies the channel end on the sending chain. */
  source_channel?: string;

  /** identifies the port on the receiving chain. */
  destination_port?: string;

  /** identifies the channel end on the receiving chain. */
  destination_channel?: string;

  /**
   * actual opaque bytes transferred directly to the application module
   * @format byte
   */
  data?: string;

  /**
   * block height after which the packet times out
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  timeout_height?: V1Height;

  /**
   * block timestamp (in nanoseconds) after which the packet times out
   * @format uint64
   */
  timeout_timestamp?: string;
}

/**
* PacketState defines the generic type necessary to retrieve and store
packet commitments, acknowledgements, and receipts.
Caller is responsible for knowing the context necessary to interpret this
state as a commitment, acknowledgement, or a receipt.
*/
export interface V1PacketState {
  /** channel port identifier. */
  port_id?: string;

  /** channel unique identifier. */
  channel_id?: string;

  /**
   * packet sequence.
   * @format uint64
   */
  sequence?: string;

  /**
   * embedded data that represents packet state.
   * @format byte
   */
  data?: string;
}

export interface V1QueryChannelClientStateResponse {
  /**
   * client state associated with the channel
   * IdentifiedClientState defines a client state with an additional client
   * identifier field.
   */
  identified_client_state?: V1IdentifiedClientState;

  /**
   * merkle proof of existence
   * @format byte
   */
  proof?: string;

  /**
   * height at which the proof was retrieved
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  proof_height?: V1Height;
}

export interface V1QueryChannelConsensusStateResponse {
  /**
   * consensus state associated with the channel
   * `Any` contains an arbitrary serialized protocol buffer message along with a
   * URL that describes the type of the serialized message.
   *
   * Protobuf library provides support to pack/unpack Any values in the form
   * of utility functions or additional generated methods of the Any type.
   * Example 1: Pack and unpack a message in C++.
   *     Foo foo = ...;
   *     Any any;
   *     any.PackFrom(foo);
   *     ...
   *     if (any.UnpackTo(&foo)) {
   *       ...
   *     }
   * Example 2: Pack and unpack a message in Java.
   *     Any any = Any.pack(foo);
   *     if (any.is(Foo.class)) {
   *       foo = any.unpack(Foo.class);
   *  Example 3: Pack and unpack a message in Python.
   *     foo = Foo(...)
   *     any = Any()
   *     any.Pack(foo)
   *     if any.Is(Foo.DESCRIPTOR):
   *       any.Unpack(foo)
   *  Example 4: Pack and unpack a message in Go
   *      foo := &pb.Foo{...}
   *      any, err := anypb.New(foo)
   *      if err != nil {
   *        ...
   *      }
   *      ...
   *      foo := &pb.Foo{}
   *      if err := any.UnmarshalTo(foo); err != nil {
   * The pack methods provided by protobuf library will by default use
   * 'type.googleapis.com/full.type.name' as the type URL and the unpack
   * methods only use the fully qualified type name after the last '/'
   * in the type URL, for example "foo.bar.com/x/y.z" will yield type
   * name "y.z".
   * JSON
   * ====
   * The JSON representation of an `Any` value uses the regular
   * representation of the deserialized, embedded message, with an
   * additional field `@type` which contains the type URL. Example:
   *     package google.profile;
   *     message Person {
   *       string first_name = 1;
   *       string last_name = 2;
   *     {
   *       "@type": "type.googleapis.com/google.profile.Person",
   *       "firstName": <string>,
   *       "lastName": <string>
   * If the embedded message type is well-known and has a custom JSON
   * representation, that representation will be embedded adding a field
   * `value` which holds the custom JSON in addition to the `@type`
   * field. Example (for message [google.protobuf.Duration][]):
   *       "@type": "type.googleapis.com/google.protobuf.Duration",
   *       "value": "1.212s"
   */
  consensus_state?: ProtobufAny;

  /** client ID associated with the consensus state */
  client_id?: string;

  /**
   * merkle proof of existence
   * @format byte
   */
  proof?: string;

  /**
   * height at which the proof was retrieved
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  proof_height?: V1Height;
}

/**
* QueryChannelResponse is the response type for the Query/Channel RPC method.
Besides the Channel end, it includes a proof and the height from which the
proof was retrieved.
*/
export interface V1QueryChannelResponse {
  /**
   * channel associated with the request identifiers
   * Channel defines pipeline for exactly-once packet delivery between specific
   * modules on separate blockchains, which has at least one end capable of
   * sending packets and one end capable of receiving packets.
   */
  channel?: Channelv1Channel;

  /**
   * merkle proof of existence
   * @format byte
   */
  proof?: string;

  /**
   * height at which the proof was retrieved
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  proof_height?: V1Height;
}

/**
 * QueryChannelsResponse is the response type for the Query/Channels RPC method.
 */
export interface V1QueryChannelsResponse {
  /** list of stored channels of the chain. */
  channels?: V1IdentifiedChannel[];

  /**
   * pagination response
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;

  /**
   * query block height
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  height?: V1Height;
}

export interface V1QueryConnectionChannelsResponse {
  /** list of channels associated with a connection. */
  channels?: V1IdentifiedChannel[];

  /**
   * pagination response
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;

  /**
   * query block height
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  height?: V1Height;
}

export interface V1QueryNextSequenceReceiveResponse {
  /**
   * next sequence receive number
   * @format uint64
   */
  next_sequence_receive?: string;

  /**
   * merkle proof of existence
   * @format byte
   */
  proof?: string;

  /**
   * height at which the proof was retrieved
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  proof_height?: V1Height;
}

export interface V1QueryPacketAcknowledgementResponse {
  /**
   * packet associated with the request fields
   * @format byte
   */
  acknowledgement?: string;

  /**
   * merkle proof of existence
   * @format byte
   */
  proof?: string;

  /**
   * height at which the proof was retrieved
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  proof_height?: V1Height;
}

export interface V1QueryPacketAcknowledgementsResponse {
  acknowledgements?: V1PacketState[];

  /**
   * pagination response
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;

  /**
   * query block height
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  height?: V1Height;
}

export interface V1QueryPacketCommitmentResponse {
  /**
   * packet associated with the request fields
   * @format byte
   */
  commitment?: string;

  /**
   * merkle proof of existence
   * @format byte
   */
  proof?: string;

  /**
   * height at which the proof was retrieved
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  proof_height?: V1Height;
}

export interface V1QueryPacketCommitmentsResponse {
  commitments?: V1PacketState[];

  /**
   * pagination response
   * PageResponse is to be embedded in gRPC response messages where the
   * corresponding request message has used PageRequest.
   *
   *  message SomeResponse {
   *          repeated Bar results = 1;
   *          PageResponse page = 2;
   *  }
   */
  pagination?: V1Beta1PageResponse;

  /**
   * query block height
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  height?: V1Height;
}

export interface V1QueryPacketReceiptResponse {
  /** success flag for if receipt exists */
  received?: boolean;

  /**
   * merkle proof of existence
   * @format byte
   */
  proof?: string;

  /**
   * height at which the proof was retrieved
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  proof_height?: V1Height;
}

export interface V1QueryUnreceivedAcksResponse {
  /** list of unreceived acknowledgement sequences */
  sequences?: string[];

  /**
   * query block height
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  height?: V1Height;
}

export interface V1QueryUnreceivedPacketsResponse {
  /** list of unreceived packet sequences */
  sequences?: string[];

  /**
   * query block height
   * Normally the RevisionHeight is incremented at each height while keeping
   * RevisionNumber the same. However some consensus algorithms may choose to
   * reset the height in certain conditions e.g. hard forks, state-machine
   * breaking changes In these cases, the RevisionNumber is incremented so that
   * height continues to be monitonically increasing even as the RevisionHeight
   * gets reset
   */
  height?: V1Height;
}

/**
* - RESPONSE_RESULT_TYPE_UNSPECIFIED: Default zero value enumeration
 - RESPONSE_RESULT_TYPE_NOOP: The message did not call the IBC application callbacks (because, for example, the packet had already been relayed)
 - RESPONSE_RESULT_TYPE_SUCCESS: The message was executed successfully
*/
export enum V1ResponseResultType {
  RESPONSE_RESULT_TYPE_UNSPECIFIED = "RESPONSE_RESULT_TYPE_UNSPECIFIED",
  RESPONSE_RESULT_TYPE_NOOP = "RESPONSE_RESULT_TYPE_NOOP",
  RESPONSE_RESULT_TYPE_SUCCESS = "RESPONSE_RESULT_TYPE_SUCCESS",
}

/**
* State defines if a channel is in one of the following states:
CLOSED, INIT, TRYOPEN, OPEN or UNINITIALIZED.

 - STATE_UNINITIALIZED_UNSPECIFIED: Default State
 - STATE_INIT: A channel has just started the opening handshake.
 - STATE_TRYOPEN: A channel has acknowledged the handshake step on the counterparty chain.
 - STATE_OPEN: A channel has completed the handshake. Open channels are
ready to send and receive packets.
 - STATE_CLOSED: A channel has been closed and can no longer be used to send or receive
packets.
*/
export enum V1State {
  STATE_UNINITIALIZED_UNSPECIFIED = "STATE_UNINITIALIZED_UNSPECIFIED",
  STATE_INIT = "STATE_INIT",
  STATE_TRYOPEN = "STATE_TRYOPEN",
  STATE_OPEN = "STATE_OPEN",
  STATE_CLOSED = "STATE_CLOSED",
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
 * @title ibc/core/channel/v1/channel.proto
 * @version version not set
 */
export class Api<SecurityDataType extends unknown> extends HttpClient<SecurityDataType> {
  /**
   * No description
   *
   * @tags Query
   * @name QueryChannels
   * @summary Channels queries all the IBC channels of a chain.
   * @request GET:/ibc/core/channel/v1/channels
   */
  queryChannels = (
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1QueryChannelsResponse, RpcStatus>({
      path: `/ibc/core/channel/v1/channels`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryChannel
   * @summary Channel queries an IBC Channel.
   * @request GET:/ibc/core/channel/v1/channels/{channel_id}/ports/{port_id}
   */
  queryChannel = (channelId: string, portId: string, params: RequestParams = {}) =>
    this.request<V1QueryChannelResponse, RpcStatus>({
      path: `/ibc/core/channel/v1/channels/${channelId}/ports/${portId}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
 * No description
 * 
 * @tags Query
 * @name QueryChannelClientState
 * @summary ChannelClientState queries for the client state for the channel associated
with the provided channel identifiers.
 * @request GET:/ibc/core/channel/v1/channels/{channel_id}/ports/{port_id}/client_state
 */
  queryChannelClientState = (channelId: string, portId: string, params: RequestParams = {}) =>
    this.request<V1QueryChannelClientStateResponse, RpcStatus>({
      path: `/ibc/core/channel/v1/channels/${channelId}/ports/${portId}/client_state`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
 * No description
 * 
 * @tags Query
 * @name QueryChannelConsensusState
 * @summary ChannelConsensusState queries for the consensus state for the channel
associated with the provided channel identifiers.
 * @request GET:/ibc/core/channel/v1/channels/{channel_id}/ports/{port_id}/consensus_state/revision/{revision_number}/height/{revision_height}
 */
  queryChannelConsensusState = (
    channelId: string,
    portId: string,
    revisionNumber: string,
    revisionHeight: string,
    params: RequestParams = {},
  ) =>
    this.request<V1QueryChannelConsensusStateResponse, RpcStatus>({
      path: `/ibc/core/channel/v1/channels/${channelId}/ports/${portId}/consensus_state/revision/${revisionNumber}/height/${revisionHeight}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryNextSequenceReceive
   * @summary NextSequenceReceive returns the next receive sequence for a given channel.
   * @request GET:/ibc/core/channel/v1/channels/{channel_id}/ports/{port_id}/next_sequence
   */
  queryNextSequenceReceive = (channelId: string, portId: string, params: RequestParams = {}) =>
    this.request<V1QueryNextSequenceReceiveResponse, RpcStatus>({
      path: `/ibc/core/channel/v1/channels/${channelId}/ports/${portId}/next_sequence`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
 * No description
 * 
 * @tags Query
 * @name QueryPacketAcknowledgements
 * @summary PacketAcknowledgements returns all the packet acknowledgements associated
with a channel.
 * @request GET:/ibc/core/channel/v1/channels/{channel_id}/ports/{port_id}/packet_acknowledgements
 */
  queryPacketAcknowledgements = (
    channelId: string,
    portId: string,
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
      packet_commitment_sequences?: string[];
    },
    params: RequestParams = {},
  ) =>
    this.request<V1QueryPacketAcknowledgementsResponse, RpcStatus>({
      path: `/ibc/core/channel/v1/channels/${channelId}/ports/${portId}/packet_acknowledgements`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryPacketAcknowledgement
   * @summary PacketAcknowledgement queries a stored packet acknowledgement hash.
   * @request GET:/ibc/core/channel/v1/channels/{channel_id}/ports/{port_id}/packet_acks/{sequence}
   */
  queryPacketAcknowledgement = (channelId: string, portId: string, sequence: string, params: RequestParams = {}) =>
    this.request<V1QueryPacketAcknowledgementResponse, RpcStatus>({
      path: `/ibc/core/channel/v1/channels/${channelId}/ports/${portId}/packet_acks/${sequence}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
 * No description
 * 
 * @tags Query
 * @name QueryPacketCommitments
 * @summary PacketCommitments returns all the packet commitments hashes associated
with a channel.
 * @request GET:/ibc/core/channel/v1/channels/{channel_id}/ports/{port_id}/packet_commitments
 */
  queryPacketCommitments = (
    channelId: string,
    portId: string,
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1QueryPacketCommitmentsResponse, RpcStatus>({
      path: `/ibc/core/channel/v1/channels/${channelId}/ports/${portId}/packet_commitments`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });

  /**
 * No description
 * 
 * @tags Query
 * @name QueryUnreceivedAcks
 * @summary UnreceivedAcks returns all the unreceived IBC acknowledgements associated
with a channel and sequences.
 * @request GET:/ibc/core/channel/v1/channels/{channel_id}/ports/{port_id}/packet_commitments/{packet_ack_sequences}/unreceived_acks
 */
  queryUnreceivedAcks = (channelId: string, portId: string, packetAckSequences: string[], params: RequestParams = {}) =>
    this.request<V1QueryUnreceivedAcksResponse, RpcStatus>({
      path: `/ibc/core/channel/v1/channels/${channelId}/ports/${portId}/packet_commitments/${packetAckSequences}/unreceived_acks`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
 * No description
 * 
 * @tags Query
 * @name QueryUnreceivedPackets
 * @summary UnreceivedPackets returns all the unreceived IBC packets associated with a
channel and sequences.
 * @request GET:/ibc/core/channel/v1/channels/{channel_id}/ports/{port_id}/packet_commitments/{packet_commitment_sequences}/unreceived_packets
 */
  queryUnreceivedPackets = (
    channelId: string,
    portId: string,
    packetCommitmentSequences: string[],
    params: RequestParams = {},
  ) =>
    this.request<V1QueryUnreceivedPacketsResponse, RpcStatus>({
      path: `/ibc/core/channel/v1/channels/${channelId}/ports/${portId}/packet_commitments/${packetCommitmentSequences}/unreceived_packets`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
   * No description
   *
   * @tags Query
   * @name QueryPacketCommitment
   * @summary PacketCommitment queries a stored packet commitment hash.
   * @request GET:/ibc/core/channel/v1/channels/{channel_id}/ports/{port_id}/packet_commitments/{sequence}
   */
  queryPacketCommitment = (channelId: string, portId: string, sequence: string, params: RequestParams = {}) =>
    this.request<V1QueryPacketCommitmentResponse, RpcStatus>({
      path: `/ibc/core/channel/v1/channels/${channelId}/ports/${portId}/packet_commitments/${sequence}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
 * No description
 * 
 * @tags Query
 * @name QueryPacketReceipt
 * @summary PacketReceipt queries if a given packet sequence has been received on the
queried chain
 * @request GET:/ibc/core/channel/v1/channels/{channel_id}/ports/{port_id}/packet_receipts/{sequence}
 */
  queryPacketReceipt = (channelId: string, portId: string, sequence: string, params: RequestParams = {}) =>
    this.request<V1QueryPacketReceiptResponse, RpcStatus>({
      path: `/ibc/core/channel/v1/channels/${channelId}/ports/${portId}/packet_receipts/${sequence}`,
      method: "GET",
      format: "json",
      ...params,
    });

  /**
 * No description
 * 
 * @tags Query
 * @name QueryConnectionChannels
 * @summary ConnectionChannels queries all the channels associated with a connection
end.
 * @request GET:/ibc/core/channel/v1/connections/{connection}/channels
 */
  queryConnectionChannels = (
    connection: string,
    query?: {
      "pagination.key"?: string;
      "pagination.offset"?: string;
      "pagination.limit"?: string;
      "pagination.count_total"?: boolean;
      "pagination.reverse"?: boolean;
    },
    params: RequestParams = {},
  ) =>
    this.request<V1QueryConnectionChannelsResponse, RpcStatus>({
      path: `/ibc/core/channel/v1/connections/${connection}/channels`,
      method: "GET",
      query: query,
      format: "json",
      ...params,
    });
}
