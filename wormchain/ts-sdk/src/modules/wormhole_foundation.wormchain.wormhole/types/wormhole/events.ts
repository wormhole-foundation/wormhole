//@ts-nocheck
/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "wormhole_foundation.wormchain.wormhole";

export interface EventGuardianSetUpdate {
  old_index: number;
  new_index: number;
}

export interface EventPostedMessage {
  emitter: Uint8Array;
  sequence: number;
  nonce: number;
  time: number;
  payload: Uint8Array;
}

export interface EventGuardianRegistered {
  guardian_key: Uint8Array;
  validator_key: Uint8Array;
}

export interface EventConsensusSetUpdate {
  old_index: number;
  new_index: number;
}

const baseEventGuardianSetUpdate: object = { old_index: 0, new_index: 0 };

export const EventGuardianSetUpdate = {
  encode(
    message: EventGuardianSetUpdate,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.old_index !== 0) {
      writer.uint32(8).uint32(message.old_index);
    }
    if (message.new_index !== 0) {
      writer.uint32(16).uint32(message.new_index);
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
          message.old_index = reader.uint32();
          break;
        case 2:
          message.new_index = reader.uint32();
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
    if (object.old_index !== undefined && object.old_index !== null) {
      message.old_index = Number(object.old_index);
    } else {
      message.old_index = 0;
    }
    if (object.new_index !== undefined && object.new_index !== null) {
      message.new_index = Number(object.new_index);
    } else {
      message.new_index = 0;
    }
    return message;
  },

  toJSON(message: EventGuardianSetUpdate): unknown {
    const obj: any = {};
    message.old_index !== undefined && (obj.old_index = message.old_index);
    message.new_index !== undefined && (obj.new_index = message.new_index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<EventGuardianSetUpdate>
  ): EventGuardianSetUpdate {
    const message = { ...baseEventGuardianSetUpdate } as EventGuardianSetUpdate;
    if (object.old_index !== undefined && object.old_index !== null) {
      message.old_index = object.old_index;
    } else {
      message.old_index = 0;
    }
    if (object.new_index !== undefined && object.new_index !== null) {
      message.new_index = object.new_index;
    } else {
      message.new_index = 0;
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
    if (message.guardian_key.length !== 0) {
      writer.uint32(10).bytes(message.guardian_key);
    }
    if (message.validator_key.length !== 0) {
      writer.uint32(18).bytes(message.validator_key);
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
          message.guardian_key = reader.bytes();
          break;
        case 2:
          message.validator_key = reader.bytes();
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
    if (object.guardian_key !== undefined && object.guardian_key !== null) {
      message.guardian_key = bytesFromBase64(object.guardian_key);
    }
    if (object.validator_key !== undefined && object.validator_key !== null) {
      message.validator_key = bytesFromBase64(object.validator_key);
    }
    return message;
  },

  toJSON(message: EventGuardianRegistered): unknown {
    const obj: any = {};
    message.guardian_key !== undefined &&
      (obj.guardian_key = base64FromBytes(
        message.guardian_key !== undefined
          ? message.guardian_key
          : new Uint8Array()
      ));
    message.validator_key !== undefined &&
      (obj.validator_key = base64FromBytes(
        message.validator_key !== undefined
          ? message.validator_key
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
    if (object.guardian_key !== undefined && object.guardian_key !== null) {
      message.guardian_key = object.guardian_key;
    } else {
      message.guardian_key = new Uint8Array();
    }
    if (object.validator_key !== undefined && object.validator_key !== null) {
      message.validator_key = object.validator_key;
    } else {
      message.validator_key = new Uint8Array();
    }
    return message;
  },
};

const baseEventConsensusSetUpdate: object = { old_index: 0, new_index: 0 };

export const EventConsensusSetUpdate = {
  encode(
    message: EventConsensusSetUpdate,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.old_index !== 0) {
      writer.uint32(8).uint32(message.old_index);
    }
    if (message.new_index !== 0) {
      writer.uint32(16).uint32(message.new_index);
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
          message.old_index = reader.uint32();
          break;
        case 2:
          message.new_index = reader.uint32();
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
    if (object.old_index !== undefined && object.old_index !== null) {
      message.old_index = Number(object.old_index);
    } else {
      message.old_index = 0;
    }
    if (object.new_index !== undefined && object.new_index !== null) {
      message.new_index = Number(object.new_index);
    } else {
      message.new_index = 0;
    }
    return message;
  },

  toJSON(message: EventConsensusSetUpdate): unknown {
    const obj: any = {};
    message.old_index !== undefined && (obj.old_index = message.old_index);
    message.new_index !== undefined && (obj.new_index = message.new_index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<EventConsensusSetUpdate>
  ): EventConsensusSetUpdate {
    const message = {
      ...baseEventConsensusSetUpdate,
    } as EventConsensusSetUpdate;
    if (object.old_index !== undefined && object.old_index !== null) {
      message.old_index = object.old_index;
    } else {
      message.old_index = 0;
    }
    if (object.new_index !== undefined && object.new_index !== null) {
      message.new_index = object.new_index;
    } else {
      message.new_index = 0;
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
