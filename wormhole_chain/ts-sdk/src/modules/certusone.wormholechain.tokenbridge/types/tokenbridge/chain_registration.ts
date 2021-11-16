//@ts-nocheck
/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "certusone.wormholechain.tokenbridge";

export interface ChainRegistration {
  chainID: number;
  emitterAddress: Uint8Array;
}

const baseChainRegistration: object = { chainID: 0 };

export const ChainRegistration = {
  encode(message: ChainRegistration, writer: Writer = Writer.create()): Writer {
    if (message.chainID !== 0) {
      writer.uint32(8).uint32(message.chainID);
    }
    if (message.emitterAddress.length !== 0) {
      writer.uint32(18).bytes(message.emitterAddress);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): ChainRegistration {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseChainRegistration } as ChainRegistration;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.chainID = reader.uint32();
          break;
        case 2:
          message.emitterAddress = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ChainRegistration {
    const message = { ...baseChainRegistration } as ChainRegistration;
    if (object.chainID !== undefined && object.chainID !== null) {
      message.chainID = Number(object.chainID);
    } else {
      message.chainID = 0;
    }
    if (object.emitterAddress !== undefined && object.emitterAddress !== null) {
      message.emitterAddress = bytesFromBase64(object.emitterAddress);
    }
    return message;
  },

  toJSON(message: ChainRegistration): unknown {
    const obj: any = {};
    message.chainID !== undefined && (obj.chainID = message.chainID);
    message.emitterAddress !== undefined &&
      (obj.emitterAddress = base64FromBytes(
        message.emitterAddress !== undefined
          ? message.emitterAddress
          : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(object: DeepPartial<ChainRegistration>): ChainRegistration {
    const message = { ...baseChainRegistration } as ChainRegistration;
    if (object.chainID !== undefined && object.chainID !== null) {
      message.chainID = object.chainID;
    } else {
      message.chainID = 0;
    }
    if (object.emitterAddress !== undefined && object.emitterAddress !== null) {
      message.emitterAddress = object.emitterAddress;
    } else {
      message.emitterAddress = new Uint8Array();
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
