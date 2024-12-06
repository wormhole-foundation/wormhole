//@ts-nocheck
/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "packetforward.v1";

/** GenesisState defines the packetforward genesis state */
export interface GenesisState {
  params:
    | Params
    | undefined;
  /**
   * key - information about forwarded packet: src_channel
   * (parsedReceiver.Channel), src_port (parsedReceiver.Port), sequence value -
   * information about original packet for refunding if necessary: retries,
   * srcPacketSender, srcPacket.DestinationChannel, srcPacket.DestinationPort
   */
  inFlightPackets: { [key: string]: InFlightPacket };
}

export interface GenesisState_InFlightPacketsEntry {
  key: string;
  value: InFlightPacket | undefined;
}

/** Params defines the set of packetforward parameters. */
export interface Params {
  feePercentage: string;
}

/**
 * InFlightPacket contains information about original packet for
 * writing the acknowledgement and refunding if necessary.
 */
export interface InFlightPacket {
  originalSenderAddress: string;
  refundChannelId: string;
  refundPortId: string;
  packetSrcChannelId: string;
  packetSrcPortId: string;
  packetTimeoutTimestamp: number;
  packetTimeoutHeight: string;
  packetData: Uint8Array;
  refundSequence: number;
  retriesRemaining: number;
  timeout: number;
  nonrefundable: boolean;
}

function createBaseGenesisState(): GenesisState {
  return { params: undefined, inFlightPackets: {} };
}

