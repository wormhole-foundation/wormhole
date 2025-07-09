//@ts-nocheck
/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { PageRequest, PageResponse } from "../../cosmos/base/query/v1beta1/pagination";
import { Config } from "./config";
import { ConsensusGuardianSetIndex } from "./consensus_guardian_set_index";
import {
  GuardianSet,
  GuardianValidator,
  ValidatorAllowedAddress,
  WasmInstantiateAllowedContractCodeId,
} from "./guardian";
import { ReplayProtection } from "./replay_protection";
import { SequenceCounter } from "./sequence_counter";

export const protobufPackage = "wormchain.wormhole";

export interface QueryAllValidatorAllowlist {
  pagination: PageRequest | undefined;
}

/** all allowlisted entries by all validators */
export interface QueryAllValidatorAllowlistResponse {
  allowlist: ValidatorAllowedAddress[];
  pagination: PageResponse | undefined;
}

export interface QueryValidatorAllowlist {
  validatorAddress: string;
  pagination: PageRequest | undefined;
}

/** all allowlisted entries by a specific validator */
export interface QueryValidatorAllowlistResponse {
  validatorAddress: string;
  allowlist: ValidatorAllowedAddress[];
  pagination: PageResponse | undefined;
}

export interface QueryGetGuardianSetRequest {
  index: number;
}

export interface QueryGetGuardianSetResponse {
  GuardianSet: GuardianSet | undefined;
}

export interface QueryAllGuardianSetRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllGuardianSetResponse {
  GuardianSet: GuardianSet[];
  pagination: PageResponse | undefined;
}

export interface QueryGetConfigRequest {
}

export interface QueryGetConfigResponse {
  Config: Config | undefined;
}

export interface QueryGetReplayProtectionRequest {
  index: string;
}

export interface QueryGetReplayProtectionResponse {
  replayProtection: ReplayProtection | undefined;
}

export interface QueryAllReplayProtectionRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllReplayProtectionResponse {
  replayProtection: ReplayProtection[];
  pagination: PageResponse | undefined;
}

export interface QueryGetSequenceCounterRequest {
  index: string;
}

export interface QueryGetSequenceCounterResponse {
  sequenceCounter: SequenceCounter | undefined;
}

export interface QueryAllSequenceCounterRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllSequenceCounterResponse {
  sequenceCounter: SequenceCounter[];
  pagination: PageResponse | undefined;
}

export interface QueryGetConsensusGuardianSetIndexRequest {
}

export interface QueryGetConsensusGuardianSetIndexResponse {
  ConsensusGuardianSetIndex: ConsensusGuardianSetIndex | undefined;
}

export interface QueryGetGuardianValidatorRequest {
  guardianKey: Uint8Array;
}

export interface QueryGetGuardianValidatorResponse {
  guardianValidator: GuardianValidator | undefined;
}

export interface QueryAllGuardianValidatorRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllGuardianValidatorResponse {
  guardianValidator: GuardianValidator[];
  pagination: PageResponse | undefined;
}

export interface QueryLatestGuardianSetIndexRequest {
}

export interface QueryLatestGuardianSetIndexResponse {
  latestGuardianSetIndex: number;
}

export interface QueryIbcComposabilityMwContractRequest {
}

export interface QueryIbcComposabilityMwContractResponse {
  contractAddress: string;
}

export interface QueryAllWasmInstantiateAllowlist {
  pagination: PageRequest | undefined;
}

/** all allowlisted entries by all validators */
export interface QueryAllWasmInstantiateAllowlistResponse {
  allowlist: WasmInstantiateAllowedContractCodeId[];
  pagination: PageResponse | undefined;
}

function createBaseQueryAllValidatorAllowlist(): QueryAllValidatorAllowlist {
  return { pagination: undefined };
}

