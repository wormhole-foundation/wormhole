//@ts-nocheck
/* eslint-disable */
import { GuardianSet } from "../wormhole/guardian_set";
import { Config } from "../wormhole/config";
import { ReplayProtection } from "../wormhole/replay_protection";
import { SequenceCounter } from "../wormhole/sequence_counter";
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "certusone.wormholechain.wormhole";

/** GenesisState defines the wormhole module's genesis state. */
export interface GenesisState {
  guardianSetList: GuardianSet[];
  guardianSetCount: number;
  config: Config | undefined;
  replayProtectionList: ReplayProtection[];
  /** this line is used by starport scaffolding # genesis/proto/state */
  sequenceCounterList: SequenceCounter[];
}

const baseGenesisState: object = { guardianSetCount: 0 };

export const GenesisState = {
  encode(message: GenesisState, writer: Writer = Writer.create()): Writer {
    for (const v of message.guardianSetList) {
      GuardianSet.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.guardianSetCount !== 0) {
      writer.uint32(16).uint32(message.guardianSetCount);
    }
    if (message.config !== undefined) {
      Config.encode(message.config, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.replayProtectionList) {
      ReplayProtection.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.sequenceCounterList) {
      SequenceCounter.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGenesisState } as GenesisState;
    message.guardianSetList = [];
    message.replayProtectionList = [];
    message.sequenceCounterList = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.guardianSetList.push(
            GuardianSet.decode(reader, reader.uint32())
          );
          break;
        case 2:
          message.guardianSetCount = reader.uint32();
          break;
        case 3:
          message.config = Config.decode(reader, reader.uint32());
          break;
        case 4:
          message.replayProtectionList.push(
            ReplayProtection.decode(reader, reader.uint32())
          );
          break;
        case 5:
          message.sequenceCounterList.push(
            SequenceCounter.decode(reader, reader.uint32())
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.guardianSetList = [];
    message.replayProtectionList = [];
    message.sequenceCounterList = [];
    if (
      object.guardianSetList !== undefined &&
      object.guardianSetList !== null
    ) {
      for (const e of object.guardianSetList) {
        message.guardianSetList.push(GuardianSet.fromJSON(e));
      }
    }
    if (
      object.guardianSetCount !== undefined &&
      object.guardianSetCount !== null
    ) {
      message.guardianSetCount = Number(object.guardianSetCount);
    } else {
      message.guardianSetCount = 0;
    }
    if (object.config !== undefined && object.config !== null) {
      message.config = Config.fromJSON(object.config);
    } else {
      message.config = undefined;
    }
    if (
      object.replayProtectionList !== undefined &&
      object.replayProtectionList !== null
    ) {
      for (const e of object.replayProtectionList) {
        message.replayProtectionList.push(ReplayProtection.fromJSON(e));
      }
    }
    if (
      object.sequenceCounterList !== undefined &&
      object.sequenceCounterList !== null
    ) {
      for (const e of object.sequenceCounterList) {
        message.sequenceCounterList.push(SequenceCounter.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    if (message.guardianSetList) {
      obj.guardianSetList = message.guardianSetList.map((e) =>
        e ? GuardianSet.toJSON(e) : undefined
      );
    } else {
      obj.guardianSetList = [];
    }
    message.guardianSetCount !== undefined &&
      (obj.guardianSetCount = message.guardianSetCount);
    message.config !== undefined &&
      (obj.config = message.config ? Config.toJSON(message.config) : undefined);
    if (message.replayProtectionList) {
      obj.replayProtectionList = message.replayProtectionList.map((e) =>
        e ? ReplayProtection.toJSON(e) : undefined
      );
    } else {
      obj.replayProtectionList = [];
    }
    if (message.sequenceCounterList) {
      obj.sequenceCounterList = message.sequenceCounterList.map((e) =>
        e ? SequenceCounter.toJSON(e) : undefined
      );
    } else {
      obj.sequenceCounterList = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = { ...baseGenesisState } as GenesisState;
    message.guardianSetList = [];
    message.replayProtectionList = [];
    message.sequenceCounterList = [];
    if (
      object.guardianSetList !== undefined &&
      object.guardianSetList !== null
    ) {
      for (const e of object.guardianSetList) {
        message.guardianSetList.push(GuardianSet.fromPartial(e));
      }
    }
    if (
      object.guardianSetCount !== undefined &&
      object.guardianSetCount !== null
    ) {
      message.guardianSetCount = object.guardianSetCount;
    } else {
      message.guardianSetCount = 0;
    }
    if (object.config !== undefined && object.config !== null) {
      message.config = Config.fromPartial(object.config);
    } else {
      message.config = undefined;
    }
    if (
      object.replayProtectionList !== undefined &&
      object.replayProtectionList !== null
    ) {
      for (const e of object.replayProtectionList) {
        message.replayProtectionList.push(ReplayProtection.fromPartial(e));
      }
    }
    if (
      object.sequenceCounterList !== undefined &&
      object.sequenceCounterList !== null
    ) {
      for (const e of object.sequenceCounterList) {
        message.sequenceCounterList.push(SequenceCounter.fromPartial(e));
      }
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
