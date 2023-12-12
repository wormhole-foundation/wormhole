//@ts-nocheck
/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import {
  PageRequest,
  PageResponse,
} from "../cosmos/base/query/v1beta1/pagination";
import {
  ValidatorAllowedAddress,
  GuardianSet,
  GuardianValidator,
  WasmInstantiateAllowedContractCodeId,
} from "../wormhole/guardian";
import { Config } from "../wormhole/config";
import { ReplayProtection } from "../wormhole/replay_protection";
import { SequenceCounter } from "../wormhole/sequence_counter";
import { ConsensusGuardianSetIndex } from "../wormhole/consensus_guardian_set_index";

export const protobufPackage = "wormhole_foundation.wormchain.wormhole";

export interface QueryAllValidatorAllowlist {
  pagination: PageRequest | undefined;
}

/** all allowlisted entries by all validators */
export interface QueryAllValidatorAllowlistResponse {
  allowlist: ValidatorAllowedAddress[];
  pagination: PageResponse | undefined;
}

export interface QueryValidatorAllowlist {
  validator_address: string;
  pagination: PageRequest | undefined;
}

/** all allowlisted entries by a specific validator */
export interface QueryValidatorAllowlistResponse {
  validator_address: string;
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

export interface QueryGetConfigRequest {}

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

export interface QueryGetConsensusGuardianSetIndexRequest {}

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

export interface QueryLatestGuardianSetIndexRequest {}

export interface QueryLatestGuardianSetIndexResponse {
  latestGuardianSetIndex: number;
}

export interface QueryIbcComposabilityMwContractRequest {}

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

const baseQueryAllValidatorAllowlist: object = {};

export const QueryAllValidatorAllowlist = {
  encode(
    message: QueryAllValidatorAllowlist,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllValidatorAllowlist {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllValidatorAllowlist,
    } as QueryAllValidatorAllowlist;
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
    const message = {
      ...baseQueryAllValidatorAllowlist,
    } as QueryAllValidatorAllowlist;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllValidatorAllowlist): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllValidatorAllowlist>
  ): QueryAllValidatorAllowlist {
    const message = {
      ...baseQueryAllValidatorAllowlist,
    } as QueryAllValidatorAllowlist;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllValidatorAllowlistResponse: object = {};

export const QueryAllValidatorAllowlistResponse = {
  encode(
    message: QueryAllValidatorAllowlistResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.allowlist) {
      ValidatorAllowedAddress.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllValidatorAllowlistResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllValidatorAllowlistResponse,
    } as QueryAllValidatorAllowlistResponse;
    message.allowlist = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.allowlist.push(
            ValidatorAllowedAddress.decode(reader, reader.uint32())
          );
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
    const message = {
      ...baseQueryAllValidatorAllowlistResponse,
    } as QueryAllValidatorAllowlistResponse;
    message.allowlist = [];
    if (object.allowlist !== undefined && object.allowlist !== null) {
      for (const e of object.allowlist) {
        message.allowlist.push(ValidatorAllowedAddress.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllValidatorAllowlistResponse): unknown {
    const obj: any = {};
    if (message.allowlist) {
      obj.allowlist = message.allowlist.map((e) =>
        e ? ValidatorAllowedAddress.toJSON(e) : undefined
      );
    } else {
      obj.allowlist = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllValidatorAllowlistResponse>
  ): QueryAllValidatorAllowlistResponse {
    const message = {
      ...baseQueryAllValidatorAllowlistResponse,
    } as QueryAllValidatorAllowlistResponse;
    message.allowlist = [];
    if (object.allowlist !== undefined && object.allowlist !== null) {
      for (const e of object.allowlist) {
        message.allowlist.push(ValidatorAllowedAddress.fromPartial(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryValidatorAllowlist: object = { validator_address: "" };

export const QueryValidatorAllowlist = {
  encode(
    message: QueryValidatorAllowlist,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.validator_address !== "") {
      writer.uint32(10).string(message.validator_address);
    }
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryValidatorAllowlist {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryValidatorAllowlist,
    } as QueryValidatorAllowlist;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator_address = reader.string();
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
    const message = {
      ...baseQueryValidatorAllowlist,
    } as QueryValidatorAllowlist;
    if (
      object.validator_address !== undefined &&
      object.validator_address !== null
    ) {
      message.validator_address = String(object.validator_address);
    } else {
      message.validator_address = "";
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryValidatorAllowlist): unknown {
    const obj: any = {};
    message.validator_address !== undefined &&
      (obj.validator_address = message.validator_address);
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryValidatorAllowlist>
  ): QueryValidatorAllowlist {
    const message = {
      ...baseQueryValidatorAllowlist,
    } as QueryValidatorAllowlist;
    if (
      object.validator_address !== undefined &&
      object.validator_address !== null
    ) {
      message.validator_address = object.validator_address;
    } else {
      message.validator_address = "";
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryValidatorAllowlistResponse: object = { validator_address: "" };

export const QueryValidatorAllowlistResponse = {
  encode(
    message: QueryValidatorAllowlistResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.validator_address !== "") {
      writer.uint32(10).string(message.validator_address);
    }
    for (const v of message.allowlist) {
      ValidatorAllowedAddress.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(26).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryValidatorAllowlistResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryValidatorAllowlistResponse,
    } as QueryValidatorAllowlistResponse;
    message.allowlist = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator_address = reader.string();
          break;
        case 2:
          message.allowlist.push(
            ValidatorAllowedAddress.decode(reader, reader.uint32())
          );
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
    const message = {
      ...baseQueryValidatorAllowlistResponse,
    } as QueryValidatorAllowlistResponse;
    message.allowlist = [];
    if (
      object.validator_address !== undefined &&
      object.validator_address !== null
    ) {
      message.validator_address = String(object.validator_address);
    } else {
      message.validator_address = "";
    }
    if (object.allowlist !== undefined && object.allowlist !== null) {
      for (const e of object.allowlist) {
        message.allowlist.push(ValidatorAllowedAddress.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryValidatorAllowlistResponse): unknown {
    const obj: any = {};
    message.validator_address !== undefined &&
      (obj.validator_address = message.validator_address);
    if (message.allowlist) {
      obj.allowlist = message.allowlist.map((e) =>
        e ? ValidatorAllowedAddress.toJSON(e) : undefined
      );
    } else {
      obj.allowlist = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryValidatorAllowlistResponse>
  ): QueryValidatorAllowlistResponse {
    const message = {
      ...baseQueryValidatorAllowlistResponse,
    } as QueryValidatorAllowlistResponse;
    message.allowlist = [];
    if (
      object.validator_address !== undefined &&
      object.validator_address !== null
    ) {
      message.validator_address = object.validator_address;
    } else {
      message.validator_address = "";
    }
    if (object.allowlist !== undefined && object.allowlist !== null) {
      for (const e of object.allowlist) {
        message.allowlist.push(ValidatorAllowedAddress.fromPartial(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryGetGuardianSetRequest: object = { index: 0 };

export const QueryGetGuardianSetRequest = {
  encode(
    message: QueryGetGuardianSetRequest,
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
  ): QueryGetGuardianSetRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetGuardianSetRequest,
    } as QueryGetGuardianSetRequest;
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
    const message = {
      ...baseQueryGetGuardianSetRequest,
    } as QueryGetGuardianSetRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = Number(object.index);
    } else {
      message.index = 0;
    }
    return message;
  },

  toJSON(message: QueryGetGuardianSetRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetGuardianSetRequest>
  ): QueryGetGuardianSetRequest {
    const message = {
      ...baseQueryGetGuardianSetRequest,
    } as QueryGetGuardianSetRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = 0;
    }
    return message;
  },
};

const baseQueryGetGuardianSetResponse: object = {};

export const QueryGetGuardianSetResponse = {
  encode(
    message: QueryGetGuardianSetResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.GuardianSet !== undefined) {
      GuardianSet.encode(
        message.GuardianSet,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetGuardianSetResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetGuardianSetResponse,
    } as QueryGetGuardianSetResponse;
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
    const message = {
      ...baseQueryGetGuardianSetResponse,
    } as QueryGetGuardianSetResponse;
    if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
      message.GuardianSet = GuardianSet.fromJSON(object.GuardianSet);
    } else {
      message.GuardianSet = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetGuardianSetResponse): unknown {
    const obj: any = {};
    message.GuardianSet !== undefined &&
      (obj.GuardianSet = message.GuardianSet
        ? GuardianSet.toJSON(message.GuardianSet)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetGuardianSetResponse>
  ): QueryGetGuardianSetResponse {
    const message = {
      ...baseQueryGetGuardianSetResponse,
    } as QueryGetGuardianSetResponse;
    if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
      message.GuardianSet = GuardianSet.fromPartial(object.GuardianSet);
    } else {
      message.GuardianSet = undefined;
    }
    return message;
  },
};

const baseQueryAllGuardianSetRequest: object = {};

export const QueryAllGuardianSetRequest = {
  encode(
    message: QueryAllGuardianSetRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllGuardianSetRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllGuardianSetRequest,
    } as QueryAllGuardianSetRequest;
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
    const message = {
      ...baseQueryAllGuardianSetRequest,
    } as QueryAllGuardianSetRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllGuardianSetRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllGuardianSetRequest>
  ): QueryAllGuardianSetRequest {
    const message = {
      ...baseQueryAllGuardianSetRequest,
    } as QueryAllGuardianSetRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllGuardianSetResponse: object = {};

export const QueryAllGuardianSetResponse = {
  encode(
    message: QueryAllGuardianSetResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.GuardianSet) {
      GuardianSet.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllGuardianSetResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllGuardianSetResponse,
    } as QueryAllGuardianSetResponse;
    message.GuardianSet = [];
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
    const message = {
      ...baseQueryAllGuardianSetResponse,
    } as QueryAllGuardianSetResponse;
    message.GuardianSet = [];
    if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
      for (const e of object.GuardianSet) {
        message.GuardianSet.push(GuardianSet.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllGuardianSetResponse): unknown {
    const obj: any = {};
    if (message.GuardianSet) {
      obj.GuardianSet = message.GuardianSet.map((e) =>
        e ? GuardianSet.toJSON(e) : undefined
      );
    } else {
      obj.GuardianSet = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllGuardianSetResponse>
  ): QueryAllGuardianSetResponse {
    const message = {
      ...baseQueryAllGuardianSetResponse,
    } as QueryAllGuardianSetResponse;
    message.GuardianSet = [];
    if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
      for (const e of object.GuardianSet) {
        message.GuardianSet.push(GuardianSet.fromPartial(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryGetConfigRequest: object = {};

export const QueryGetConfigRequest = {
  encode(_: QueryGetConfigRequest, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryGetConfigRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryGetConfigRequest } as QueryGetConfigRequest;
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
    const message = { ...baseQueryGetConfigRequest } as QueryGetConfigRequest;
    return message;
  },

  toJSON(_: QueryGetConfigRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<QueryGetConfigRequest>): QueryGetConfigRequest {
    const message = { ...baseQueryGetConfigRequest } as QueryGetConfigRequest;
    return message;
  },
};

const baseQueryGetConfigResponse: object = {};

export const QueryGetConfigResponse = {
  encode(
    message: QueryGetConfigResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.Config !== undefined) {
      Config.encode(message.Config, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): QueryGetConfigResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseQueryGetConfigResponse } as QueryGetConfigResponse;
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
    const message = { ...baseQueryGetConfigResponse } as QueryGetConfigResponse;
    if (object.Config !== undefined && object.Config !== null) {
      message.Config = Config.fromJSON(object.Config);
    } else {
      message.Config = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetConfigResponse): unknown {
    const obj: any = {};
    message.Config !== undefined &&
      (obj.Config = message.Config ? Config.toJSON(message.Config) : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetConfigResponse>
  ): QueryGetConfigResponse {
    const message = { ...baseQueryGetConfigResponse } as QueryGetConfigResponse;
    if (object.Config !== undefined && object.Config !== null) {
      message.Config = Config.fromPartial(object.Config);
    } else {
      message.Config = undefined;
    }
    return message;
  },
};

const baseQueryGetReplayProtectionRequest: object = { index: "" };

export const QueryGetReplayProtectionRequest = {
  encode(
    message: QueryGetReplayProtectionRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetReplayProtectionRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetReplayProtectionRequest,
    } as QueryGetReplayProtectionRequest;
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
    const message = {
      ...baseQueryGetReplayProtectionRequest,
    } as QueryGetReplayProtectionRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    return message;
  },

  toJSON(message: QueryGetReplayProtectionRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetReplayProtectionRequest>
  ): QueryGetReplayProtectionRequest {
    const message = {
      ...baseQueryGetReplayProtectionRequest,
    } as QueryGetReplayProtectionRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    return message;
  },
};

const baseQueryGetReplayProtectionResponse: object = {};

export const QueryGetReplayProtectionResponse = {
  encode(
    message: QueryGetReplayProtectionResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.replayProtection !== undefined) {
      ReplayProtection.encode(
        message.replayProtection,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetReplayProtectionResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetReplayProtectionResponse,
    } as QueryGetReplayProtectionResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.replayProtection = ReplayProtection.decode(
            reader,
            reader.uint32()
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetReplayProtectionResponse {
    const message = {
      ...baseQueryGetReplayProtectionResponse,
    } as QueryGetReplayProtectionResponse;
    if (
      object.replayProtection !== undefined &&
      object.replayProtection !== null
    ) {
      message.replayProtection = ReplayProtection.fromJSON(
        object.replayProtection
      );
    } else {
      message.replayProtection = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetReplayProtectionResponse): unknown {
    const obj: any = {};
    message.replayProtection !== undefined &&
      (obj.replayProtection = message.replayProtection
        ? ReplayProtection.toJSON(message.replayProtection)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetReplayProtectionResponse>
  ): QueryGetReplayProtectionResponse {
    const message = {
      ...baseQueryGetReplayProtectionResponse,
    } as QueryGetReplayProtectionResponse;
    if (
      object.replayProtection !== undefined &&
      object.replayProtection !== null
    ) {
      message.replayProtection = ReplayProtection.fromPartial(
        object.replayProtection
      );
    } else {
      message.replayProtection = undefined;
    }
    return message;
  },
};

const baseQueryAllReplayProtectionRequest: object = {};

export const QueryAllReplayProtectionRequest = {
  encode(
    message: QueryAllReplayProtectionRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllReplayProtectionRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllReplayProtectionRequest,
    } as QueryAllReplayProtectionRequest;
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
    const message = {
      ...baseQueryAllReplayProtectionRequest,
    } as QueryAllReplayProtectionRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllReplayProtectionRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllReplayProtectionRequest>
  ): QueryAllReplayProtectionRequest {
    const message = {
      ...baseQueryAllReplayProtectionRequest,
    } as QueryAllReplayProtectionRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllReplayProtectionResponse: object = {};

export const QueryAllReplayProtectionResponse = {
  encode(
    message: QueryAllReplayProtectionResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.replayProtection) {
      ReplayProtection.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllReplayProtectionResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllReplayProtectionResponse,
    } as QueryAllReplayProtectionResponse;
    message.replayProtection = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.replayProtection.push(
            ReplayProtection.decode(reader, reader.uint32())
          );
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
    const message = {
      ...baseQueryAllReplayProtectionResponse,
    } as QueryAllReplayProtectionResponse;
    message.replayProtection = [];
    if (
      object.replayProtection !== undefined &&
      object.replayProtection !== null
    ) {
      for (const e of object.replayProtection) {
        message.replayProtection.push(ReplayProtection.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllReplayProtectionResponse): unknown {
    const obj: any = {};
    if (message.replayProtection) {
      obj.replayProtection = message.replayProtection.map((e) =>
        e ? ReplayProtection.toJSON(e) : undefined
      );
    } else {
      obj.replayProtection = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllReplayProtectionResponse>
  ): QueryAllReplayProtectionResponse {
    const message = {
      ...baseQueryAllReplayProtectionResponse,
    } as QueryAllReplayProtectionResponse;
    message.replayProtection = [];
    if (
      object.replayProtection !== undefined &&
      object.replayProtection !== null
    ) {
      for (const e of object.replayProtection) {
        message.replayProtection.push(ReplayProtection.fromPartial(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryGetSequenceCounterRequest: object = { index: "" };

export const QueryGetSequenceCounterRequest = {
  encode(
    message: QueryGetSequenceCounterRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.index !== "") {
      writer.uint32(10).string(message.index);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetSequenceCounterRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetSequenceCounterRequest,
    } as QueryGetSequenceCounterRequest;
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
    const message = {
      ...baseQueryGetSequenceCounterRequest,
    } as QueryGetSequenceCounterRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    return message;
  },

  toJSON(message: QueryGetSequenceCounterRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetSequenceCounterRequest>
  ): QueryGetSequenceCounterRequest {
    const message = {
      ...baseQueryGetSequenceCounterRequest,
    } as QueryGetSequenceCounterRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    return message;
  },
};

const baseQueryGetSequenceCounterResponse: object = {};

export const QueryGetSequenceCounterResponse = {
  encode(
    message: QueryGetSequenceCounterResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.sequenceCounter !== undefined) {
      SequenceCounter.encode(
        message.sequenceCounter,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetSequenceCounterResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetSequenceCounterResponse,
    } as QueryGetSequenceCounterResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sequenceCounter = SequenceCounter.decode(
            reader,
            reader.uint32()
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetSequenceCounterResponse {
    const message = {
      ...baseQueryGetSequenceCounterResponse,
    } as QueryGetSequenceCounterResponse;
    if (
      object.sequenceCounter !== undefined &&
      object.sequenceCounter !== null
    ) {
      message.sequenceCounter = SequenceCounter.fromJSON(
        object.sequenceCounter
      );
    } else {
      message.sequenceCounter = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetSequenceCounterResponse): unknown {
    const obj: any = {};
    message.sequenceCounter !== undefined &&
      (obj.sequenceCounter = message.sequenceCounter
        ? SequenceCounter.toJSON(message.sequenceCounter)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetSequenceCounterResponse>
  ): QueryGetSequenceCounterResponse {
    const message = {
      ...baseQueryGetSequenceCounterResponse,
    } as QueryGetSequenceCounterResponse;
    if (
      object.sequenceCounter !== undefined &&
      object.sequenceCounter !== null
    ) {
      message.sequenceCounter = SequenceCounter.fromPartial(
        object.sequenceCounter
      );
    } else {
      message.sequenceCounter = undefined;
    }
    return message;
  },
};

const baseQueryAllSequenceCounterRequest: object = {};

export const QueryAllSequenceCounterRequest = {
  encode(
    message: QueryAllSequenceCounterRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllSequenceCounterRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllSequenceCounterRequest,
    } as QueryAllSequenceCounterRequest;
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
    const message = {
      ...baseQueryAllSequenceCounterRequest,
    } as QueryAllSequenceCounterRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllSequenceCounterRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllSequenceCounterRequest>
  ): QueryAllSequenceCounterRequest {
    const message = {
      ...baseQueryAllSequenceCounterRequest,
    } as QueryAllSequenceCounterRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllSequenceCounterResponse: object = {};

export const QueryAllSequenceCounterResponse = {
  encode(
    message: QueryAllSequenceCounterResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.sequenceCounter) {
      SequenceCounter.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllSequenceCounterResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllSequenceCounterResponse,
    } as QueryAllSequenceCounterResponse;
    message.sequenceCounter = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sequenceCounter.push(
            SequenceCounter.decode(reader, reader.uint32())
          );
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
    const message = {
      ...baseQueryAllSequenceCounterResponse,
    } as QueryAllSequenceCounterResponse;
    message.sequenceCounter = [];
    if (
      object.sequenceCounter !== undefined &&
      object.sequenceCounter !== null
    ) {
      for (const e of object.sequenceCounter) {
        message.sequenceCounter.push(SequenceCounter.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllSequenceCounterResponse): unknown {
    const obj: any = {};
    if (message.sequenceCounter) {
      obj.sequenceCounter = message.sequenceCounter.map((e) =>
        e ? SequenceCounter.toJSON(e) : undefined
      );
    } else {
      obj.sequenceCounter = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllSequenceCounterResponse>
  ): QueryAllSequenceCounterResponse {
    const message = {
      ...baseQueryAllSequenceCounterResponse,
    } as QueryAllSequenceCounterResponse;
    message.sequenceCounter = [];
    if (
      object.sequenceCounter !== undefined &&
      object.sequenceCounter !== null
    ) {
      for (const e of object.sequenceCounter) {
        message.sequenceCounter.push(SequenceCounter.fromPartial(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryGetConsensusGuardianSetIndexRequest: object = {};

export const QueryGetConsensusGuardianSetIndexRequest = {
  encode(
    _: QueryGetConsensusGuardianSetIndexRequest,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetConsensusGuardianSetIndexRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetConsensusGuardianSetIndexRequest,
    } as QueryGetConsensusGuardianSetIndexRequest;
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
    const message = {
      ...baseQueryGetConsensusGuardianSetIndexRequest,
    } as QueryGetConsensusGuardianSetIndexRequest;
    return message;
  },

  toJSON(_: QueryGetConsensusGuardianSetIndexRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<QueryGetConsensusGuardianSetIndexRequest>
  ): QueryGetConsensusGuardianSetIndexRequest {
    const message = {
      ...baseQueryGetConsensusGuardianSetIndexRequest,
    } as QueryGetConsensusGuardianSetIndexRequest;
    return message;
  },
};

const baseQueryGetConsensusGuardianSetIndexResponse: object = {};

export const QueryGetConsensusGuardianSetIndexResponse = {
  encode(
    message: QueryGetConsensusGuardianSetIndexResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.ConsensusGuardianSetIndex !== undefined) {
      ConsensusGuardianSetIndex.encode(
        message.ConsensusGuardianSetIndex,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetConsensusGuardianSetIndexResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetConsensusGuardianSetIndexResponse,
    } as QueryGetConsensusGuardianSetIndexResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.ConsensusGuardianSetIndex = ConsensusGuardianSetIndex.decode(
            reader,
            reader.uint32()
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetConsensusGuardianSetIndexResponse {
    const message = {
      ...baseQueryGetConsensusGuardianSetIndexResponse,
    } as QueryGetConsensusGuardianSetIndexResponse;
    if (
      object.ConsensusGuardianSetIndex !== undefined &&
      object.ConsensusGuardianSetIndex !== null
    ) {
      message.ConsensusGuardianSetIndex = ConsensusGuardianSetIndex.fromJSON(
        object.ConsensusGuardianSetIndex
      );
    } else {
      message.ConsensusGuardianSetIndex = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetConsensusGuardianSetIndexResponse): unknown {
    const obj: any = {};
    message.ConsensusGuardianSetIndex !== undefined &&
      (obj.ConsensusGuardianSetIndex = message.ConsensusGuardianSetIndex
        ? ConsensusGuardianSetIndex.toJSON(message.ConsensusGuardianSetIndex)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetConsensusGuardianSetIndexResponse>
  ): QueryGetConsensusGuardianSetIndexResponse {
    const message = {
      ...baseQueryGetConsensusGuardianSetIndexResponse,
    } as QueryGetConsensusGuardianSetIndexResponse;
    if (
      object.ConsensusGuardianSetIndex !== undefined &&
      object.ConsensusGuardianSetIndex !== null
    ) {
      message.ConsensusGuardianSetIndex = ConsensusGuardianSetIndex.fromPartial(
        object.ConsensusGuardianSetIndex
      );
    } else {
      message.ConsensusGuardianSetIndex = undefined;
    }
    return message;
  },
};

const baseQueryGetGuardianValidatorRequest: object = {};

export const QueryGetGuardianValidatorRequest = {
  encode(
    message: QueryGetGuardianValidatorRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.guardianKey.length !== 0) {
      writer.uint32(10).bytes(message.guardianKey);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetGuardianValidatorRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetGuardianValidatorRequest,
    } as QueryGetGuardianValidatorRequest;
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
    const message = {
      ...baseQueryGetGuardianValidatorRequest,
    } as QueryGetGuardianValidatorRequest;
    if (object.guardianKey !== undefined && object.guardianKey !== null) {
      message.guardianKey = bytesFromBase64(object.guardianKey);
    }
    return message;
  },

  toJSON(message: QueryGetGuardianValidatorRequest): unknown {
    const obj: any = {};
    message.guardianKey !== undefined &&
      (obj.guardianKey = base64FromBytes(
        message.guardianKey !== undefined
          ? message.guardianKey
          : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetGuardianValidatorRequest>
  ): QueryGetGuardianValidatorRequest {
    const message = {
      ...baseQueryGetGuardianValidatorRequest,
    } as QueryGetGuardianValidatorRequest;
    if (object.guardianKey !== undefined && object.guardianKey !== null) {
      message.guardianKey = object.guardianKey;
    } else {
      message.guardianKey = new Uint8Array();
    }
    return message;
  },
};

const baseQueryGetGuardianValidatorResponse: object = {};

export const QueryGetGuardianValidatorResponse = {
  encode(
    message: QueryGetGuardianValidatorResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.guardianValidator !== undefined) {
      GuardianValidator.encode(
        message.guardianValidator,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetGuardianValidatorResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetGuardianValidatorResponse,
    } as QueryGetGuardianValidatorResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.guardianValidator = GuardianValidator.decode(
            reader,
            reader.uint32()
          );
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetGuardianValidatorResponse {
    const message = {
      ...baseQueryGetGuardianValidatorResponse,
    } as QueryGetGuardianValidatorResponse;
    if (
      object.guardianValidator !== undefined &&
      object.guardianValidator !== null
    ) {
      message.guardianValidator = GuardianValidator.fromJSON(
        object.guardianValidator
      );
    } else {
      message.guardianValidator = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetGuardianValidatorResponse): unknown {
    const obj: any = {};
    message.guardianValidator !== undefined &&
      (obj.guardianValidator = message.guardianValidator
        ? GuardianValidator.toJSON(message.guardianValidator)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetGuardianValidatorResponse>
  ): QueryGetGuardianValidatorResponse {
    const message = {
      ...baseQueryGetGuardianValidatorResponse,
    } as QueryGetGuardianValidatorResponse;
    if (
      object.guardianValidator !== undefined &&
      object.guardianValidator !== null
    ) {
      message.guardianValidator = GuardianValidator.fromPartial(
        object.guardianValidator
      );
    } else {
      message.guardianValidator = undefined;
    }
    return message;
  },
};

const baseQueryAllGuardianValidatorRequest: object = {};

export const QueryAllGuardianValidatorRequest = {
  encode(
    message: QueryAllGuardianValidatorRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllGuardianValidatorRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllGuardianValidatorRequest,
    } as QueryAllGuardianValidatorRequest;
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
    const message = {
      ...baseQueryAllGuardianValidatorRequest,
    } as QueryAllGuardianValidatorRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllGuardianValidatorRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllGuardianValidatorRequest>
  ): QueryAllGuardianValidatorRequest {
    const message = {
      ...baseQueryAllGuardianValidatorRequest,
    } as QueryAllGuardianValidatorRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllGuardianValidatorResponse: object = {};

export const QueryAllGuardianValidatorResponse = {
  encode(
    message: QueryAllGuardianValidatorResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.guardianValidator) {
      GuardianValidator.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllGuardianValidatorResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllGuardianValidatorResponse,
    } as QueryAllGuardianValidatorResponse;
    message.guardianValidator = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.guardianValidator.push(
            GuardianValidator.decode(reader, reader.uint32())
          );
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
    const message = {
      ...baseQueryAllGuardianValidatorResponse,
    } as QueryAllGuardianValidatorResponse;
    message.guardianValidator = [];
    if (
      object.guardianValidator !== undefined &&
      object.guardianValidator !== null
    ) {
      for (const e of object.guardianValidator) {
        message.guardianValidator.push(GuardianValidator.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllGuardianValidatorResponse): unknown {
    const obj: any = {};
    if (message.guardianValidator) {
      obj.guardianValidator = message.guardianValidator.map((e) =>
        e ? GuardianValidator.toJSON(e) : undefined
      );
    } else {
      obj.guardianValidator = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllGuardianValidatorResponse>
  ): QueryAllGuardianValidatorResponse {
    const message = {
      ...baseQueryAllGuardianValidatorResponse,
    } as QueryAllGuardianValidatorResponse;
    message.guardianValidator = [];
    if (
      object.guardianValidator !== undefined &&
      object.guardianValidator !== null
    ) {
      for (const e of object.guardianValidator) {
        message.guardianValidator.push(GuardianValidator.fromPartial(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryLatestGuardianSetIndexRequest: object = {};

export const QueryLatestGuardianSetIndexRequest = {
  encode(
    _: QueryLatestGuardianSetIndexRequest,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryLatestGuardianSetIndexRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryLatestGuardianSetIndexRequest,
    } as QueryLatestGuardianSetIndexRequest;
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
    const message = {
      ...baseQueryLatestGuardianSetIndexRequest,
    } as QueryLatestGuardianSetIndexRequest;
    return message;
  },

  toJSON(_: QueryLatestGuardianSetIndexRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<QueryLatestGuardianSetIndexRequest>
  ): QueryLatestGuardianSetIndexRequest {
    const message = {
      ...baseQueryLatestGuardianSetIndexRequest,
    } as QueryLatestGuardianSetIndexRequest;
    return message;
  },
};

const baseQueryLatestGuardianSetIndexResponse: object = {
  latestGuardianSetIndex: 0,
};

export const QueryLatestGuardianSetIndexResponse = {
  encode(
    message: QueryLatestGuardianSetIndexResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.latestGuardianSetIndex !== 0) {
      writer.uint32(8).uint32(message.latestGuardianSetIndex);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryLatestGuardianSetIndexResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryLatestGuardianSetIndexResponse,
    } as QueryLatestGuardianSetIndexResponse;
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
    const message = {
      ...baseQueryLatestGuardianSetIndexResponse,
    } as QueryLatestGuardianSetIndexResponse;
    if (
      object.latestGuardianSetIndex !== undefined &&
      object.latestGuardianSetIndex !== null
    ) {
      message.latestGuardianSetIndex = Number(object.latestGuardianSetIndex);
    } else {
      message.latestGuardianSetIndex = 0;
    }
    return message;
  },

  toJSON(message: QueryLatestGuardianSetIndexResponse): unknown {
    const obj: any = {};
    message.latestGuardianSetIndex !== undefined &&
      (obj.latestGuardianSetIndex = message.latestGuardianSetIndex);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryLatestGuardianSetIndexResponse>
  ): QueryLatestGuardianSetIndexResponse {
    const message = {
      ...baseQueryLatestGuardianSetIndexResponse,
    } as QueryLatestGuardianSetIndexResponse;
    if (
      object.latestGuardianSetIndex !== undefined &&
      object.latestGuardianSetIndex !== null
    ) {
      message.latestGuardianSetIndex = object.latestGuardianSetIndex;
    } else {
      message.latestGuardianSetIndex = 0;
    }
    return message;
  },
};

const baseQueryIbcComposabilityMwContractRequest: object = {};

export const QueryIbcComposabilityMwContractRequest = {
  encode(
    _: QueryIbcComposabilityMwContractRequest,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryIbcComposabilityMwContractRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryIbcComposabilityMwContractRequest,
    } as QueryIbcComposabilityMwContractRequest;
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
    const message = {
      ...baseQueryIbcComposabilityMwContractRequest,
    } as QueryIbcComposabilityMwContractRequest;
    return message;
  },

  toJSON(_: QueryIbcComposabilityMwContractRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<QueryIbcComposabilityMwContractRequest>
  ): QueryIbcComposabilityMwContractRequest {
    const message = {
      ...baseQueryIbcComposabilityMwContractRequest,
    } as QueryIbcComposabilityMwContractRequest;
    return message;
  },
};

const baseQueryIbcComposabilityMwContractResponse: object = {
  contractAddress: "",
};

export const QueryIbcComposabilityMwContractResponse = {
  encode(
    message: QueryIbcComposabilityMwContractResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.contractAddress !== "") {
      writer.uint32(10).string(message.contractAddress);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryIbcComposabilityMwContractResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryIbcComposabilityMwContractResponse,
    } as QueryIbcComposabilityMwContractResponse;
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
    const message = {
      ...baseQueryIbcComposabilityMwContractResponse,
    } as QueryIbcComposabilityMwContractResponse;
    if (
      object.contractAddress !== undefined &&
      object.contractAddress !== null
    ) {
      message.contractAddress = String(object.contractAddress);
    } else {
      message.contractAddress = "";
    }
    return message;
  },

  toJSON(message: QueryIbcComposabilityMwContractResponse): unknown {
    const obj: any = {};
    message.contractAddress !== undefined &&
      (obj.contractAddress = message.contractAddress);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryIbcComposabilityMwContractResponse>
  ): QueryIbcComposabilityMwContractResponse {
    const message = {
      ...baseQueryIbcComposabilityMwContractResponse,
    } as QueryIbcComposabilityMwContractResponse;
    if (
      object.contractAddress !== undefined &&
      object.contractAddress !== null
    ) {
      message.contractAddress = object.contractAddress;
    } else {
      message.contractAddress = "";
    }
    return message;
  },
};

const baseQueryAllWasmInstantiateAllowlist: object = {};

export const QueryAllWasmInstantiateAllowlist = {
  encode(
    message: QueryAllWasmInstantiateAllowlist,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllWasmInstantiateAllowlist {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllWasmInstantiateAllowlist,
    } as QueryAllWasmInstantiateAllowlist;
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
    const message = {
      ...baseQueryAllWasmInstantiateAllowlist,
    } as QueryAllWasmInstantiateAllowlist;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllWasmInstantiateAllowlist): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllWasmInstantiateAllowlist>
  ): QueryAllWasmInstantiateAllowlist {
    const message = {
      ...baseQueryAllWasmInstantiateAllowlist,
    } as QueryAllWasmInstantiateAllowlist;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllWasmInstantiateAllowlistResponse: object = {};

export const QueryAllWasmInstantiateAllowlistResponse = {
  encode(
    message: QueryAllWasmInstantiateAllowlistResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.allowlist) {
      WasmInstantiateAllowedContractCodeId.encode(
        v!,
        writer.uint32(10).fork()
      ).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(
        message.pagination,
        writer.uint32(18).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryAllWasmInstantiateAllowlistResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllWasmInstantiateAllowlistResponse,
    } as QueryAllWasmInstantiateAllowlistResponse;
    message.allowlist = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.allowlist.push(
            WasmInstantiateAllowedContractCodeId.decode(reader, reader.uint32())
          );
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
    const message = {
      ...baseQueryAllWasmInstantiateAllowlistResponse,
    } as QueryAllWasmInstantiateAllowlistResponse;
    message.allowlist = [];
    if (object.allowlist !== undefined && object.allowlist !== null) {
      for (const e of object.allowlist) {
        message.allowlist.push(
          WasmInstantiateAllowedContractCodeId.fromJSON(e)
        );
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllWasmInstantiateAllowlistResponse): unknown {
    const obj: any = {};
    if (message.allowlist) {
      obj.allowlist = message.allowlist.map((e) =>
        e ? WasmInstantiateAllowedContractCodeId.toJSON(e) : undefined
      );
    } else {
      obj.allowlist = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllWasmInstantiateAllowlistResponse>
  ): QueryAllWasmInstantiateAllowlistResponse {
    const message = {
      ...baseQueryAllWasmInstantiateAllowlistResponse,
    } as QueryAllWasmInstantiateAllowlistResponse;
    message.allowlist = [];
    if (object.allowlist !== undefined && object.allowlist !== null) {
      for (const e of object.allowlist) {
        message.allowlist.push(
          WasmInstantiateAllowedContractCodeId.fromPartial(e)
        );
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

/** Query defines the gRPC querier service. */
export interface Query {
  /** Queries a guardianSet by index. */
  GuardianSet(
    request: QueryGetGuardianSetRequest
  ): Promise<QueryGetGuardianSetResponse>;
  /** Queries a list of guardianSet items. */
  GuardianSetAll(
    request: QueryAllGuardianSetRequest
  ): Promise<QueryAllGuardianSetResponse>;
  /** Queries a config by index. */
  Config(request: QueryGetConfigRequest): Promise<QueryGetConfigResponse>;
  /** Queries a replayProtection by index. */
  ReplayProtection(
    request: QueryGetReplayProtectionRequest
  ): Promise<QueryGetReplayProtectionResponse>;
  /** Queries a list of replayProtection items. */
  ReplayProtectionAll(
    request: QueryAllReplayProtectionRequest
  ): Promise<QueryAllReplayProtectionResponse>;
  /** Queries a sequenceCounter by index. */
  SequenceCounter(
    request: QueryGetSequenceCounterRequest
  ): Promise<QueryGetSequenceCounterResponse>;
  /** Queries a list of sequenceCounter items. */
  SequenceCounterAll(
    request: QueryAllSequenceCounterRequest
  ): Promise<QueryAllSequenceCounterResponse>;
  /** Queries a ConsensusGuardianSetIndex by index. */
  ConsensusGuardianSetIndex(
    request: QueryGetConsensusGuardianSetIndexRequest
  ): Promise<QueryGetConsensusGuardianSetIndexResponse>;
  /** Queries a GuardianValidator by index. */
  GuardianValidator(
    request: QueryGetGuardianValidatorRequest
  ): Promise<QueryGetGuardianValidatorResponse>;
  /** Queries a list of GuardianValidator items. */
  GuardianValidatorAll(
    request: QueryAllGuardianValidatorRequest
  ): Promise<QueryAllGuardianValidatorResponse>;
  /** Queries a list of LatestGuardianSetIndex items. */
  LatestGuardianSetIndex(
    request: QueryLatestGuardianSetIndexRequest
  ): Promise<QueryLatestGuardianSetIndexResponse>;
  AllowlistAll(
    request: QueryAllValidatorAllowlist
  ): Promise<QueryAllValidatorAllowlistResponse>;
  Allowlist(
    request: QueryValidatorAllowlist
  ): Promise<QueryValidatorAllowlistResponse>;
  IbcComposabilityMwContract(
    request: QueryIbcComposabilityMwContractRequest
  ): Promise<QueryIbcComposabilityMwContractResponse>;
  WasmInstantiateAllowlistAll(
    request: QueryAllWasmInstantiateAllowlist
  ): Promise<QueryAllWasmInstantiateAllowlistResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  GuardianSet(
    request: QueryGetGuardianSetRequest
  ): Promise<QueryGetGuardianSetResponse> {
    const data = QueryGetGuardianSetRequest.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "GuardianSet",
      data
    );
    return promise.then((data) =>
      QueryGetGuardianSetResponse.decode(new Reader(data))
    );
  }

  GuardianSetAll(
    request: QueryAllGuardianSetRequest
  ): Promise<QueryAllGuardianSetResponse> {
    const data = QueryAllGuardianSetRequest.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "GuardianSetAll",
      data
    );
    return promise.then((data) =>
      QueryAllGuardianSetResponse.decode(new Reader(data))
    );
  }

  Config(request: QueryGetConfigRequest): Promise<QueryGetConfigResponse> {
    const data = QueryGetConfigRequest.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "Config",
      data
    );
    return promise.then((data) =>
      QueryGetConfigResponse.decode(new Reader(data))
    );
  }

  ReplayProtection(
    request: QueryGetReplayProtectionRequest
  ): Promise<QueryGetReplayProtectionResponse> {
    const data = QueryGetReplayProtectionRequest.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "ReplayProtection",
      data
    );
    return promise.then((data) =>
      QueryGetReplayProtectionResponse.decode(new Reader(data))
    );
  }

  ReplayProtectionAll(
    request: QueryAllReplayProtectionRequest
  ): Promise<QueryAllReplayProtectionResponse> {
    const data = QueryAllReplayProtectionRequest.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "ReplayProtectionAll",
      data
    );
    return promise.then((data) =>
      QueryAllReplayProtectionResponse.decode(new Reader(data))
    );
  }

  SequenceCounter(
    request: QueryGetSequenceCounterRequest
  ): Promise<QueryGetSequenceCounterResponse> {
    const data = QueryGetSequenceCounterRequest.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "SequenceCounter",
      data
    );
    return promise.then((data) =>
      QueryGetSequenceCounterResponse.decode(new Reader(data))
    );
  }

  SequenceCounterAll(
    request: QueryAllSequenceCounterRequest
  ): Promise<QueryAllSequenceCounterResponse> {
    const data = QueryAllSequenceCounterRequest.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "SequenceCounterAll",
      data
    );
    return promise.then((data) =>
      QueryAllSequenceCounterResponse.decode(new Reader(data))
    );
  }

  ConsensusGuardianSetIndex(
    request: QueryGetConsensusGuardianSetIndexRequest
  ): Promise<QueryGetConsensusGuardianSetIndexResponse> {
    const data = QueryGetConsensusGuardianSetIndexRequest.encode(
      request
    ).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "ConsensusGuardianSetIndex",
      data
    );
    return promise.then((data) =>
      QueryGetConsensusGuardianSetIndexResponse.decode(new Reader(data))
    );
  }

  GuardianValidator(
    request: QueryGetGuardianValidatorRequest
  ): Promise<QueryGetGuardianValidatorResponse> {
    const data = QueryGetGuardianValidatorRequest.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "GuardianValidator",
      data
    );
    return promise.then((data) =>
      QueryGetGuardianValidatorResponse.decode(new Reader(data))
    );
  }

  GuardianValidatorAll(
    request: QueryAllGuardianValidatorRequest
  ): Promise<QueryAllGuardianValidatorResponse> {
    const data = QueryAllGuardianValidatorRequest.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "GuardianValidatorAll",
      data
    );
    return promise.then((data) =>
      QueryAllGuardianValidatorResponse.decode(new Reader(data))
    );
  }

  LatestGuardianSetIndex(
    request: QueryLatestGuardianSetIndexRequest
  ): Promise<QueryLatestGuardianSetIndexResponse> {
    const data = QueryLatestGuardianSetIndexRequest.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "LatestGuardianSetIndex",
      data
    );
    return promise.then((data) =>
      QueryLatestGuardianSetIndexResponse.decode(new Reader(data))
    );
  }

  AllowlistAll(
    request: QueryAllValidatorAllowlist
  ): Promise<QueryAllValidatorAllowlistResponse> {
    const data = QueryAllValidatorAllowlist.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "AllowlistAll",
      data
    );
    return promise.then((data) =>
      QueryAllValidatorAllowlistResponse.decode(new Reader(data))
    );
  }

  Allowlist(
    request: QueryValidatorAllowlist
  ): Promise<QueryValidatorAllowlistResponse> {
    const data = QueryValidatorAllowlist.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "Allowlist",
      data
    );
    return promise.then((data) =>
      QueryValidatorAllowlistResponse.decode(new Reader(data))
    );
  }

  IbcComposabilityMwContract(
    request: QueryIbcComposabilityMwContractRequest
  ): Promise<QueryIbcComposabilityMwContractResponse> {
    const data = QueryIbcComposabilityMwContractRequest.encode(
      request
    ).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "IbcComposabilityMwContract",
      data
    );
    return promise.then((data) =>
      QueryIbcComposabilityMwContractResponse.decode(new Reader(data))
    );
  }

  WasmInstantiateAllowlistAll(
    request: QueryAllWasmInstantiateAllowlist
  ): Promise<QueryAllWasmInstantiateAllowlistResponse> {
    const data = QueryAllWasmInstantiateAllowlist.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Query",
      "WasmInstantiateAllowlistAll",
      data
    );
    return promise.then((data) =>
      QueryAllWasmInstantiateAllowlistResponse.decode(new Reader(data))
    );
  }
}

interface Rpc {
  request(
    service: string,
    method: string,
    data: Uint8Array
  ): Promise<Uint8Array>;
}

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
