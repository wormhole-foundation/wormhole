//@ts-nocheck
/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "wormchain.wormhole";

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

function createBaseEventGuardianSetUpdate(): EventGuardianSetUpdate {
  return { oldIndex: 0, newIndex: 0 };
}

export const EventGuardianSetUpdate = {
  encode(message: EventGuardianSetUpdate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.oldIndex !== 0) {
      writer.uint32(8).uint32(message.oldIndex);
    }
    if (message.newIndex !== 0) {
      writer.uint32(16).uint32(message.newIndex);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): EventGuardianSetUpdate {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEventGuardianSetUpdate();
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
    return {
      oldIndex: isSet(object.oldIndex) ? Number(object.oldIndex) : 0,
      newIndex: isSet(object.newIndex) ? Number(object.newIndex) : 0,
    };
  },

  toJSON(message: EventGuardianSetUpdate): unknown {
    const obj: any = {};
    message.oldIndex !== undefined && (obj.oldIndex = Math.round(message.oldIndex));
    message.newIndex !== undefined && (obj.newIndex = Math.round(message.newIndex));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<EventGuardianSetUpdate>, I>>(object: I): EventGuardianSetUpdate {
    const message = createBaseEventGuardianSetUpdate();
    message.oldIndex = object.oldIndex ?? 0;
    message.newIndex = object.newIndex ?? 0;
    return message;
  },
};

function createBaseEventPostedMessage(): EventPostedMessage {
  return { emitter: new Uint8Array(), sequence: 0, nonce: 0, time: 0, payload: new Uint8Array() };
}

export const EventPostedMessage = {
  encode(message: EventPostedMessage, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): EventPostedMessage {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEventPostedMessage();
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
    return {
      emitter: isSet(object.emitter) ? bytesFromBase64(object.emitter) : new Uint8Array(),
      sequence: isSet(object.sequence) ? Number(object.sequence) : 0,
      nonce: isSet(object.nonce) ? Number(object.nonce) : 0,
      time: isSet(object.time) ? Number(object.time) : 0,
      payload: isSet(object.payload) ? bytesFromBase64(object.payload) : new Uint8Array(),
    };
  },

  toJSON(message: EventPostedMessage): unknown {
    const obj: any = {};
    message.emitter !== undefined
      && (obj.emitter = base64FromBytes(message.emitter !== undefined ? message.emitter : new Uint8Array()));
    message.sequence !== undefined && (obj.sequence = Math.round(message.sequence));
    message.nonce !== undefined && (obj.nonce = Math.round(message.nonce));
    message.time !== undefined && (obj.time = Math.round(message.time));
    message.payload !== undefined
      && (obj.payload = base64FromBytes(message.payload !== undefined ? message.payload : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<EventPostedMessage>, I>>(object: I): EventPostedMessage {
    const message = createBaseEventPostedMessage();
    message.emitter = object.emitter ?? new Uint8Array();
    message.sequence = object.sequence ?? 0;
    message.nonce = object.nonce ?? 0;
    message.time = object.time ?? 0;
    message.payload = object.payload ?? new Uint8Array();
    return message;
  },
};

function createBaseEventGuardianRegistered(): EventGuardianRegistered {
  return { guardianKey: new Uint8Array(), validatorKey: new Uint8Array() };
}

export const EventGuardianRegistered = {
  encode(message: EventGuardianRegistered, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.guardianKey.length !== 0) {
      writer.uint32(10).bytes(message.guardianKey);
    }
    if (message.validatorKey.length !== 0) {
      writer.uint32(18).bytes(message.validatorKey);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): EventGuardianRegistered {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEventGuardianRegistered();
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
    return {
      guardianKey: isSet(object.guardianKey) ? bytesFromBase64(object.guardianKey) : new Uint8Array(),
      validatorKey: isSet(object.validatorKey) ? bytesFromBase64(object.validatorKey) : new Uint8Array(),
    };
  },

  toJSON(message: EventGuardianRegistered): unknown {
    const obj: any = {};
    message.guardianKey !== undefined
      && (obj.guardianKey = base64FromBytes(
        message.guardianKey !== undefined ? message.guardianKey : new Uint8Array(),
      ));
    message.validatorKey !== undefined
      && (obj.validatorKey = base64FromBytes(
        message.validatorKey !== undefined ? message.validatorKey : new Uint8Array(),
      ));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<EventGuardianRegistered>, I>>(object: I): EventGuardianRegistered {
    const message = createBaseEventGuardianRegistered();
    message.guardianKey = object.guardianKey ?? new Uint8Array();
    message.validatorKey = object.validatorKey ?? new Uint8Array();
    return message;
  },
};

function createBaseEventConsensusSetUpdate(): EventConsensusSetUpdate {
  return { oldIndex: 0, newIndex: 0 };
}

export const EventConsensusSetUpdate = {
  encode(message: EventConsensusSetUpdate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.oldIndex !== 0) {
      writer.uint32(8).uint32(message.oldIndex);
    }
    if (message.newIndex !== 0) {
      writer.uint32(16).uint32(message.newIndex);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): EventConsensusSetUpdate {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEventConsensusSetUpdate();
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
    return {
      oldIndex: isSet(object.oldIndex) ? Number(object.oldIndex) : 0,
      newIndex: isSet(object.newIndex) ? Number(object.newIndex) : 0,
    };
  },

  toJSON(message: EventConsensusSetUpdate): unknown {
    const obj: any = {};
    message.oldIndex !== undefined && (obj.oldIndex = Math.round(message.oldIndex));
    message.newIndex !== undefined && (obj.newIndex = Math.round(message.newIndex));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<EventConsensusSetUpdate>, I>>(object: I): EventConsensusSetUpdate {
    const message = createBaseEventConsensusSetUpdate();
    message.oldIndex = object.oldIndex ?? 0;
    message.newIndex = object.newIndex ?? 0;
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
