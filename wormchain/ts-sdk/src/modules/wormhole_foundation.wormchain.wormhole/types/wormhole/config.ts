//@ts-nocheck
/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "wormhole_foundation.wormchain.wormhole";

export interface Config {
  guardian_set_expiration: number;
  governance_emitter: Uint8Array;
  governance_chain: number;
  chain_id: number;
}

const baseConfig: object = {
  guardian_set_expiration: 0,
  governance_chain: 0,
  chain_id: 0,
};

export const Config = {
  encode(message: Config, writer: Writer = Writer.create()): Writer {
    if (message.guardian_set_expiration !== 0) {
      writer.uint32(8).uint64(message.guardian_set_expiration);
    }
    if (message.governance_emitter.length !== 0) {
      writer.uint32(18).bytes(message.governance_emitter);
    }
    if (message.governance_chain !== 0) {
      writer.uint32(24).uint32(message.governance_chain);
    }
    if (message.chain_id !== 0) {
      writer.uint32(32).uint32(message.chain_id);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): Config {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseConfig } as Config;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.guardian_set_expiration = longToNumber(
            reader.uint64() as Long
          );
          break;
        case 2:
          message.governance_emitter = reader.bytes();
          break;
        case 3:
          message.governance_chain = reader.uint32();
          break;
        case 4:
          message.chain_id = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Config {
    const message = { ...baseConfig } as Config;
    if (
      object.guardian_set_expiration !== undefined &&
      object.guardian_set_expiration !== null
    ) {
      message.guardian_set_expiration = Number(object.guardian_set_expiration);
    } else {
      message.guardian_set_expiration = 0;
    }
    if (
      object.governance_emitter !== undefined &&
      object.governance_emitter !== null
    ) {
      message.governance_emitter = bytesFromBase64(object.governance_emitter);
    }
    if (
      object.governance_chain !== undefined &&
      object.governance_chain !== null
    ) {
      message.governance_chain = Number(object.governance_chain);
    } else {
      message.governance_chain = 0;
    }
    if (object.chain_id !== undefined && object.chain_id !== null) {
      message.chain_id = Number(object.chain_id);
    } else {
      message.chain_id = 0;
    }
    return message;
  },

  toJSON(message: Config): unknown {
    const obj: any = {};
    message.guardian_set_expiration !== undefined &&
      (obj.guardian_set_expiration = message.guardian_set_expiration);
    message.governance_emitter !== undefined &&
      (obj.governance_emitter = base64FromBytes(
        message.governance_emitter !== undefined
          ? message.governance_emitter
          : new Uint8Array()
      ));
    message.governance_chain !== undefined &&
      (obj.governance_chain = message.governance_chain);
    message.chain_id !== undefined && (obj.chain_id = message.chain_id);
    return obj;
  },

  fromPartial(object: DeepPartial<Config>): Config {
    const message = { ...baseConfig } as Config;
    if (
      object.guardian_set_expiration !== undefined &&
      object.guardian_set_expiration !== null
    ) {
      message.guardian_set_expiration = object.guardian_set_expiration;
    } else {
      message.guardian_set_expiration = 0;
    }
    if (
      object.governance_emitter !== undefined &&
      object.governance_emitter !== null
    ) {
      message.governance_emitter = object.governance_emitter;
    } else {
      message.governance_emitter = new Uint8Array();
    }
    if (
      object.governance_chain !== undefined &&
      object.governance_chain !== null
    ) {
      message.governance_chain = object.governance_chain;
    } else {
      message.governance_chain = 0;
    }
    if (object.chain_id !== undefined && object.chain_id !== null) {
      message.chain_id = object.chain_id;
    } else {
      message.chain_id = 0;
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
