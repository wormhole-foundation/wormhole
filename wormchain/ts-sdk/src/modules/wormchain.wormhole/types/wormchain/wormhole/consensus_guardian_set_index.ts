//@ts-nocheck
/* eslint-disable */
import _m0 from "protobufjs/minimal";

export const protobufPackage = "wormchain.wormhole";

export interface ConsensusGuardianSetIndex {
  index: number;
}

function createBaseConsensusGuardianSetIndex(): ConsensusGuardianSetIndex {
  return { index: 0 };
}

export const ConsensusGuardianSetIndex = {
  encode(message: ConsensusGuardianSetIndex, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== 0) {
      writer.uint32(8).uint32(message.index);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ConsensusGuardianSetIndex {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseConsensusGuardianSetIndex();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ConsensusGuardianSetIndex {
    return { index: isSet(object.index) ? Number(object.index) : 0 };
  },

  toJSON(message: ConsensusGuardianSetIndex): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = Math.round(message.index));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ConsensusGuardianSetIndex>, I>>(object: I): ConsensusGuardianSetIndex {
    const message = createBaseConsensusGuardianSetIndex();
    message.index = object.index ?? 0;
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
