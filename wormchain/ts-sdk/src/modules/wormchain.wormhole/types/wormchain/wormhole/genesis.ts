//@ts-nocheck
/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { Config } from "./config";
import { ConsensusGuardianSetIndex } from "./consensus_guardian_set_index";
import {
  GuardianSet,
  GuardianValidator,
  IbcComposabilityMwContract,
  ValidatorAllowedAddress,
  WasmInstantiateAllowedContractCodeId,
} from "./guardian";
import { ReplayProtection } from "./replay_protection";
import { SequenceCounter } from "./sequence_counter";

export const protobufPackage = "wormchain.wormhole";

/** GenesisState defines the wormhole module's genesis state. */
export interface GenesisState {
  guardianSetList: GuardianSet[];
  config: Config | undefined;
  replayProtectionList: ReplayProtection[];
  sequenceCounterList: SequenceCounter[];
  consensusGuardianSetIndex: ConsensusGuardianSetIndex | undefined;
  guardianValidatorList: GuardianValidator[];
  allowedAddresses: ValidatorAllowedAddress[];
  wasmInstantiateAllowlist: WasmInstantiateAllowedContractCodeId[];
  /** this line is used by starport scaffolding # genesis/proto/state */
  ibcComposabilityMwContract: IbcComposabilityMwContract | undefined;
}

function createBaseGenesisState(): GenesisState {
  return {
    guardianSetList: [],
    config: undefined,
    replayProtectionList: [],
    sequenceCounterList: [],
    consensusGuardianSetIndex: undefined,
    guardianValidatorList: [],
    allowedAddresses: [],
    wasmInstantiateAllowlist: [],
    ibcComposabilityMwContract: undefined,
  };
}

