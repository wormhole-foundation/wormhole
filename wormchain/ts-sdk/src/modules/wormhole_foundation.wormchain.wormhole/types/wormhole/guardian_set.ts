//@ts-nocheck
//@ts-nocheck
/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "wormhole_foundation.wormchain.wormhole";

export interface GuardianSet {
  index: number;
  keys: Uint8Array[];
  expirationTime: number;
}

const baseGuardianSet: object = { index: 0, expirationTime: 0 };

export const GuardianSet = {
  encode(message: GuardianSet, writer: Writer = Writer.create()): Writer {
    if (message.index !== 0) {
      writer.uint32(8).uint32(message.index);
    }
    for (const v of message.keys) {
      writer.uint32(18).bytes(v!);
    }
    if (message.expirationTime !== 0) {
      writer.uint32(24).uint64(message.expirationTime);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GuardianSet {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGuardianSet } as GuardianSet;
    message.keys = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.uint32();
          break;
        case 2:
          message.keys.push(reader.bytes());
          break;
        case 3:
          message.expirationTime = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GuardianSet {
    const message = { ...baseGuardianSet } as GuardianSet;
    message.keys = [];
    if (object.index !== undefined && object.index !== null) {
      message.index = Number(object.index);
    } else {
      message.index = 0;
    }
    if (object.keys !== undefined && object.keys !== null) {
      for (const e of object.keys) {
        message.keys.push(bytesFromBase64(e));
      }
    }
    if (object.expirationTime !== undefined && object.expirationTime !== null) {
      message.expirationTime = Number(object.expirationTime);
    } else {
      message.expirationTime = 0;
    }
    return message;
  },

  toJSON(message: GuardianSet): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    if (message.keys) {
      obj.keys = message.keys.map((e) =>
        base64FromBytes(e !== undefined ? e : new Uint8Array())
      );
    } else {
      obj.keys = [];
    }
    message.expirationTime !== undefined &&
      (obj.expirationTime = message.expirationTime);
    return obj;
  },

  fromPartial(object: DeepPartial<GuardianSet>): GuardianSet {
    const message = { ...baseGuardianSet } as GuardianSet;
    message.keys = [];
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = 0;
    }
    if (object.keys !== undefined && object.keys !== null) {
      for (const e of object.keys) {
        message.keys.push(e);
      }
    }
    if (object.expirationTime !== undefined && object.expirationTime !== null) {
      message.expirationTime = object.expirationTime;
    } else {
      message.expirationTime = 0;
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