export const QueryAllValidatorAllowlist = {
  encode(message: QueryAllValidatorAllowlist, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllValidatorAllowlist {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllValidatorAllowlist();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllValidatorAllowlist {
    return { pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined };
  },

  toJSON(message: QueryAllValidatorAllowlist): unknown {
    const obj: any = {};
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllValidatorAllowlist>, I>>(object: I): QueryAllValidatorAllowlist {
    const message = createBaseQueryAllValidatorAllowlist();
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryAllValidatorAllowlistResponse(): QueryAllValidatorAllowlistResponse {
  return { allowlist: [], pagination: undefined };
}

export const QueryAllValidatorAllowlistResponse = {
  encode(message: QueryAllValidatorAllowlistResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.allowlist) {
      ValidatorAllowedAddress.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllValidatorAllowlistResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllValidatorAllowlistResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.allowlist.push(ValidatorAllowedAddress.decode(reader, reader.uint32()));
          break;
        case 2:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllValidatorAllowlistResponse {
    return {
      allowlist: Array.isArray(object?.allowlist)
        ? object.allowlist.map((e: any) => ValidatorAllowedAddress.fromJSON(e))
        : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryAllValidatorAllowlistResponse): unknown {
    const obj: any = {};
    if (message.allowlist) {
      obj.allowlist = message.allowlist.map((e) => e ? ValidatorAllowedAddress.toJSON(e) : undefined);
    } else {
      obj.allowlist = [];
    }
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllValidatorAllowlistResponse>, I>>(
    object: I,
  ): QueryAllValidatorAllowlistResponse {
    const message = createBaseQueryAllValidatorAllowlistResponse();
    message.allowlist = object.allowlist?.map((e) => ValidatorAllowedAddress.fromPartial(e)) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryValidatorAllowlist(): QueryValidatorAllowlist {
  return { validatorAddress: "", pagination: undefined };
}

export const QueryValidatorAllowlist = {
  encode(message: QueryValidatorAllowlist, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.validatorAddress !== "") {
      writer.uint32(10).string(message.validatorAddress);
    }
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryValidatorAllowlist {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryValidatorAllowlist();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validatorAddress = reader.string();
          break;
        case 2:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryValidatorAllowlist {
    return {
      validatorAddress: isSet(object.validatorAddress) ? String(object.validatorAddress) : "",
      pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryValidatorAllowlist): unknown {
    const obj: any = {};
    message.validatorAddress !== undefined && (obj.validatorAddress = message.validatorAddress);
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryValidatorAllowlist>, I>>(object: I): QueryValidatorAllowlist {
    const message = createBaseQueryValidatorAllowlist();
    message.validatorAddress = object.validatorAddress ?? "";
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryValidatorAllowlistResponse(): QueryValidatorAllowlistResponse {
  return { validatorAddress: "", allowlist: [], pagination: undefined };
}

export const QueryValidatorAllowlistResponse = {
  encode(message: QueryValidatorAllowlistResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.validatorAddress !== "") {
      writer.uint32(10).string(message.validatorAddress);
    }
    for (const v of message.allowlist) {
      ValidatorAllowedAddress.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryValidatorAllowlistResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryValidatorAllowlistResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validatorAddress = reader.string();
          break;
        case 2:
          message.allowlist.push(ValidatorAllowedAddress.decode(reader, reader.uint32()));
          break;
        case 3:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryValidatorAllowlistResponse {
    return {
      validatorAddress: isSet(object.validatorAddress) ? String(object.validatorAddress) : "",
      allowlist: Array.isArray(object?.allowlist)
        ? object.allowlist.map((e: any) => ValidatorAllowedAddress.fromJSON(e))
        : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryValidatorAllowlistResponse): unknown {
    const obj: any = {};
    message.validatorAddress !== undefined && (obj.validatorAddress = message.validatorAddress);
    if (message.allowlist) {
      obj.allowlist = message.allowlist.map((e) => e ? ValidatorAllowedAddress.toJSON(e) : undefined);
    } else {
      obj.allowlist = [];
    }
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryValidatorAllowlistResponse>, I>>(
    object: I,
  ): QueryValidatorAllowlistResponse {
    const message = createBaseQueryValidatorAllowlistResponse();
    message.validatorAddress = object.validatorAddress ?? "";
    message.allowlist = object.allowlist?.map((e) => ValidatorAllowedAddress.fromPartial(e)) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryGetGuardianSetRequest(): QueryGetGuardianSetRequest {
  return { index: 0 };
}

export const QueryGetGuardianSetRequest = {
  encode(message: QueryGetGuardianSetRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== 0) {
      writer.uint32(8).uint32(message.index);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetGuardianSetRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetGuardianSetRequest();
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

  fromJSON(object: any): QueryGetGuardianSetRequest {
    return { index: isSet(object.index) ? Number(object.index) : 0 };
  },

  toJSON(message: QueryGetGuardianSetRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = Math.round(message.index));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetGuardianSetRequest>, I>>(object: I): QueryGetGuardianSetRequest {
    const message = createBaseQueryGetGuardianSetRequest();
    message.index = object.index ?? 0;
    return message;
  },
};

function createBaseQueryGetGuardianSetResponse(): QueryGetGuardianSetResponse {
  return { GuardianSet: undefined };
}

export const QueryGetGuardianSetResponse = {
  encode(message: QueryGetGuardianSetResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.GuardianSet !== undefined) {
      GuardianSet.encode(message.GuardianSet, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetGuardianSetResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetGuardianSetResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.GuardianSet = GuardianSet.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetGuardianSetResponse {
    return { GuardianSet: isSet(object.GuardianSet) ? GuardianSet.fromJSON(object.GuardianSet) : undefined };
  },

  toJSON(message: QueryGetGuardianSetResponse): unknown {
    const obj: any = {};
    message.GuardianSet !== undefined
      && (obj.GuardianSet = message.GuardianSet ? GuardianSet.toJSON(message.GuardianSet) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetGuardianSetResponse>, I>>(object: I): QueryGetGuardianSetResponse {
    const message = createBaseQueryGetGuardianSetResponse();
    message.GuardianSet = (object.GuardianSet !== undefined && object.GuardianSet !== null)
      ? GuardianSet.fromPartial(object.GuardianSet)
      : undefined;
    return message;
  },
};

function createBaseQueryAllGuardianSetRequest(): QueryAllGuardianSetRequest {
  return { pagination: undefined };
}

export const QueryAllGuardianSetRequest = {
  encode(message: QueryAllGuardianSetRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllGuardianSetRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllGuardianSetRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllGuardianSetRequest {
    return { pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined };
  },

  toJSON(message: QueryAllGuardianSetRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllGuardianSetRequest>, I>>(object: I): QueryAllGuardianSetRequest {
    const message = createBaseQueryAllGuardianSetRequest();
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryAllGuardianSetResponse(): QueryAllGuardianSetResponse {
  return { GuardianSet: [], pagination: undefined };
}

export const QueryAllGuardianSetResponse = {
  encode(message: QueryAllGuardianSetResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.GuardianSet) {
      GuardianSet.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllGuardianSetResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllGuardianSetResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.GuardianSet.push(GuardianSet.decode(reader, reader.uint32()));
          break;
        case 2:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllGuardianSetResponse {
    return {
      GuardianSet: Array.isArray(object?.GuardianSet)
        ? object.GuardianSet.map((e: any) => GuardianSet.fromJSON(e))
        : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryAllGuardianSetResponse): unknown {
    const obj: any = {};
    if (message.GuardianSet) {
      obj.GuardianSet = message.GuardianSet.map((e) => e ? GuardianSet.toJSON(e) : undefined);
    } else {
      obj.GuardianSet = [];
    }
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllGuardianSetResponse>, I>>(object: I): QueryAllGuardianSetResponse {
    const message = createBaseQueryAllGuardianSetResponse();
    message.GuardianSet = object.GuardianSet?.map((e) => GuardianSet.fromPartial(e)) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryGetConfigRequest(): QueryGetConfigRequest {
  return {};
}

export const QueryGetConfigRequest = {
  encode(_: QueryGetConfigRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetConfigRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetConfigRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): QueryGetConfigRequest {
    return {};
  },

  toJSON(_: QueryGetConfigRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetConfigRequest>, I>>(_: I): QueryGetConfigRequest {
    const message = createBaseQueryGetConfigRequest();
    return message;
  },
};

function createBaseQueryGetConfigResponse(): QueryGetConfigResponse {
  return { Config: undefined };
}

export const QueryGetConfigResponse = {
  encode(message: QueryGetConfigResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.Config !== undefined) {
      Config.encode(message.Config, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetConfigResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetConfigResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.Config = Config.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetConfigResponse {
    return { Config: isSet(object.Config) ? Config.fromJSON(object.Config) : undefined };
  },

  toJSON(message: QueryGetConfigResponse): unknown {
    const obj: any = {};
    message.Config !== undefined && (obj.Config = message.Config ? Config.toJSON(message.Config) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetConfigResponse>, I>>(object: I): QueryGetConfigResponse {
    const message = createBaseQueryGetConfigResponse();
    message.Config = (object.Config !== undefined && object.Config !== null)
      ? Config.fromPartial(object.Config)
      : undefined;
    return message;
  },
};

function createBaseQueryGetReplayProtectionRequest(): QueryGetReplayProtectionRequest {
  return { index: "" };
}

export const QueryGetReplayProtectionRequest = {
  encode(message: QueryGetReplayProtectionRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetReplayProtectionRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetReplayProtectionRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetReplayProtectionRequest {
    return { index: isSet(object.index) ? String(object.index) : "" };
  },

  toJSON(message: QueryGetReplayProtectionRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetReplayProtectionRequest>, I>>(
    object: I,
  ): QueryGetReplayProtectionRequest {
    const message = createBaseQueryGetReplayProtectionRequest();
    message.index = object.index ?? "";
    return message;
  },
};

function createBaseQueryGetReplayProtectionResponse(): QueryGetReplayProtectionResponse {
  return { replayProtection: undefined };
}

export const QueryGetReplayProtectionResponse = {
  encode(message: QueryGetReplayProtectionResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.replayProtection !== undefined) {
      ReplayProtection.encode(message.replayProtection, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetReplayProtectionResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetReplayProtectionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.replayProtection = ReplayProtection.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetReplayProtectionResponse {
    return {
      replayProtection: isSet(object.replayProtection) ? ReplayProtection.fromJSON(object.replayProtection) : undefined,
    };
  },

  toJSON(message: QueryGetReplayProtectionResponse): unknown {
    const obj: any = {};
    message.replayProtection !== undefined && (obj.replayProtection = message.replayProtection
      ? ReplayProtection.toJSON(message.replayProtection)
      : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetReplayProtectionResponse>, I>>(
    object: I,
  ): QueryGetReplayProtectionResponse {
    const message = createBaseQueryGetReplayProtectionResponse();
    message.replayProtection = (object.replayProtection !== undefined && object.replayProtection !== null)
      ? ReplayProtection.fromPartial(object.replayProtection)
      : undefined;
    return message;
  },
};

function createBaseQueryAllReplayProtectionRequest(): QueryAllReplayProtectionRequest {
  return { pagination: undefined };
}

export const QueryAllReplayProtectionRequest = {
  encode(message: QueryAllReplayProtectionRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllReplayProtectionRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllReplayProtectionRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllReplayProtectionRequest {
    return { pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined };
  },

  toJSON(message: QueryAllReplayProtectionRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllReplayProtectionRequest>, I>>(
    object: I,
  ): QueryAllReplayProtectionRequest {
    const message = createBaseQueryAllReplayProtectionRequest();
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryAllReplayProtectionResponse(): QueryAllReplayProtectionResponse {
  return { replayProtection: [], pagination: undefined };
}

export const QueryAllReplayProtectionResponse = {
  encode(message: QueryAllReplayProtectionResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.replayProtection) {
      ReplayProtection.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllReplayProtectionResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllReplayProtectionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.replayProtection.push(ReplayProtection.decode(reader, reader.uint32()));
          break;
        case 2:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllReplayProtectionResponse {
    return {
      replayProtection: Array.isArray(object?.replayProtection)
        ? object.replayProtection.map((e: any) => ReplayProtection.fromJSON(e))
        : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryAllReplayProtectionResponse): unknown {
    const obj: any = {};
    if (message.replayProtection) {
      obj.replayProtection = message.replayProtection.map((e) => e ? ReplayProtection.toJSON(e) : undefined);
    } else {
      obj.replayProtection = [];
    }
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllReplayProtectionResponse>, I>>(
    object: I,
  ): QueryAllReplayProtectionResponse {
    const message = createBaseQueryAllReplayProtectionResponse();
    message.replayProtection = object.replayProtection?.map((e) => ReplayProtection.fromPartial(e)) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryGetSequenceCounterRequest(): QueryGetSequenceCounterRequest {
  return { index: "" };
}

export const QueryGetSequenceCounterRequest = {
  encode(message: QueryGetSequenceCounterRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetSequenceCounterRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetSequenceCounterRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetSequenceCounterRequest {
    return { index: isSet(object.index) ? String(object.index) : "" };
  },

  toJSON(message: QueryGetSequenceCounterRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetSequenceCounterRequest>, I>>(
    object: I,
  ): QueryGetSequenceCounterRequest {
    const message = createBaseQueryGetSequenceCounterRequest();
    message.index = object.index ?? "";
    return message;
  },
};

function createBaseQueryGetSequenceCounterResponse(): QueryGetSequenceCounterResponse {
  return { sequenceCounter: undefined };
}

export const QueryGetSequenceCounterResponse = {
  encode(message: QueryGetSequenceCounterResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sequenceCounter !== undefined) {
      SequenceCounter.encode(message.sequenceCounter, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetSequenceCounterResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetSequenceCounterResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sequenceCounter = SequenceCounter.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetSequenceCounterResponse {
    return {
      sequenceCounter: isSet(object.sequenceCounter) ? SequenceCounter.fromJSON(object.sequenceCounter) : undefined,
    };
  },

  toJSON(message: QueryGetSequenceCounterResponse): unknown {
    const obj: any = {};
    message.sequenceCounter !== undefined
      && (obj.sequenceCounter = message.sequenceCounter ? SequenceCounter.toJSON(message.sequenceCounter) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetSequenceCounterResponse>, I>>(
    object: I,
  ): QueryGetSequenceCounterResponse {
    const message = createBaseQueryGetSequenceCounterResponse();
    message.sequenceCounter = (object.sequenceCounter !== undefined && object.sequenceCounter !== null)
      ? SequenceCounter.fromPartial(object.sequenceCounter)
      : undefined;
    return message;
  },
};

function createBaseQueryAllSequenceCounterRequest(): QueryAllSequenceCounterRequest {
  return { pagination: undefined };
}

export const QueryAllSequenceCounterRequest = {
  encode(message: QueryAllSequenceCounterRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllSequenceCounterRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllSequenceCounterRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllSequenceCounterRequest {
    return { pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined };
  },

  toJSON(message: QueryAllSequenceCounterRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllSequenceCounterRequest>, I>>(
    object: I,
  ): QueryAllSequenceCounterRequest {
    const message = createBaseQueryAllSequenceCounterRequest();
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryAllSequenceCounterResponse(): QueryAllSequenceCounterResponse {
  return { sequenceCounter: [], pagination: undefined };
}

export const QueryAllSequenceCounterResponse = {
  encode(message: QueryAllSequenceCounterResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.sequenceCounter) {
      SequenceCounter.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllSequenceCounterResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllSequenceCounterResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sequenceCounter.push(SequenceCounter.decode(reader, reader.uint32()));
          break;
        case 2:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllSequenceCounterResponse {
    return {
      sequenceCounter: Array.isArray(object?.sequenceCounter)
        ? object.sequenceCounter.map((e: any) => SequenceCounter.fromJSON(e))
        : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryAllSequenceCounterResponse): unknown {
    const obj: any = {};
    if (message.sequenceCounter) {
      obj.sequenceCounter = message.sequenceCounter.map((e) => e ? SequenceCounter.toJSON(e) : undefined);
    } else {
      obj.sequenceCounter = [];
    }
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllSequenceCounterResponse>, I>>(
    object: I,
  ): QueryAllSequenceCounterResponse {
    const message = createBaseQueryAllSequenceCounterResponse();
    message.sequenceCounter = object.sequenceCounter?.map((e) => SequenceCounter.fromPartial(e)) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryGetConsensusGuardianSetIndexRequest(): QueryGetConsensusGuardianSetIndexRequest {
  return {};
}

export const QueryGetConsensusGuardianSetIndexRequest = {
  encode(_: QueryGetConsensusGuardianSetIndexRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetConsensusGuardianSetIndexRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetConsensusGuardianSetIndexRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): QueryGetConsensusGuardianSetIndexRequest {
    return {};
  },

  toJSON(_: QueryGetConsensusGuardianSetIndexRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetConsensusGuardianSetIndexRequest>, I>>(
    _: I,
  ): QueryGetConsensusGuardianSetIndexRequest {
    const message = createBaseQueryGetConsensusGuardianSetIndexRequest();
    return message;
  },
};

function createBaseQueryGetConsensusGuardianSetIndexResponse(): QueryGetConsensusGuardianSetIndexResponse {
  return { ConsensusGuardianSetIndex: undefined };
}

export const QueryGetConsensusGuardianSetIndexResponse = {
  encode(message: QueryGetConsensusGuardianSetIndexResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.ConsensusGuardianSetIndex !== undefined) {
      ConsensusGuardianSetIndex.encode(message.ConsensusGuardianSetIndex, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetConsensusGuardianSetIndexResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetConsensusGuardianSetIndexResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.ConsensusGuardianSetIndex = ConsensusGuardianSetIndex.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetConsensusGuardianSetIndexResponse {
    return {
      ConsensusGuardianSetIndex: isSet(object.ConsensusGuardianSetIndex)
        ? ConsensusGuardianSetIndex.fromJSON(object.ConsensusGuardianSetIndex)
        : undefined,
    };
  },

  toJSON(message: QueryGetConsensusGuardianSetIndexResponse): unknown {
    const obj: any = {};
    message.ConsensusGuardianSetIndex !== undefined
      && (obj.ConsensusGuardianSetIndex = message.ConsensusGuardianSetIndex
        ? ConsensusGuardianSetIndex.toJSON(message.ConsensusGuardianSetIndex)
        : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetConsensusGuardianSetIndexResponse>, I>>(
    object: I,
  ): QueryGetConsensusGuardianSetIndexResponse {
    const message = createBaseQueryGetConsensusGuardianSetIndexResponse();
    message.ConsensusGuardianSetIndex =
      (object.ConsensusGuardianSetIndex !== undefined && object.ConsensusGuardianSetIndex !== null)
        ? ConsensusGuardianSetIndex.fromPartial(object.ConsensusGuardianSetIndex)
        : undefined;
    return message;
  },
};

function createBaseQueryGetGuardianValidatorRequest(): QueryGetGuardianValidatorRequest {
  return { guardianKey: new Uint8Array() };
}

export const QueryGetGuardianValidatorRequest = {
  encode(message: QueryGetGuardianValidatorRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.guardianKey.length !== 0) {
      writer.uint32(10).bytes(message.guardianKey);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetGuardianValidatorRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetGuardianValidatorRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.guardianKey = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetGuardianValidatorRequest {
    return { guardianKey: isSet(object.guardianKey) ? bytesFromBase64(object.guardianKey) : new Uint8Array() };
  },

  toJSON(message: QueryGetGuardianValidatorRequest): unknown {
    const obj: any = {};
    message.guardianKey !== undefined
      && (obj.guardianKey = base64FromBytes(
        message.guardianKey !== undefined ? message.guardianKey : new Uint8Array(),
      ));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetGuardianValidatorRequest>, I>>(
    object: I,
  ): QueryGetGuardianValidatorRequest {
    const message = createBaseQueryGetGuardianValidatorRequest();
    message.guardianKey = object.guardianKey ?? new Uint8Array();
    return message;
  },
};

function createBaseQueryGetGuardianValidatorResponse(): QueryGetGuardianValidatorResponse {
  return { guardianValidator: undefined };
}

export const QueryGetGuardianValidatorResponse = {
  encode(message: QueryGetGuardianValidatorResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.guardianValidator !== undefined) {
      GuardianValidator.encode(message.guardianValidator, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryGetGuardianValidatorResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryGetGuardianValidatorResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.guardianValidator = GuardianValidator.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetGuardianValidatorResponse {
    return {
      guardianValidator: isSet(object.guardianValidator)
        ? GuardianValidator.fromJSON(object.guardianValidator)
        : undefined,
    };
  },

  toJSON(message: QueryGetGuardianValidatorResponse): unknown {
    const obj: any = {};
    message.guardianValidator !== undefined && (obj.guardianValidator = message.guardianValidator
      ? GuardianValidator.toJSON(message.guardianValidator)
      : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryGetGuardianValidatorResponse>, I>>(
    object: I,
  ): QueryGetGuardianValidatorResponse {
    const message = createBaseQueryGetGuardianValidatorResponse();
    message.guardianValidator = (object.guardianValidator !== undefined && object.guardianValidator !== null)
      ? GuardianValidator.fromPartial(object.guardianValidator)
      : undefined;
    return message;
  },
};

function createBaseQueryAllGuardianValidatorRequest(): QueryAllGuardianValidatorRequest {
  return { pagination: undefined };
}

export const QueryAllGuardianValidatorRequest = {
  encode(message: QueryAllGuardianValidatorRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllGuardianValidatorRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllGuardianValidatorRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllGuardianValidatorRequest {
    return { pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined };
  },

  toJSON(message: QueryAllGuardianValidatorRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllGuardianValidatorRequest>, I>>(
    object: I,
  ): QueryAllGuardianValidatorRequest {
    const message = createBaseQueryAllGuardianValidatorRequest();
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryAllGuardianValidatorResponse(): QueryAllGuardianValidatorResponse {
  return { guardianValidator: [], pagination: undefined };
}

export const QueryAllGuardianValidatorResponse = {
  encode(message: QueryAllGuardianValidatorResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.guardianValidator) {
      GuardianValidator.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllGuardianValidatorResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllGuardianValidatorResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.guardianValidator.push(GuardianValidator.decode(reader, reader.uint32()));
          break;
        case 2:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllGuardianValidatorResponse {
    return {
      guardianValidator: Array.isArray(object?.guardianValidator)
        ? object.guardianValidator.map((e: any) => GuardianValidator.fromJSON(e))
        : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryAllGuardianValidatorResponse): unknown {
    const obj: any = {};
    if (message.guardianValidator) {
      obj.guardianValidator = message.guardianValidator.map((e) => e ? GuardianValidator.toJSON(e) : undefined);
    } else {
      obj.guardianValidator = [];
    }
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllGuardianValidatorResponse>, I>>(
    object: I,
  ): QueryAllGuardianValidatorResponse {
    const message = createBaseQueryAllGuardianValidatorResponse();
    message.guardianValidator = object.guardianValidator?.map((e) => GuardianValidator.fromPartial(e)) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryLatestGuardianSetIndexRequest(): QueryLatestGuardianSetIndexRequest {
  return {};
}

export const QueryLatestGuardianSetIndexRequest = {
  encode(_: QueryLatestGuardianSetIndexRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryLatestGuardianSetIndexRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryLatestGuardianSetIndexRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): QueryLatestGuardianSetIndexRequest {
    return {};
  },

  toJSON(_: QueryLatestGuardianSetIndexRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryLatestGuardianSetIndexRequest>, I>>(
    _: I,
  ): QueryLatestGuardianSetIndexRequest {
    const message = createBaseQueryLatestGuardianSetIndexRequest();
    return message;
  },
};

function createBaseQueryLatestGuardianSetIndexResponse(): QueryLatestGuardianSetIndexResponse {
  return { latestGuardianSetIndex: 0 };
}

export const QueryLatestGuardianSetIndexResponse = {
  encode(message: QueryLatestGuardianSetIndexResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.latestGuardianSetIndex !== 0) {
      writer.uint32(8).uint32(message.latestGuardianSetIndex);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryLatestGuardianSetIndexResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryLatestGuardianSetIndexResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.latestGuardianSetIndex = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryLatestGuardianSetIndexResponse {
    return { latestGuardianSetIndex: isSet(object.latestGuardianSetIndex) ? Number(object.latestGuardianSetIndex) : 0 };
  },

  toJSON(message: QueryLatestGuardianSetIndexResponse): unknown {
    const obj: any = {};
    message.latestGuardianSetIndex !== undefined
      && (obj.latestGuardianSetIndex = Math.round(message.latestGuardianSetIndex));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryLatestGuardianSetIndexResponse>, I>>(
    object: I,
  ): QueryLatestGuardianSetIndexResponse {
    const message = createBaseQueryLatestGuardianSetIndexResponse();
    message.latestGuardianSetIndex = object.latestGuardianSetIndex ?? 0;
    return message;
  },
};

function createBaseQueryIbcComposabilityMwContractRequest(): QueryIbcComposabilityMwContractRequest {
  return {};
}

export const QueryIbcComposabilityMwContractRequest = {
  encode(_: QueryIbcComposabilityMwContractRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryIbcComposabilityMwContractRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryIbcComposabilityMwContractRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): QueryIbcComposabilityMwContractRequest {
    return {};
  },

  toJSON(_: QueryIbcComposabilityMwContractRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryIbcComposabilityMwContractRequest>, I>>(
    _: I,
  ): QueryIbcComposabilityMwContractRequest {
    const message = createBaseQueryIbcComposabilityMwContractRequest();
    return message;
  },
};

function createBaseQueryIbcComposabilityMwContractResponse(): QueryIbcComposabilityMwContractResponse {
  return { contractAddress: "" };
}

export const QueryIbcComposabilityMwContractResponse = {
  encode(message: QueryIbcComposabilityMwContractResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.contractAddress !== "") {
      writer.uint32(10).string(message.contractAddress);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryIbcComposabilityMwContractResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryIbcComposabilityMwContractResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.contractAddress = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryIbcComposabilityMwContractResponse {
    return { contractAddress: isSet(object.contractAddress) ? String(object.contractAddress) : "" };
  },

  toJSON(message: QueryIbcComposabilityMwContractResponse): unknown {
    const obj: any = {};
    message.contractAddress !== undefined && (obj.contractAddress = message.contractAddress);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryIbcComposabilityMwContractResponse>, I>>(
    object: I,
  ): QueryIbcComposabilityMwContractResponse {
    const message = createBaseQueryIbcComposabilityMwContractResponse();
    message.contractAddress = object.contractAddress ?? "";
    return message;
  },
};

function createBaseQueryAllWasmInstantiateAllowlist(): QueryAllWasmInstantiateAllowlist {
  return { pagination: undefined };
}

export const QueryAllWasmInstantiateAllowlist = {
  encode(message: QueryAllWasmInstantiateAllowlist, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllWasmInstantiateAllowlist {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllWasmInstantiateAllowlist();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllWasmInstantiateAllowlist {
    return { pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined };
  },

  toJSON(message: QueryAllWasmInstantiateAllowlist): unknown {
    const obj: any = {};
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllWasmInstantiateAllowlist>, I>>(
    object: I,
  ): QueryAllWasmInstantiateAllowlist {
    const message = createBaseQueryAllWasmInstantiateAllowlist();
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseQueryAllWasmInstantiateAllowlistResponse(): QueryAllWasmInstantiateAllowlistResponse {
  return { allowlist: [], pagination: undefined };
}

export const QueryAllWasmInstantiateAllowlistResponse = {
  encode(message: QueryAllWasmInstantiateAllowlistResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.allowlist) {
      WasmInstantiateAllowedContractCodeId.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryAllWasmInstantiateAllowlistResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryAllWasmInstantiateAllowlistResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.allowlist.push(WasmInstantiateAllowedContractCodeId.decode(reader, reader.uint32()));
          break;
        case 2:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryAllWasmInstantiateAllowlistResponse {
    return {
      allowlist: Array.isArray(object?.allowlist)
        ? object.allowlist.map((e: any) => WasmInstantiateAllowedContractCodeId.fromJSON(e))
        : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: QueryAllWasmInstantiateAllowlistResponse): unknown {
    const obj: any = {};
    if (message.allowlist) {
      obj.allowlist = message.allowlist.map((e) => e ? WasmInstantiateAllowedContractCodeId.toJSON(e) : undefined);
    } else {
      obj.allowlist = [];
    }
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<QueryAllWasmInstantiateAllowlistResponse>, I>>(
    object: I,
  ): QueryAllWasmInstantiateAllowlistResponse {
    const message = createBaseQueryAllWasmInstantiateAllowlistResponse();
    message.allowlist = object.allowlist?.map((e) => WasmInstantiateAllowedContractCodeId.fromPartial(e)) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

/** Query defines the gRPC querier service. */
export interface Query {
  /** Queries a guardianSet by index. */
  GuardianSet(request: QueryGetGuardianSetRequest): Promise<QueryGetGuardianSetResponse>;
  /** Queries a list of guardianSet items. */
  GuardianSetAll(request: QueryAllGuardianSetRequest): Promise<QueryAllGuardianSetResponse>;
  /** Queries a config by index. */
  Config(request: QueryGetConfigRequest): Promise<QueryGetConfigResponse>;
  /** Queries a replayProtection by index. */
  ReplayProtection(request: QueryGetReplayProtectionRequest): Promise<QueryGetReplayProtectionResponse>;
  /** Queries a list of replayProtection items. */
  ReplayProtectionAll(request: QueryAllReplayProtectionRequest): Promise<QueryAllReplayProtectionResponse>;
  /** Queries a sequenceCounter by index. */
  SequenceCounter(request: QueryGetSequenceCounterRequest): Promise<QueryGetSequenceCounterResponse>;
  /** Queries a list of sequenceCounter items. */
  SequenceCounterAll(request: QueryAllSequenceCounterRequest): Promise<QueryAllSequenceCounterResponse>;
  /** Queries a ConsensusGuardianSetIndex by index. */
  ConsensusGuardianSetIndex(
    request: QueryGetConsensusGuardianSetIndexRequest,
  ): Promise<QueryGetConsensusGuardianSetIndexResponse>;
  /** Queries a GuardianValidator by index. */
  GuardianValidator(request: QueryGetGuardianValidatorRequest): Promise<QueryGetGuardianValidatorResponse>;
  /** Queries a list of GuardianValidator items. */
  GuardianValidatorAll(request: QueryAllGuardianValidatorRequest): Promise<QueryAllGuardianValidatorResponse>;
  /** Queries a list of LatestGuardianSetIndex items. */
  LatestGuardianSetIndex(request: QueryLatestGuardianSetIndexRequest): Promise<QueryLatestGuardianSetIndexResponse>;
  AllowlistAll(request: QueryAllValidatorAllowlist): Promise<QueryAllValidatorAllowlistResponse>;
  Allowlist(request: QueryValidatorAllowlist): Promise<QueryValidatorAllowlistResponse>;
  IbcComposabilityMwContract(
    request: QueryIbcComposabilityMwContractRequest,
  ): Promise<QueryIbcComposabilityMwContractResponse>;
  WasmInstantiateAllowlistAll(
    request: QueryAllWasmInstantiateAllowlist,
  ): Promise<QueryAllWasmInstantiateAllowlistResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
    this.GuardianSet = this.GuardianSet.bind(this);
    this.GuardianSetAll = this.GuardianSetAll.bind(this);
    this.Config = this.Config.bind(this);
    this.ReplayProtection = this.ReplayProtection.bind(this);
    this.ReplayProtectionAll = this.ReplayProtectionAll.bind(this);
    this.SequenceCounter = this.SequenceCounter.bind(this);
    this.SequenceCounterAll = this.SequenceCounterAll.bind(this);
    this.ConsensusGuardianSetIndex = this.ConsensusGuardianSetIndex.bind(this);
    this.GuardianValidator = this.GuardianValidator.bind(this);
    this.GuardianValidatorAll = this.GuardianValidatorAll.bind(this);
    this.LatestGuardianSetIndex = this.LatestGuardianSetIndex.bind(this);
    this.AllowlistAll = this.AllowlistAll.bind(this);
    this.Allowlist = this.Allowlist.bind(this);
    this.IbcComposabilityMwContract = this.IbcComposabilityMwContract.bind(this);
    this.WasmInstantiateAllowlistAll = this.WasmInstantiateAllowlistAll.bind(this);
  }
  GuardianSet(request: QueryGetGuardianSetRequest): Promise<QueryGetGuardianSetResponse> {
    const data = QueryGetGuardianSetRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "GuardianSet", data);
    return promise.then((data) => QueryGetGuardianSetResponse.decode(new _m0.Reader(data)));
  }

  GuardianSetAll(request: QueryAllGuardianSetRequest): Promise<QueryAllGuardianSetResponse> {
    const data = QueryAllGuardianSetRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "GuardianSetAll", data);
    return promise.then((data) => QueryAllGuardianSetResponse.decode(new _m0.Reader(data)));
  }

  Config(request: QueryGetConfigRequest): Promise<QueryGetConfigResponse> {
    const data = QueryGetConfigRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "Config", data);
    return promise.then((data) => QueryGetConfigResponse.decode(new _m0.Reader(data)));
  }

  ReplayProtection(request: QueryGetReplayProtectionRequest): Promise<QueryGetReplayProtectionResponse> {
    const data = QueryGetReplayProtectionRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "ReplayProtection", data);
    return promise.then((data) => QueryGetReplayProtectionResponse.decode(new _m0.Reader(data)));
  }

  ReplayProtectionAll(request: QueryAllReplayProtectionRequest): Promise<QueryAllReplayProtectionResponse> {
    const data = QueryAllReplayProtectionRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "ReplayProtectionAll", data);
    return promise.then((data) => QueryAllReplayProtectionResponse.decode(new _m0.Reader(data)));
  }

  SequenceCounter(request: QueryGetSequenceCounterRequest): Promise<QueryGetSequenceCounterResponse> {
    const data = QueryGetSequenceCounterRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "SequenceCounter", data);
    return promise.then((data) => QueryGetSequenceCounterResponse.decode(new _m0.Reader(data)));
  }

  SequenceCounterAll(request: QueryAllSequenceCounterRequest): Promise<QueryAllSequenceCounterResponse> {
    const data = QueryAllSequenceCounterRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "SequenceCounterAll", data);
    return promise.then((data) => QueryAllSequenceCounterResponse.decode(new _m0.Reader(data)));
  }

  ConsensusGuardianSetIndex(
    request: QueryGetConsensusGuardianSetIndexRequest,
  ): Promise<QueryGetConsensusGuardianSetIndexResponse> {
    const data = QueryGetConsensusGuardianSetIndexRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "ConsensusGuardianSetIndex", data);
    return promise.then((data) => QueryGetConsensusGuardianSetIndexResponse.decode(new _m0.Reader(data)));
  }

  GuardianValidator(request: QueryGetGuardianValidatorRequest): Promise<QueryGetGuardianValidatorResponse> {
    const data = QueryGetGuardianValidatorRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "GuardianValidator", data);
    return promise.then((data) => QueryGetGuardianValidatorResponse.decode(new _m0.Reader(data)));
  }

  GuardianValidatorAll(request: QueryAllGuardianValidatorRequest): Promise<QueryAllGuardianValidatorResponse> {
    const data = QueryAllGuardianValidatorRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "GuardianValidatorAll", data);
    return promise.then((data) => QueryAllGuardianValidatorResponse.decode(new _m0.Reader(data)));
  }

  LatestGuardianSetIndex(request: QueryLatestGuardianSetIndexRequest): Promise<QueryLatestGuardianSetIndexResponse> {
    const data = QueryLatestGuardianSetIndexRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "LatestGuardianSetIndex", data);
    return promise.then((data) => QueryLatestGuardianSetIndexResponse.decode(new _m0.Reader(data)));
  }

  AllowlistAll(request: QueryAllValidatorAllowlist): Promise<QueryAllValidatorAllowlistResponse> {
    const data = QueryAllValidatorAllowlist.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "AllowlistAll", data);
    return promise.then((data) => QueryAllValidatorAllowlistResponse.decode(new _m0.Reader(data)));
  }

  Allowlist(request: QueryValidatorAllowlist): Promise<QueryValidatorAllowlistResponse> {
    const data = QueryValidatorAllowlist.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "Allowlist", data);
    return promise.then((data) => QueryValidatorAllowlistResponse.decode(new _m0.Reader(data)));
  }

  IbcComposabilityMwContract(
    request: QueryIbcComposabilityMwContractRequest,
  ): Promise<QueryIbcComposabilityMwContractResponse> {
    const data = QueryIbcComposabilityMwContractRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "IbcComposabilityMwContract", data);
    return promise.then((data) => QueryIbcComposabilityMwContractResponse.decode(new _m0.Reader(data)));
  }

  WasmInstantiateAllowlistAll(
    request: QueryAllWasmInstantiateAllowlist,
  ): Promise<QueryAllWasmInstantiateAllowlistResponse> {
    const data = QueryAllWasmInstantiateAllowlist.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Query", "WasmInstantiateAllowlistAll", data);
    return promise.then((data) => QueryAllWasmInstantiateAllowlistResponse.decode(new _m0.Reader(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}

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

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