export const GenesisState = {
  encode(message: GenesisState, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    Object.entries(message.inFlightPackets).forEach(([key, value]) => {
      GenesisState_InFlightPacketsEntry.encode({ key: key as any, value }, writer.uint32(18).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisState();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          const entry2 = GenesisState_InFlightPacketsEntry.decode(reader, reader.uint32());
          if (entry2.value !== undefined) {
            message.inFlightPackets[entry2.key] = entry2.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GenesisState {
    return {
      params: isSet(object.params) ? Params.fromJSON(object.params) : undefined,
      inFlightPackets: isObject(object.inFlightPackets)
        ? Object.entries(object.inFlightPackets).reduce<{ [key: string]: InFlightPacket }>((acc, [key, value]) => {
          acc[key] = InFlightPacket.fromJSON(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    message.params !== undefined && (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    obj.inFlightPackets = {};
    if (message.inFlightPackets) {
      Object.entries(message.inFlightPackets).forEach(([k, v]) => {
        obj.inFlightPackets[k] = InFlightPacket.toJSON(v);
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GenesisState>, I>>(object: I): GenesisState {
    const message = createBaseGenesisState();
    message.params = (object.params !== undefined && object.params !== null)
      ? Params.fromPartial(object.params)
      : undefined;
    message.inFlightPackets = Object.entries(object.inFlightPackets ?? {}).reduce<{ [key: string]: InFlightPacket }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = InFlightPacket.fromPartial(value);
        }
        return acc;
      },
      {},
    );
    return message;
  },
};

function createBaseGenesisState_InFlightPacketsEntry(): GenesisState_InFlightPacketsEntry {
  return { key: "", value: undefined };
}

export const GenesisState_InFlightPacketsEntry = {
  encode(message: GenesisState_InFlightPacketsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      InFlightPacket.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GenesisState_InFlightPacketsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisState_InFlightPacketsEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = InFlightPacket.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GenesisState_InFlightPacketsEntry {
    return {
      key: isSet(object.key) ? String(object.key) : "",
      value: isSet(object.value) ? InFlightPacket.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: GenesisState_InFlightPacketsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined && (obj.value = message.value ? InFlightPacket.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GenesisState_InFlightPacketsEntry>, I>>(
    object: I,
  ): GenesisState_InFlightPacketsEntry {
    const message = createBaseGenesisState_InFlightPacketsEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null)
      ? InFlightPacket.fromPartial(object.value)
      : undefined;
    return message;
  },
};

function createBaseParams(): Params {
  return { feePercentage: "" };
}

export const Params = {
  encode(message: Params, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.feePercentage !== "") {
      writer.uint32(10).string(message.feePercentage);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Params {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.feePercentage = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Params {
    return { feePercentage: isSet(object.feePercentage) ? String(object.feePercentage) : "" };
  },

  toJSON(message: Params): unknown {
    const obj: any = {};
    message.feePercentage !== undefined && (obj.feePercentage = message.feePercentage);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Params>, I>>(object: I): Params {
    const message = createBaseParams();
    message.feePercentage = object.feePercentage ?? "";
    return message;
  },
};

function createBaseInFlightPacket(): InFlightPacket {
  return {
    originalSenderAddress: "",
    refundChannelId: "",
    refundPortId: "",
    packetSrcChannelId: "",
    packetSrcPortId: "",
    packetTimeoutTimestamp: 0,
    packetTimeoutHeight: "",
    packetData: new Uint8Array(),
    refundSequence: 0,
    retriesRemaining: 0,
    timeout: 0,
    nonrefundable: false,
  };
}

export const InFlightPacket = {
  encode(message: InFlightPacket, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.originalSenderAddress !== "") {
      writer.uint32(10).string(message.originalSenderAddress);
    }
    if (message.refundChannelId !== "") {
      writer.uint32(18).string(message.refundChannelId);
    }
    if (message.refundPortId !== "") {
      writer.uint32(26).string(message.refundPortId);
    }
    if (message.packetSrcChannelId !== "") {
      writer.uint32(34).string(message.packetSrcChannelId);
    }
    if (message.packetSrcPortId !== "") {
      writer.uint32(42).string(message.packetSrcPortId);
    }
    if (message.packetTimeoutTimestamp !== 0) {
      writer.uint32(48).uint64(message.packetTimeoutTimestamp);
    }
    if (message.packetTimeoutHeight !== "") {
      writer.uint32(58).string(message.packetTimeoutHeight);
    }
    if (message.packetData.length !== 0) {
      writer.uint32(66).bytes(message.packetData);
    }
    if (message.refundSequence !== 0) {
      writer.uint32(72).uint64(message.refundSequence);
    }
    if (message.retriesRemaining !== 0) {
      writer.uint32(80).int32(message.retriesRemaining);
    }
    if (message.timeout !== 0) {
      writer.uint32(88).uint64(message.timeout);
    }
    if (message.nonrefundable === true) {
      writer.uint32(96).bool(message.nonrefundable);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): InFlightPacket {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInFlightPacket();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.originalSenderAddress = reader.string();
          break;
        case 2:
          message.refundChannelId = reader.string();
          break;
        case 3:
          message.refundPortId = reader.string();
          break;
        case 4:
          message.packetSrcChannelId = reader.string();
          break;
        case 5:
          message.packetSrcPortId = reader.string();
          break;
        case 6:
          message.packetTimeoutTimestamp = longToNumber(reader.uint64() as Long);
          break;
        case 7:
          message.packetTimeoutHeight = reader.string();
          break;
        case 8:
          message.packetData = reader.bytes();
          break;
        case 9:
          message.refundSequence = longToNumber(reader.uint64() as Long);
          break;
        case 10:
          message.retriesRemaining = reader.int32();
          break;
        case 11:
          message.timeout = longToNumber(reader.uint64() as Long);
          break;
        case 12:
          message.nonrefundable = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): InFlightPacket {
    return {
      originalSenderAddress: isSet(object.originalSenderAddress) ? String(object.originalSenderAddress) : "",
      refundChannelId: isSet(object.refundChannelId) ? String(object.refundChannelId) : "",
      refundPortId: isSet(object.refundPortId) ? String(object.refundPortId) : "",
      packetSrcChannelId: isSet(object.packetSrcChannelId) ? String(object.packetSrcChannelId) : "",
      packetSrcPortId: isSet(object.packetSrcPortId) ? String(object.packetSrcPortId) : "",
      packetTimeoutTimestamp: isSet(object.packetTimeoutTimestamp) ? Number(object.packetTimeoutTimestamp) : 0,
      packetTimeoutHeight: isSet(object.packetTimeoutHeight) ? String(object.packetTimeoutHeight) : "",
      packetData: isSet(object.packetData) ? bytesFromBase64(object.packetData) : new Uint8Array(),
      refundSequence: isSet(object.refundSequence) ? Number(object.refundSequence) : 0,
      retriesRemaining: isSet(object.retriesRemaining) ? Number(object.retriesRemaining) : 0,
      timeout: isSet(object.timeout) ? Number(object.timeout) : 0,
      nonrefundable: isSet(object.nonrefundable) ? Boolean(object.nonrefundable) : false,
    };
  },

  toJSON(message: InFlightPacket): unknown {
    const obj: any = {};
    message.originalSenderAddress !== undefined && (obj.originalSenderAddress = message.originalSenderAddress);
    message.refundChannelId !== undefined && (obj.refundChannelId = message.refundChannelId);
    message.refundPortId !== undefined && (obj.refundPortId = message.refundPortId);
    message.packetSrcChannelId !== undefined && (obj.packetSrcChannelId = message.packetSrcChannelId);
    message.packetSrcPortId !== undefined && (obj.packetSrcPortId = message.packetSrcPortId);
    message.packetTimeoutTimestamp !== undefined
      && (obj.packetTimeoutTimestamp = Math.round(message.packetTimeoutTimestamp));
    message.packetTimeoutHeight !== undefined && (obj.packetTimeoutHeight = message.packetTimeoutHeight);
    message.packetData !== undefined
      && (obj.packetData = base64FromBytes(message.packetData !== undefined ? message.packetData : new Uint8Array()));
    message.refundSequence !== undefined && (obj.refundSequence = Math.round(message.refundSequence));
    message.retriesRemaining !== undefined && (obj.retriesRemaining = Math.round(message.retriesRemaining));
    message.timeout !== undefined && (obj.timeout = Math.round(message.timeout));
    message.nonrefundable !== undefined && (obj.nonrefundable = message.nonrefundable);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<InFlightPacket>, I>>(object: I): InFlightPacket {
    const message = createBaseInFlightPacket();
    message.originalSenderAddress = object.originalSenderAddress ?? "";
    message.refundChannelId = object.refundChannelId ?? "";
    message.refundPortId = object.refundPortId ?? "";
    message.packetSrcChannelId = object.packetSrcChannelId ?? "";
    message.packetSrcPortId = object.packetSrcPortId ?? "";
    message.packetTimeoutTimestamp = object.packetTimeoutTimestamp ?? 0;
    message.packetTimeoutHeight = object.packetTimeoutHeight ?? "";
    message.packetData = object.packetData ?? new Uint8Array();
    message.refundSequence = object.refundSequence ?? 0;
    message.retriesRemaining = object.retriesRemaining ?? 0;
    message.timeout = object.timeout ?? 0;
    message.nonrefundable = object.nonrefundable ?? false;
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

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
