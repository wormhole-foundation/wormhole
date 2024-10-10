//@ts-nocheck
/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Height } from "../../client/v1/client";

export const protobufPackage = "ibc.core.channel.v1";

/**
 * State defines if a channel is in one of the following states:
 * CLOSED, INIT, TRYOPEN, OPEN or UNINITIALIZED.
 */
export enum State {
  /** STATE_UNINITIALIZED_UNSPECIFIED - Default State */
  STATE_UNINITIALIZED_UNSPECIFIED = 0,
  /** STATE_INIT - A channel has just started the opening handshake. */
  STATE_INIT = 1,
  /** STATE_TRYOPEN - A channel has acknowledged the handshake step on the counterparty chain. */
  STATE_TRYOPEN = 2,
  /**
   * STATE_OPEN - A channel has completed the handshake. Open channels are
   * ready to send and receive packets.
   */
  STATE_OPEN = 3,
  /**
   * STATE_CLOSED - A channel has been closed and can no longer be used to send or receive
   * packets.
   */
  STATE_CLOSED = 4,
  UNRECOGNIZED = -1,
}

export function stateFromJSON(object: any): State {
  switch (object) {
    case 0:
    case "STATE_UNINITIALIZED_UNSPECIFIED":
      return State.STATE_UNINITIALIZED_UNSPECIFIED;
    case 1:
    case "STATE_INIT":
      return State.STATE_INIT;
    case 2:
    case "STATE_TRYOPEN":
      return State.STATE_TRYOPEN;
    case 3:
    case "STATE_OPEN":
      return State.STATE_OPEN;
    case 4:
    case "STATE_CLOSED":
      return State.STATE_CLOSED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return State.UNRECOGNIZED;
  }
}

