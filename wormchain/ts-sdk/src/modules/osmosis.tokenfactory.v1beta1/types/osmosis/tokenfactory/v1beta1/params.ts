//@ts-nocheck
/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Coin } from "../../../cosmos/base/v1beta1/coin";

export const protobufPackage = "osmosis.tokenfactory.v1beta1";

/** Params defines the parameters for the tokenfactory module. */
export interface Params {
  denomCreationFee: Coin[];
  /**
   * if denom_creation_fee is an empty array, then this field is used to add
   * more gas consumption to the base cost.
   * https://github.com/CosmWasm/token-factory/issues/11
   */
  denomCreationGasConsume: number;
}

function createBaseParams(): Params {
  return { denomCreationFee: [], denomCreationGasConsume: 0 };
}

export const Params = {
  encode(message: Params, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.denomCreationFee) {
      Coin.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.denomCreationGasConsume !== 0) {
      writer.uint32(16).uint64(message.denomCreationGasConsume);
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
          message.denomCreationFee.push(Coin.decode(reader, reader.uint32()));
          break;
        case 2:
          message.denomCreationGasConsume = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Params {
    return {
      denomCreationFee: Array.isArray(object?.denomCreationFee)
        ? object.denomCreationFee.map((e: any) => Coin.fromJSON(e))
        : [],
      denomCreationGasConsume: isSet(object.denomCreationGasConsume) ? Number(object.denomCreationGasConsume) : 0,
    };
  },

  toJSON(message: Params): unknown {
    const obj: any = {};
    if (message.denomCreationFee) {
      obj.denomCreationFee = message.denomCreationFee.map((e) => e ? Coin.toJSON(e) : undefined);
    } else {
      obj.denomCreationFee = [];
    }
    message.denomCreationGasConsume !== undefined
      && (obj.denomCreationGasConsume = Math.round(message.denomCreationGasConsume));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Params>, I>>(object: I): Params {
    const message = createBaseParams();
    message.denomCreationFee = object.denomCreationFee?.map((e) => Coin.fromPartial(e)) || [];
    message.denomCreationGasConsume = object.denomCreationGasConsume ?? 0;
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