export const GenesisState = {
  encode(message: GenesisState, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.guardianSetList) {
      GuardianSet.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.config !== undefined) {
      Config.encode(message.config, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.replayProtectionList) {
      ReplayProtection.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.sequenceCounterList) {
      SequenceCounter.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    if (message.consensusGuardianSetIndex !== undefined) {
      ConsensusGuardianSetIndex.encode(message.consensusGuardianSetIndex, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.guardianValidatorList) {
      GuardianValidator.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.allowedAddresses) {
      ValidatorAllowedAddress.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    for (const v of message.wasmInstantiateAllowlist) {
      WasmInstantiateAllowedContractCodeId.encode(v!, writer.uint32(66).fork()).ldelim();
    }
    if (message.ibcComposabilityMwContract !== undefined) {
      IbcComposabilityMwContract.encode(message.ibcComposabilityMwContract, writer.uint32(74).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisState();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.guardianSetList.push(GuardianSet.decode(reader, reader.uint32()));
          break;
        case 2:
          message.config = Config.decode(reader, reader.uint32());
          break;
        case 3:
          message.replayProtectionList.push(ReplayProtection.decode(reader, reader.uint32()));
          break;
        case 4:
          message.sequenceCounterList.push(SequenceCounter.decode(reader, reader.uint32()));
          break;
        case 5:
          message.consensusGuardianSetIndex = ConsensusGuardianSetIndex.decode(reader, reader.uint32());
          break;
        case 6:
          message.guardianValidatorList.push(GuardianValidator.decode(reader, reader.uint32()));
          break;
        case 7:
          message.allowedAddresses.push(ValidatorAllowedAddress.decode(reader, reader.uint32()));
          break;
        case 8:
          message.wasmInstantiateAllowlist.push(WasmInstantiateAllowedContractCodeId.decode(reader, reader.uint32()));
          break;
        case 9:
          message.ibcComposabilityMwContract = IbcComposabilityMwContract.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GenesisState {
    return {
      guardianSetList: Array.isArray(object?.guardianSetList)
        ? object.guardianSetList.map((e: any) => GuardianSet.fromJSON(e))
        : [],
      config: isSet(object.config) ? Config.fromJSON(object.config) : undefined,
      replayProtectionList: Array.isArray(object?.replayProtectionList)
        ? object.replayProtectionList.map((e: any) => ReplayProtection.fromJSON(e))
        : [],
      sequenceCounterList: Array.isArray(object?.sequenceCounterList)
        ? object.sequenceCounterList.map((e: any) => SequenceCounter.fromJSON(e))
        : [],
      consensusGuardianSetIndex: isSet(object.consensusGuardianSetIndex)
        ? ConsensusGuardianSetIndex.fromJSON(object.consensusGuardianSetIndex)
        : undefined,
      guardianValidatorList: Array.isArray(object?.guardianValidatorList)
        ? object.guardianValidatorList.map((e: any) => GuardianValidator.fromJSON(e))
        : [],
      allowedAddresses: Array.isArray(object?.allowedAddresses)
        ? object.allowedAddresses.map((e: any) => ValidatorAllowedAddress.fromJSON(e))
        : [],
      wasmInstantiateAllowlist: Array.isArray(object?.wasmInstantiateAllowlist)
        ? object.wasmInstantiateAllowlist.map((e: any) => WasmInstantiateAllowedContractCodeId.fromJSON(e))
        : [],
      ibcComposabilityMwContract: isSet(object.ibcComposabilityMwContract)
        ? IbcComposabilityMwContract.fromJSON(object.ibcComposabilityMwContract)
        : undefined,
    };
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    if (message.guardianSetList) {
      obj.guardianSetList = message.guardianSetList.map((e) => e ? GuardianSet.toJSON(e) : undefined);
    } else {
      obj.guardianSetList = [];
    }
    message.config !== undefined && (obj.config = message.config ? Config.toJSON(message.config) : undefined);
    if (message.replayProtectionList) {
      obj.replayProtectionList = message.replayProtectionList.map((e) => e ? ReplayProtection.toJSON(e) : undefined);
    } else {
      obj.replayProtectionList = [];
    }
    if (message.sequenceCounterList) {
      obj.sequenceCounterList = message.sequenceCounterList.map((e) => e ? SequenceCounter.toJSON(e) : undefined);
    } else {
      obj.sequenceCounterList = [];
    }
    message.consensusGuardianSetIndex !== undefined
      && (obj.consensusGuardianSetIndex = message.consensusGuardianSetIndex
        ? ConsensusGuardianSetIndex.toJSON(message.consensusGuardianSetIndex)
        : undefined);
    if (message.guardianValidatorList) {
      obj.guardianValidatorList = message.guardianValidatorList.map((e) => e ? GuardianValidator.toJSON(e) : undefined);
    } else {
      obj.guardianValidatorList = [];
    }
    if (message.allowedAddresses) {
      obj.allowedAddresses = message.allowedAddresses.map((e) => e ? ValidatorAllowedAddress.toJSON(e) : undefined);
    } else {
      obj.allowedAddresses = [];
    }
    if (message.wasmInstantiateAllowlist) {
      obj.wasmInstantiateAllowlist = message.wasmInstantiateAllowlist.map((e) =>
        e ? WasmInstantiateAllowedContractCodeId.toJSON(e) : undefined
      );
    } else {
      obj.wasmInstantiateAllowlist = [];
    }
    message.ibcComposabilityMwContract !== undefined
      && (obj.ibcComposabilityMwContract = message.ibcComposabilityMwContract
        ? IbcComposabilityMwContract.toJSON(message.ibcComposabilityMwContract)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GenesisState>, I>>(object: I): GenesisState {
    const message = createBaseGenesisState();
    message.guardianSetList = object.guardianSetList?.map((e) => GuardianSet.fromPartial(e)) || [];
    message.config = (object.config !== undefined && object.config !== null)
      ? Config.fromPartial(object.config)
      : undefined;
    message.replayProtectionList = object.replayProtectionList?.map((e) => ReplayProtection.fromPartial(e)) || [];
    message.sequenceCounterList = object.sequenceCounterList?.map((e) => SequenceCounter.fromPartial(e)) || [];
    message.consensusGuardianSetIndex =
      (object.consensusGuardianSetIndex !== undefined && object.consensusGuardianSetIndex !== null)
        ? ConsensusGuardianSetIndex.fromPartial(object.consensusGuardianSetIndex)
        : undefined;
    message.guardianValidatorList = object.guardianValidatorList?.map((e) => GuardianValidator.fromPartial(e)) || [];
    message.allowedAddresses = object.allowedAddresses?.map((e) => ValidatorAllowedAddress.fromPartial(e)) || [];
    message.wasmInstantiateAllowlist =
      object.wasmInstantiateAllowlist?.map((e) => WasmInstantiateAllowedContractCodeId.fromPartial(e)) || [];
    message.ibcComposabilityMwContract =
      (object.ibcComposabilityMwContract !== undefined && object.ibcComposabilityMwContract !== null)
        ? IbcComposabilityMwContract.fromPartial(object.ibcComposabilityMwContract)
        : undefined;
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