export function stateToJSON(object: State): string {
  switch (object) {
    case State.STATE_UNINITIALIZED_UNSPECIFIED:
      return "STATE_UNINITIALIZED_UNSPECIFIED";
    case State.STATE_INIT:
      return "STATE_INIT";
    case State.STATE_TRYOPEN:
      return "STATE_TRYOPEN";
    case State.STATE_OPEN:
      return "STATE_OPEN";
    case State.STATE_CLOSED:
      return "STATE_CLOSED";
    case State.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** Order defines if a channel is ORDERED or UNORDERED */
export enum Order {
  /** ORDER_NONE_UNSPECIFIED - zero-value for channel ordering */
  ORDER_NONE_UNSPECIFIED = 0,
  /**
   * ORDER_UNORDERED - packets can be delivered in any order, which may differ from the order in
   * which they were sent.
   */
  ORDER_UNORDERED = 1,
  /** ORDER_ORDERED - packets are delivered exactly in the order which they were sent */
  ORDER_ORDERED = 2,
  UNRECOGNIZED = -1,
}

export function orderFromJSON(object: any): Order {
  switch (object) {
    case 0:
    case "ORDER_NONE_UNSPECIFIED":
      return Order.ORDER_NONE_UNSPECIFIED;
    case 1:
    case "ORDER_UNORDERED":
      return Order.ORDER_UNORDERED;
    case 2:
    case "ORDER_ORDERED":
      return Order.ORDER_ORDERED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Order.UNRECOGNIZED;
  }
}

export function orderToJSON(object: Order): string {
  switch (object) {
    case Order.ORDER_NONE_UNSPECIFIED:
      return "ORDER_NONE_UNSPECIFIED";
    case Order.ORDER_UNORDERED:
      return "ORDER_UNORDERED";
    case Order.ORDER_ORDERED:
      return "ORDER_ORDERED";
    case Order.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/**
 * Channel defines pipeline for exactly-once packet delivery between specific
 * modules on separate blockchains, which has at least one end capable of
 * sending packets and one end capable of receiving packets.
 */
export interface Channel {
  /** current state of the channel end */
  state: State;
  /** whether the channel is ordered or unordered */
  ordering: Order;
  /** counterparty channel end */
  counterparty:
    | Counterparty
    | undefined;
  /**
   * list of connection identifiers, in order, along which packets sent on
   * this channel will travel
   */
  connectionHops: string[];
  /** opaque channel version, which is agreed upon during the handshake */
  version: string;
}

/**
 * IdentifiedChannel defines a channel with additional port and channel
 * identifier fields.
 */
export interface IdentifiedChannel {
  /** current state of the channel end */
  state: State;
  /** whether the channel is ordered or unordered */
  ordering: Order;
  /** counterparty channel end */
  counterparty:
    | Counterparty
    | undefined;
  /**
   * list of connection identifiers, in order, along which packets sent on
   * this channel will travel
   */
  connectionHops: string[];
  /** opaque channel version, which is agreed upon during the handshake */
  version: string;
  /** port identifier */
  portId: string;
  /** channel identifier */
  channelId: string;
}

/** Counterparty defines a channel end counterparty */
export interface Counterparty {
  /** port on the counterparty chain which owns the other end of the channel. */
  portId: string;
  /** channel end on the counterparty chain */
  channelId: string;
}

/** Packet defines a type that carries data across different chains through IBC */
export interface Packet {
  /**
   * number corresponds to the order of sends and receives, where a Packet
   * with an earlier sequence number must be sent and received before a Packet
   * with a later sequence number.
   */
  sequence: number;
  /** identifies the port on the sending chain. */
  sourcePort: string;
  /** identifies the channel end on the sending chain. */
  sourceChannel: string;
  /** identifies the port on the receiving chain. */
  destinationPort: string;
  /** identifies the channel end on the receiving chain. */
  destinationChannel: string;
  /** actual opaque bytes transferred directly to the application module */
  data: Uint8Array;
  /** block height after which the packet times out */
  timeoutHeight:
    | Height
    | undefined;
  /** block timestamp (in nanoseconds) after which the packet times out */
  timeoutTimestamp: number;
}

/**
 * PacketState defines the generic type necessary to retrieve and store
 * packet commitments, acknowledgements, and receipts.
 * Caller is responsible for knowing the context necessary to interpret this
 * state as a commitment, acknowledgement, or a receipt.
 */
export interface PacketState {
  /** channel port identifier. */
  portId: string;
  /** channel unique identifier. */
  channelId: string;
  /** packet sequence. */
  sequence: number;
  /** embedded data that represents packet state. */
  data: Uint8Array;
}

/**
 * PacketId is an identifer for a unique Packet
 * Source chains refer to packets by source port/channel
 * Destination chains refer to packets by destination port/channel
 */
export interface PacketId {
  /** channel port identifier */
  portId: string;
  /** channel unique identifier */
  channelId: string;
  /** packet sequence */
  sequence: number;
}

/**
 * Acknowledgement is the recommended acknowledgement format to be used by
 * app-specific protocols.
 * NOTE: The field numbers 21 and 22 were explicitly chosen to avoid accidental
 * conflicts with other protobuf message formats used for acknowledgements.
 * The first byte of any message with this format will be the non-ASCII values
 * `0xaa` (result) or `0xb2` (error). Implemented as defined by ICS:
 * https://github.com/cosmos/ibc/tree/master/spec/core/ics-004-channel-and-packet-semantics#acknowledgement-envelope
 */
export interface Acknowledgement {
  result: Uint8Array | undefined;
  error: string | undefined;
}

function createBaseChannel(): Channel {
  return { state: 0, ordering: 0, counterparty: undefined, connectionHops: [], version: "" };
}

export const Channel = {
  encode(message: Channel, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.state !== 0) {
      writer.uint32(8).int32(message.state);
    }
    if (message.ordering !== 0) {
      writer.uint32(16).int32(message.ordering);
    }
    if (message.counterparty !== undefined) {
      Counterparty.encode(message.counterparty, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.connectionHops) {
      writer.uint32(34).string(v!);
    }
    if (message.version !== "") {
      writer.uint32(42).string(message.version);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Channel {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChannel();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.state = reader.int32() as any;
          break;
        case 2:
          message.ordering = reader.int32() as any;
          break;
        case 3:
          message.counterparty = Counterparty.decode(reader, reader.uint32());
          break;
        case 4:
          message.connectionHops.push(reader.string());
          break;
        case 5:
          message.version = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Channel {
    return {
      state: isSet(object.state) ? stateFromJSON(object.state) : 0,
      ordering: isSet(object.ordering) ? orderFromJSON(object.ordering) : 0,
      counterparty: isSet(object.counterparty) ? Counterparty.fromJSON(object.counterparty) : undefined,
      connectionHops: Array.isArray(object?.connectionHops) ? object.connectionHops.map((e: any) => String(e)) : [],
      version: isSet(object.version) ? String(object.version) : "",
    };
  },

  toJSON(message: Channel): unknown {
    const obj: any = {};
    message.state !== undefined && (obj.state = stateToJSON(message.state));
    message.ordering !== undefined && (obj.ordering = orderToJSON(message.ordering));
    message.counterparty !== undefined
      && (obj.counterparty = message.counterparty ? Counterparty.toJSON(message.counterparty) : undefined);
    if (message.connectionHops) {
      obj.connectionHops = message.connectionHops.map((e) => e);
    } else {
      obj.connectionHops = [];
    }
    message.version !== undefined && (obj.version = message.version);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Channel>, I>>(object: I): Channel {
    const message = createBaseChannel();
    message.state = object.state ?? 0;
    message.ordering = object.ordering ?? 0;
    message.counterparty = (object.counterparty !== undefined && object.counterparty !== null)
      ? Counterparty.fromPartial(object.counterparty)
      : undefined;
    message.connectionHops = object.connectionHops?.map((e) => e) || [];
    message.version = object.version ?? "";
    return message;
  },
};

function createBaseIdentifiedChannel(): IdentifiedChannel {
  return { state: 0, ordering: 0, counterparty: undefined, connectionHops: [], version: "", portId: "", channelId: "" };
}

export const IdentifiedChannel = {
  encode(message: IdentifiedChannel, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.state !== 0) {
      writer.uint32(8).int32(message.state);
    }
    if (message.ordering !== 0) {
      writer.uint32(16).int32(message.ordering);
    }
    if (message.counterparty !== undefined) {
      Counterparty.encode(message.counterparty, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.connectionHops) {
      writer.uint32(34).string(v!);
    }
    if (message.version !== "") {
      writer.uint32(42).string(message.version);
    }
    if (message.portId !== "") {
      writer.uint32(50).string(message.portId);
    }
    if (message.channelId !== "") {
      writer.uint32(58).string(message.channelId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IdentifiedChannel {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIdentifiedChannel();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.state = reader.int32() as any;
          break;
        case 2:
          message.ordering = reader.int32() as any;
          break;
        case 3:
          message.counterparty = Counterparty.decode(reader, reader.uint32());
          break;
        case 4:
          message.connectionHops.push(reader.string());
          break;
        case 5:
          message.version = reader.string();
          break;
        case 6:
          message.portId = reader.string();
          break;
        case 7:
          message.channelId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): IdentifiedChannel {
    return {
      state: isSet(object.state) ? stateFromJSON(object.state) : 0,
      ordering: isSet(object.ordering) ? orderFromJSON(object.ordering) : 0,
      counterparty: isSet(object.counterparty) ? Counterparty.fromJSON(object.counterparty) : undefined,
      connectionHops: Array.isArray(object?.connectionHops) ? object.connectionHops.map((e: any) => String(e)) : [],
      version: isSet(object.version) ? String(object.version) : "",
      portId: isSet(object.portId) ? String(object.portId) : "",
      channelId: isSet(object.channelId) ? String(object.channelId) : "",
    };
  },

  toJSON(message: IdentifiedChannel): unknown {
    const obj: any = {};
    message.state !== undefined && (obj.state = stateToJSON(message.state));
    message.ordering !== undefined && (obj.ordering = orderToJSON(message.ordering));
    message.counterparty !== undefined
      && (obj.counterparty = message.counterparty ? Counterparty.toJSON(message.counterparty) : undefined);
    if (message.connectionHops) {
      obj.connectionHops = message.connectionHops.map((e) => e);
    } else {
      obj.connectionHops = [];
    }
    message.version !== undefined && (obj.version = message.version);
    message.portId !== undefined && (obj.portId = message.portId);
    message.channelId !== undefined && (obj.channelId = message.channelId);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<IdentifiedChannel>, I>>(object: I): IdentifiedChannel {
    const message = createBaseIdentifiedChannel();
    message.state = object.state ?? 0;
    message.ordering = object.ordering ?? 0;
    message.counterparty = (object.counterparty !== undefined && object.counterparty !== null)
      ? Counterparty.fromPartial(object.counterparty)
      : undefined;
    message.connectionHops = object.connectionHops?.map((e) => e) || [];
    message.version = object.version ?? "";
    message.portId = object.portId ?? "";
    message.channelId = object.channelId ?? "";
    return message;
  },
};

function createBaseCounterparty(): Counterparty {
  return { portId: "", channelId: "" };
}

export const Counterparty = {
  encode(message: Counterparty, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.portId !== "") {
      writer.uint32(10).string(message.portId);
    }
    if (message.channelId !== "") {
      writer.uint32(18).string(message.channelId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Counterparty {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCounterparty();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.portId = reader.string();
          break;
        case 2:
          message.channelId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Counterparty {
    return {
      portId: isSet(object.portId) ? String(object.portId) : "",
      channelId: isSet(object.channelId) ? String(object.channelId) : "",
    };
  },

  toJSON(message: Counterparty): unknown {
    const obj: any = {};
    message.portId !== undefined && (obj.portId = message.portId);
    message.channelId !== undefined && (obj.channelId = message.channelId);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Counterparty>, I>>(object: I): Counterparty {
    const message = createBaseCounterparty();
    message.portId = object.portId ?? "";
    message.channelId = object.channelId ?? "";
    return message;
  },
};

function createBasePacket(): Packet {
  return {
    sequence: 0,
    sourcePort: "",
    sourceChannel: "",
    destinationPort: "",
    destinationChannel: "",
    data: new Uint8Array(),
    timeoutHeight: undefined,
    timeoutTimestamp: 0,
  };
}

export const Packet = {
  encode(message: Packet, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sequence !== 0) {
      writer.uint32(8).uint64(message.sequence);
    }
    if (message.sourcePort !== "") {
      writer.uint32(18).string(message.sourcePort);
    }
    if (message.sourceChannel !== "") {
      writer.uint32(26).string(message.sourceChannel);
    }
    if (message.destinationPort !== "") {
      writer.uint32(34).string(message.destinationPort);
    }
    if (message.destinationChannel !== "") {
      writer.uint32(42).string(message.destinationChannel);
    }
    if (message.data.length !== 0) {
      writer.uint32(50).bytes(message.data);
    }
    if (message.timeoutHeight !== undefined) {
      Height.encode(message.timeoutHeight, writer.uint32(58).fork()).ldelim();
    }
    if (message.timeoutTimestamp !== 0) {
      writer.uint32(64).uint64(message.timeoutTimestamp);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Packet {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePacket();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sequence = longToNumber(reader.uint64() as Long);
          break;
        case 2:
          message.sourcePort = reader.string();
          break;
        case 3:
          message.sourceChannel = reader.string();
          break;
        case 4:
          message.destinationPort = reader.string();
          break;
        case 5:
          message.destinationChannel = reader.string();
          break;
        case 6:
          message.data = reader.bytes();
          break;
        case 7:
          message.timeoutHeight = Height.decode(reader, reader.uint32());
          break;
        case 8:
          message.timeoutTimestamp = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Packet {
    return {
      sequence: isSet(object.sequence) ? Number(object.sequence) : 0,
      sourcePort: isSet(object.sourcePort) ? String(object.sourcePort) : "",
      sourceChannel: isSet(object.sourceChannel) ? String(object.sourceChannel) : "",
      destinationPort: isSet(object.destinationPort) ? String(object.destinationPort) : "",
      destinationChannel: isSet(object.destinationChannel) ? String(object.destinationChannel) : "",
      data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(),
      timeoutHeight: isSet(object.timeoutHeight) ? Height.fromJSON(object.timeoutHeight) : undefined,
      timeoutTimestamp: isSet(object.timeoutTimestamp) ? Number(object.timeoutTimestamp) : 0,
    };
  },

  toJSON(message: Packet): unknown {
    const obj: any = {};
    message.sequence !== undefined && (obj.sequence = Math.round(message.sequence));
    message.sourcePort !== undefined && (obj.sourcePort = message.sourcePort);
    message.sourceChannel !== undefined && (obj.sourceChannel = message.sourceChannel);
    message.destinationPort !== undefined && (obj.destinationPort = message.destinationPort);
    message.destinationChannel !== undefined && (obj.destinationChannel = message.destinationChannel);
    message.data !== undefined
      && (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
    message.timeoutHeight !== undefined
      && (obj.timeoutHeight = message.timeoutHeight ? Height.toJSON(message.timeoutHeight) : undefined);
    message.timeoutTimestamp !== undefined && (obj.timeoutTimestamp = Math.round(message.timeoutTimestamp));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Packet>, I>>(object: I): Packet {
    const message = createBasePacket();
    message.sequence = object.sequence ?? 0;
    message.sourcePort = object.sourcePort ?? "";
    message.sourceChannel = object.sourceChannel ?? "";
    message.destinationPort = object.destinationPort ?? "";
    message.destinationChannel = object.destinationChannel ?? "";
    message.data = object.data ?? new Uint8Array();
    message.timeoutHeight = (object.timeoutHeight !== undefined && object.timeoutHeight !== null)
      ? Height.fromPartial(object.timeoutHeight)
      : undefined;
    message.timeoutTimestamp = object.timeoutTimestamp ?? 0;
    return message;
  },
};

function createBasePacketState(): PacketState {
  return { portId: "", channelId: "", sequence: 0, data: new Uint8Array() };
}

export const PacketState = {
  encode(message: PacketState, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.portId !== "") {
      writer.uint32(10).string(message.portId);
    }
    if (message.channelId !== "") {
      writer.uint32(18).string(message.channelId);
    }
    if (message.sequence !== 0) {
      writer.uint32(24).uint64(message.sequence);
    }
    if (message.data.length !== 0) {
      writer.uint32(34).bytes(message.data);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PacketState {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePacketState();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.portId = reader.string();
          break;
        case 2:
          message.channelId = reader.string();
          break;
        case 3:
          message.sequence = longToNumber(reader.uint64() as Long);
          break;
        case 4:
          message.data = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PacketState {
    return {
      portId: isSet(object.portId) ? String(object.portId) : "",
      channelId: isSet(object.channelId) ? String(object.channelId) : "",
      sequence: isSet(object.sequence) ? Number(object.sequence) : 0,
      data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(),
    };
  },

  toJSON(message: PacketState): unknown {
    const obj: any = {};
    message.portId !== undefined && (obj.portId = message.portId);
    message.channelId !== undefined && (obj.channelId = message.channelId);
    message.sequence !== undefined && (obj.sequence = Math.round(message.sequence));
    message.data !== undefined
      && (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PacketState>, I>>(object: I): PacketState {
    const message = createBasePacketState();
    message.portId = object.portId ?? "";
    message.channelId = object.channelId ?? "";
    message.sequence = object.sequence ?? 0;
    message.data = object.data ?? new Uint8Array();
    return message;
  },
};

function createBasePacketId(): PacketId {
  return { portId: "", channelId: "", sequence: 0 };
}

export const PacketId = {
  encode(message: PacketId, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.portId !== "") {
      writer.uint32(10).string(message.portId);
    }
    if (message.channelId !== "") {
      writer.uint32(18).string(message.channelId);
    }
    if (message.sequence !== 0) {
      writer.uint32(24).uint64(message.sequence);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PacketId {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePacketId();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.portId = reader.string();
          break;
        case 2:
          message.channelId = reader.string();
          break;
        case 3:
          message.sequence = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PacketId {
    return {
      portId: isSet(object.portId) ? String(object.portId) : "",
      channelId: isSet(object.channelId) ? String(object.channelId) : "",
      sequence: isSet(object.sequence) ? Number(object.sequence) : 0,
    };
  },

  toJSON(message: PacketId): unknown {
    const obj: any = {};
    message.portId !== undefined && (obj.portId = message.portId);
    message.channelId !== undefined && (obj.channelId = message.channelId);
    message.sequence !== undefined && (obj.sequence = Math.round(message.sequence));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PacketId>, I>>(object: I): PacketId {
    const message = createBasePacketId();
    message.portId = object.portId ?? "";
    message.channelId = object.channelId ?? "";
    message.sequence = object.sequence ?? 0;
    return message;
  },
};

function createBaseAcknowledgement(): Acknowledgement {
  return { result: undefined, error: undefined };
}

export const Acknowledgement = {
  encode(message: Acknowledgement, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.result !== undefined) {
      writer.uint32(170).bytes(message.result);
    }
    if (message.error !== undefined) {
      writer.uint32(178).string(message.error);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Acknowledgement {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAcknowledgement();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 21:
          message.result = reader.bytes();
          break;
        case 22:
          message.error = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Acknowledgement {
    return {
      result: isSet(object.result) ? bytesFromBase64(object.result) : undefined,
      error: isSet(object.error) ? String(object.error) : undefined,
    };
  },

  toJSON(message: Acknowledgement): unknown {
    const obj: any = {};
    message.result !== undefined
      && (obj.result = message.result !== undefined ? base64FromBytes(message.result) : undefined);
    message.error !== undefined && (obj.error = message.error);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Acknowledgement>, I>>(object: I): Acknowledgement {
    const message = createBaseAcknowledgement();
    message.result = object.result ?? undefined;
    message.error = object.error ?? undefined;
    return message;
  },
};

declare var self: any | undefined;
declare var window: any | undefined;
declare var global: any | undefined;
var globalThis: any = (() => {
  if (typeof globalThis !== "undefined") {
    return globalThis;
  }
  if (typeof self !== "undefined") {
    return self;
  }
  if (typeof window !== "undefined") {
    return window;
  }
  if (typeof global !== "undefined") {
    return global;
  }
  throw "Unable to locate global object";
})();

function bytesFromBase64(b64: string): Uint8Array {
  if (globalThis.Buffer) {
    return Uint8Array.from(globalThis.Buffer.from(b64, "base64"));
  } else {
    const bin = globalThis.atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; ++i) {
      arr[i] = bin.charCodeAt(i);
    }
    return arr;
  }
}

function base64FromBytes(arr: Uint8Array): string {
  if (globalThis.Buffer) {
    return globalThis.Buffer.from(arr).toString("base64");
  } else {
    const bin: string[] = [];
    arr.forEach((byte) => {
      bin.push(String.fromCharCode(byte));
    });
    return globalThis.btoa(bin.join(""));
  }
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
