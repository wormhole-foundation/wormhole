//@ts-nocheck
/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "certusone.wormholechain.wormhole";

export interface EventGuardianSetUpdate {
  oldIndex: number;
  newIndex: number;
}

export interface EventPostedMessage {
  emitter: Uint8Array;
  sequence: number;
  nonce: number;
  time: number;
  payload: Uint8Array;
}

export interface EventGuardianRegistered {
  guardianKey: Uint8Array;
  validatorKey: Uint8Array;
}

export interface EventConsensusSetUpdate {
  oldIndex: number;
  newIndex: number;
}

const baseEventGuardianSetUpdate: object = { oldIndex: 0, newIndex: 0 };

export const EventGuardianSetUpdate = {
  encode(
    message: EventGuardianSetUpdate,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.oldIndex !== 0) {
      writer.uint32(8).uint32(message.oldIndex);
    }
    if (message.newIndex !== 0) {
      writer.uint32(16).uint32(message.newIndex);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): EventGuardianSetUpdate {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseEventGuardianSetUpdate } as EventGuardianSetUpdate;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.oldIndex = reader.uint32();
          break;
        case 2:
          message.newIndex = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): EventGuardianSetUpdate {
    const message = { ...baseEventGuardianSetUpdate } as EventGuardianSetUpdate;
    if (object.oldIndex !== undefined && object.oldIndex !== null) {
      message.oldIndex = Number(object.oldIndex);
    } else {
      message.oldIndex = 0;
    }
    if (object.newIndex !== undefined && object.newIndex !== null) {
      message.newIndex = Number(object.newIndex);
    } else {
      message.newIndex = 0;
    }
    return message;
  },

  toJSON(message: EventGuardianSetUpdate): unknown {
    const obj: any = {};
    message.oldIndex !== undefined && (obj.oldIndex = message.oldIndex);
    message.newIndex !== undefined && (obj.newIndex = message.newIndex);
    return obj;
  },

  fromPartial(
    object: DeepPartial<EventGuardianSetUpdate>
  ): EventGuardianSetUpdate {
    const message = { ...baseEventGuardianSetUpdate } as EventGuardianSetUpdate;
    if (object.oldIndex !== undefined && object.oldIndex !== null) {
      message.oldIndex = object.oldIndex;
    } else {
      message.oldIndex = 0;
    }
    if (object.newIndex !== undefined && object.newIndex !== null) {
      message.newIndex = object.newIndex;
    } else {
      message.newIndex = 0;
    }
    return message;
  },
};

const baseEventPostedMessage: object = { sequence: 0, nonce: 0, time: 0 };

