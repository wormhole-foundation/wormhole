//@ts-nocheck
/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "wormhole_foundation.wormchain.wormhole";

export interface ConsensusGuardianSetIndex {
  index: number;
}

const baseConsensusGuardianSetIndex: object = { index: 0 };

export const ConsensusGuardianSetIndex = {
  encode(
    message: ConsensusGuardianSetIndex,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== 0) {
      writer.uint32(8).uint32(message.index);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): ConsensusGuardianSetIndex {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseConsensusGuardianSetIndex,
    } as ConsensusGuardianSetIndex;
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
    const message = {
      ...baseConsensusGuardianSetIndex,
    } as ConsensusGuardianSetIndex;
    if (object.index !== undefined && object.index !== null) {
      message.index = Number(object.index);
    } else {
      message.index = 0;
    }
    return message;
  },

  toJSON(message: ConsensusGuardianSetIndex): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<ConsensusGuardianSetIndex>
  ): ConsensusGuardianSetIndex {
    const message = {
      ...baseConsensusGuardianSetIndex,
    } as ConsensusGuardianSetIndex;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = 0;
    }
    return message;
  },
};

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
