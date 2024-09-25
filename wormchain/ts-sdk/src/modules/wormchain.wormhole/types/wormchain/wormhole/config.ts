//@ts-nocheck
/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "wormchain.wormhole";

export interface Config {
  guardianSetExpiration: number;
  governanceEmitter: Uint8Array;
  governanceChain: number;
  chainId: number;
}

function createBaseConfig(): Config {
  return { guardianSetExpiration: 0, governanceEmitter: new Uint8Array(), governanceChain: 0, chainId: 0 };
}

export const Config = {
  encode(message: Config, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.guardianSetExpiration !== 0) {
      writer.uint32(8).uint64(message.guardianSetExpiration);
    }
    if (message.governanceEmitter.length !== 0) {
      writer.uint32(18).bytes(message.governanceEmitter);
    }
    if (message.governanceChain !== 0) {
      writer.uint32(24).uint32(message.governanceChain);
    }
    if (message.chainId !== 0) {
      writer.uint32(32).uint32(message.chainId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Config {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.guardianSetExpiration = longToNumber(reader.uint64() as Long);
          break;
        case 2:
          message.governanceEmitter = reader.bytes();
          break;
        case 3:
          message.governanceChain = reader.uint32();
          break;
        case 4:
          message.chainId = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Config {
    return {
      guardianSetExpiration: isSet(object.guardianSetExpiration) ? Number(object.guardianSetExpiration) : 0,
      governanceEmitter: isSet(object.governanceEmitter) ? bytesFromBase64(object.governanceEmitter) : new Uint8Array(),
      governanceChain: isSet(object.governanceChain) ? Number(object.governanceChain) : 0,
      chainId: isSet(object.chainId) ? Number(object.chainId) : 0,
    };
  },

  toJSON(message: Config): unknown {
    const obj: any = {};
    message.guardianSetExpiration !== undefined
      && (obj.guardianSetExpiration = Math.round(message.guardianSetExpiration));
    message.governanceEmitter !== undefined
      && (obj.governanceEmitter = base64FromBytes(
        message.governanceEmitter !== undefined ? message.governanceEmitter : new Uint8Array(),
      ));
    message.governanceChain !== undefined && (obj.governanceChain = Math.round(message.governanceChain));
    message.chainId !== undefined && (obj.chainId = Math.round(message.chainId));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Config>, I>>(object: I): Config {
    const message = createBaseConfig();
    message.guardianSetExpiration = object.guardianSetExpiration ?? 0;
    message.governanceEmitter = object.governanceEmitter ?? new Uint8Array();
    message.governanceChain = object.governanceChain ?? 0;
    message.chainId = object.chainId ?? 0;
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
