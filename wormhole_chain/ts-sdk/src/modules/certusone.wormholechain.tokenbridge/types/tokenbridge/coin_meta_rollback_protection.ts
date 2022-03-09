//@ts-nocheck
/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "certusone.wormholechain.tokenbridge";

export interface CoinMetaRollbackProtection {
  index: string;
  lastUpdateSequence: number;
}

const baseCoinMetaRollbackProtection: object = {
  index: "",
  lastUpdateSequence: 0,
};

export const CoinMetaRollbackProtection = {
  encode(
    message: CoinMetaRollbackProtection,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    if (message.lastUpdateSequence !== 0) {
      writer.uint32(16).uint64(message.lastUpdateSequence);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): CoinMetaRollbackProtection {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseCoinMetaRollbackProtection,
    } as CoinMetaRollbackProtection;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        case 2:
          message.lastUpdateSequence = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): CoinMetaRollbackProtection {
    const message = {
      ...baseCoinMetaRollbackProtection,
    } as CoinMetaRollbackProtection;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    if (
      object.lastUpdateSequence !== undefined &&
      object.lastUpdateSequence !== null
    ) {
      message.lastUpdateSequence = Number(object.lastUpdateSequence);
    } else {
      message.lastUpdateSequence = 0;
    }
    return message;
  },

  toJSON(message: CoinMetaRollbackProtection): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    message.lastUpdateSequence !== undefined &&
      (obj.lastUpdateSequence = message.lastUpdateSequence);
    return obj;
  },

  fromPartial(
    object: DeepPartial<CoinMetaRollbackProtection>
  ): CoinMetaRollbackProtection {
    const message = {
      ...baseCoinMetaRollbackProtection,
    } as CoinMetaRollbackProtection;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    if (
      object.lastUpdateSequence !== undefined &&
      object.lastUpdateSequence !== null
    ) {
      message.lastUpdateSequence = object.lastUpdateSequence;
    } else {
      message.lastUpdateSequence = 0;
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
