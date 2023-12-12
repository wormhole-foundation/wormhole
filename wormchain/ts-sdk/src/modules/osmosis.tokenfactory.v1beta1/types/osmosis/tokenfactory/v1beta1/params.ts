//@ts-nocheck
/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Coin } from "../../../cosmos/base/v1beta1/coin";

export const protobufPackage = "osmosis.tokenfactory.v1beta1";

/** Params defines the parameters for the tokenfactory module. */
export interface Params {
  denom_creation_fee: Coin[];
  /**
   * if denom_creation_fee is an empty array, then this field is used to add more gas consumption
   * to the base cost.
   * https://github.com/CosmWasm/token-factory/issues/11
   */
  denom_creation_gas_consume: number;
}

const baseParams: object = { denom_creation_gas_consume: 0 };

export const Params = {
  encode(message: Params, writer: Writer = Writer.create()): Writer {
    for (const v of message.denom_creation_fee) {
      Coin.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.denom_creation_gas_consume !== 0) {
      writer.uint32(16).uint64(message.denom_creation_gas_consume);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): Params {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseParams } as Params;
    message.denom_creation_fee = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.denom_creation_fee.push(Coin.decode(reader, reader.uint32()));
          break;
        case 2:
          message.denom_creation_gas_consume = longToNumber(
            reader.uint64() as Long
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Params {
    const message = { ...baseParams } as Params;
    message.denom_creation_fee = [];
    if (
      object.denom_creation_fee !== undefined &&
      object.denom_creation_fee !== null
    ) {
      for (const e of object.denom_creation_fee) {
        message.denom_creation_fee.push(Coin.fromJSON(e));
      }
    }
    if (
      object.denom_creation_gas_consume !== undefined &&
      object.denom_creation_gas_consume !== null
    ) {
      message.denom_creation_gas_consume = Number(
        object.denom_creation_gas_consume
      );
    } else {
      message.denom_creation_gas_consume = 0;
    }
    return message;
  },

  toJSON(message: Params): unknown {
    const obj: any = {};
    if (message.denom_creation_fee) {
      obj.denom_creation_fee = message.denom_creation_fee.map((e) =>
        e ? Coin.toJSON(e) : undefined
      );
    } else {
      obj.denom_creation_fee = [];
    }
    message.denom_creation_gas_consume !== undefined &&
      (obj.denom_creation_gas_consume = message.denom_creation_gas_consume);
    return obj;
  },

  fromPartial(object: DeepPartial<Params>): Params {
    const message = { ...baseParams } as Params;
    message.denom_creation_fee = [];
    if (
      object.denom_creation_fee !== undefined &&
      object.denom_creation_fee !== null
    ) {
      for (const e of object.denom_creation_fee) {
        message.denom_creation_fee.push(Coin.fromPartial(e));
      }
    }
    if (
      object.denom_creation_gas_consume !== undefined &&
      object.denom_creation_gas_consume !== null
    ) {
      message.denom_creation_gas_consume = object.denom_creation_gas_consume;
    } else {
      message.denom_creation_gas_consume = 0;
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
