/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { GuardianSet } from "../wormhole/guardian_set";
import {
  PageRequest,
  PageResponse,
} from "../cosmos/base/query/v1beta1/pagination";
import { Config } from "../wormhole/config";
import { ReplayProtection } from "../wormhole/replay_protection";
import { SequenceCounter } from "../wormhole/sequence_counter";

export const protobufPackage = "certusone.wormholechain.wormhole";

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
      "certusone.wormholechain.wormhole.Query",
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
      "certusone.wormholechain.wormhole.Query",
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
      "certusone.wormholechain.wormhole.Query",
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
      "certusone.wormholechain.wormhole.Query",
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
      "certusone.wormholechain.wormhole.Query",
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
      "certusone.wormholechain.wormhole.Query",
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
      "certusone.wormholechain.wormhole.Query",
      "SequenceCounterAll",
      data
    );
    return promise.then((data) =>
      QueryAllSequenceCounterResponse.decode(new Reader(data))
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