export const EventPostedMessage = {
  encode(
    message: EventPostedMessage,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.emitter.length !== 0) {
      writer.uint32(10).bytes(message.emitter);
    }
    if (message.sequence !== 0) {
      writer.uint32(16).uint64(message.sequence);
    }
    if (message.nonce !== 0) {
      writer.uint32(24).uint32(message.nonce);
    }
    if (message.time !== 0) {
      writer.uint32(32).uint64(message.time);
    }
    if (message.payload.length !== 0) {
      writer.uint32(42).bytes(message.payload);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): EventPostedMessage {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseEventPostedMessage } as EventPostedMessage;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.emitter = reader.bytes();
          break;
        case 2:
          message.sequence = longToNumber(reader.uint64() as Long);
          break;
        case 3:
          message.nonce = reader.uint32();
          break;
        case 4:
          message.time = longToNumber(reader.uint64() as Long);
          break;
        case 5:
          message.payload = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): EventPostedMessage {
    const message = { ...baseEventPostedMessage } as EventPostedMessage;
    if (object.emitter !== undefined && object.emitter !== null) {
      message.emitter = bytesFromBase64(object.emitter);
    }
    if (object.sequence !== undefined && object.sequence !== null) {
      message.sequence = Number(object.sequence);
    } else {
      message.sequence = 0;
    }
    if (object.nonce !== undefined && object.nonce !== null) {
      message.nonce = Number(object.nonce);
    } else {
      message.nonce = 0;
    }
    if (object.time !== undefined && object.time !== null) {
      message.time = Number(object.time);
    } else {
      message.time = 0;
    }
    if (object.payload !== undefined && object.payload !== null) {
      message.payload = bytesFromBase64(object.payload);
    }
    return message;
  },

  toJSON(message: EventPostedMessage): unknown {
    const obj: any = {};
    message.emitter !== undefined &&
      (obj.emitter = base64FromBytes(
        message.emitter !== undefined ? message.emitter : new Uint8Array()
      ));
    message.sequence !== undefined && (obj.sequence = message.sequence);
    message.nonce !== undefined && (obj.nonce = message.nonce);
    message.time !== undefined && (obj.time = message.time);
    message.payload !== undefined &&
      (obj.payload = base64FromBytes(
        message.payload !== undefined ? message.payload : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(object: DeepPartial<EventPostedMessage>): EventPostedMessage {
    const message = { ...baseEventPostedMessage } as EventPostedMessage;
    if (object.emitter !== undefined && object.emitter !== null) {
      message.emitter = object.emitter;
    } else {
      message.emitter = new Uint8Array();
    }
    if (object.sequence !== undefined && object.sequence !== null) {
      message.sequence = object.sequence;
    } else {
      message.sequence = 0;
    }
    if (object.nonce !== undefined && object.nonce !== null) {
      message.nonce = object.nonce;
    } else {
      message.nonce = 0;
    }
    if (object.time !== undefined && object.time !== null) {
      message.time = object.time;
    } else {
      message.time = 0;
    }
    if (object.payload !== undefined && object.payload !== null) {
      message.payload = object.payload;
    } else {
      message.payload = new Uint8Array();
    }
    return message;
  },
};

const baseEventGuardianRegistered: object = {};

export const EventGuardianRegistered = {
  encode(
    message: EventGuardianRegistered,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.guardianKey.length !== 0) {
      writer.uint32(10).bytes(message.guardianKey);
    }
    if (message.validatorKey.length !== 0) {
      writer.uint32(18).bytes(message.validatorKey);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): EventGuardianRegistered {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseEventGuardianRegistered,
    } as EventGuardianRegistered;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.guardianKey = reader.bytes();
          break;
        case 2:
          message.validatorKey = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): EventGuardianRegistered {
    const message = {
      ...baseEventGuardianRegistered,
    } as EventGuardianRegistered;
    if (object.guardianKey !== undefined && object.guardianKey !== null) {
      message.guardianKey = bytesFromBase64(object.guardianKey);
    }
    if (object.validatorKey !== undefined && object.validatorKey !== null) {
      message.validatorKey = bytesFromBase64(object.validatorKey);
    }
    return message;
  },

  toJSON(message: EventGuardianRegistered): unknown {
    const obj: any = {};
    message.guardianKey !== undefined &&
      (obj.guardianKey = base64FromBytes(
        message.guardianKey !== undefined
          ? message.guardianKey
          : new Uint8Array()
      ));
    message.validatorKey !== undefined &&
      (obj.validatorKey = base64FromBytes(
        message.validatorKey !== undefined
          ? message.validatorKey
          : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(
    object: DeepPartial<EventGuardianRegistered>
  ): EventGuardianRegistered {
    const message = {
      ...baseEventGuardianRegistered,
    } as EventGuardianRegistered;
    if (object.guardianKey !== undefined && object.guardianKey !== null) {
      message.guardianKey = object.guardianKey;
    } else {
      message.guardianKey = new Uint8Array();
    }
    if (object.validatorKey !== undefined && object.validatorKey !== null) {
      message.validatorKey = object.validatorKey;
    } else {
      message.validatorKey = new Uint8Array();
    }
    return message;
  },
};

const baseEventConsensusSetUpdate: object = { oldIndex: 0, newIndex: 0 };

export const EventConsensusSetUpdate = {
  encode(
    message: EventConsensusSetUpdate,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.oldIndex !== 0) {
      writer.uint32(8).uint32(message.oldIndex);
    }
    if (message.newIndex !== 0) {
      writer.uint32(16).uint32(message.newIndex);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): EventConsensusSetUpdate {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseEventConsensusSetUpdate,
    } as EventConsensusSetUpdate;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.oldIndex = reader.uint32();
          break;
        case 2:
          message.newIndex = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): EventConsensusSetUpdate {
    const message = {
      ...baseEventConsensusSetUpdate,
    } as EventConsensusSetUpdate;
    if (object.oldIndex !== undefined && object.oldIndex !== null) {
      message.oldIndex = Number(object.oldIndex);
    } else {
      message.oldIndex = 0;
    }
    if (object.newIndex !== undefined && object.newIndex !== null) {
      message.newIndex = Number(object.newIndex);
    } else {
      message.newIndex = 0;
    }
    return message;
  },

  toJSON(message: EventConsensusSetUpdate): unknown {
    const obj: any = {};
    message.oldIndex !== undefined && (obj.oldIndex = message.oldIndex);
    message.newIndex !== undefined && (obj.newIndex = message.newIndex);
    return obj;
  },

  fromPartial(
    object: DeepPartial<EventConsensusSetUpdate>
  ): EventConsensusSetUpdate {
    const message = {
      ...baseEventConsensusSetUpdate,
    } as EventConsensusSetUpdate;
    if (object.oldIndex !== undefined && object.oldIndex !== null) {
      message.oldIndex = object.oldIndex;
    } else {
      message.oldIndex = 0;
    }
    if (object.newIndex !== undefined && object.newIndex !== null) {
      message.newIndex = object.newIndex;
    } else {
      message.newIndex = 0;
    }
    return message;
  },
};

declare var self: any | undefined;
declare var window: any | undefined;
var globalThis: any = (() => {
  if (typeof globalThis !== "undefined") return globalThis;
  if (typeof self !== "undefined") return self;
  if (typeof window !== "undefined") return window;
  if (typeof global !== "undefined") return global;
  throw "Unable to locate global object";
})();

const atob: (b64: string) => string =
  globalThis.atob ||
  ((b64) => globalThis.Buffer.from(b64, "base64").toString("binary"));
function bytesFromBase64(b64: string): Uint8Array {
  const bin = atob(b64);
  const arr = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; ++i) {
    arr[i] = bin.charCodeAt(i);
  }
  return arr;
}

const btoa: (bin: string) => string =
  globalThis.btoa ||
  ((bin) => globalThis.Buffer.from(bin, "binary").toString("base64"));
function base64FromBytes(arr: Uint8Array): string {
  const bin: string[] = [];
  for (let i = 0; i < arr.byteLength; ++i) {
    bin.push(String.fromCharCode(arr[i]));
  }
  return btoa(bin.join(""));
}

type Builtin = Date | Function | Uint8Array | string | number | undefined;
export type DeepPartial<T> = T extends Builtin
  ? T
  : T extends Array<infer U>
  ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U>
  ? ReadonlyArray<DeepPartial<U>>
  : T extends {}
  ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

if (util.Long !== Long) {
  util.Long = Long as any;
  configure();
}
