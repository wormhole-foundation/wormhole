//@ts-nocheck
/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { Config } from "../tokenbridge/config";
import { ReplayProtection } from "../tokenbridge/replay_protection";
import {
  PageRequest,
  PageResponse,
} from "../cosmos/base/query/v1beta1/pagination";
import { ChainRegistration } from "../tokenbridge/chain_registration";
import { CoinMetaRollbackProtection } from "../tokenbridge/coin_meta_rollback_protection";

export const protobufPackage = "certusone.wormholechain.tokenbridge";

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

export interface QueryGetChainRegistrationRequest {
  chainID: number;
}

export interface QueryGetChainRegistrationResponse {
  chainRegistration: ChainRegistration | undefined;
}

export interface QueryAllChainRegistrationRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllChainRegistrationResponse {
  chainRegistration: ChainRegistration[];
  pagination: PageResponse | undefined;
}

export interface QueryGetCoinMetaRollbackProtectionRequest {
  index: string;
}

export interface QueryGetCoinMetaRollbackProtectionResponse {
  coinMetaRollbackProtection: CoinMetaRollbackProtection | undefined;
}

export interface QueryAllCoinMetaRollbackProtectionRequest {
  pagination: PageRequest | undefined;
}

export interface QueryAllCoinMetaRollbackProtectionResponse {
  coinMetaRollbackProtection: CoinMetaRollbackProtection[];
  pagination: PageResponse | undefined;
}

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

const baseQueryGetChainRegistrationRequest: object = { chainID: 0 };

export const QueryGetChainRegistrationRequest = {
  encode(
    message: QueryGetChainRegistrationRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.chainID !== 0) {
      writer.uint32(8).uint32(message.chainID);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetChainRegistrationRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetChainRegistrationRequest,
    } as QueryGetChainRegistrationRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.chainID = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): QueryGetChainRegistrationRequest {
    const message = {
      ...baseQueryGetChainRegistrationRequest,
    } as QueryGetChainRegistrationRequest;
    if (object.chainID !== undefined && object.chainID !== null) {
      message.chainID = Number(object.chainID);
    } else {
      message.chainID = 0;
    }
    return message;
  },

  toJSON(message: QueryGetChainRegistrationRequest): unknown {
    const obj: any = {};
    message.chainID !== undefined && (obj.chainID = message.chainID);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetChainRegistrationRequest>
  ): QueryGetChainRegistrationRequest {
    const message = {
      ...baseQueryGetChainRegistrationRequest,
    } as QueryGetChainRegistrationRequest;
    if (object.chainID !== undefined && object.chainID !== null) {
      message.chainID = object.chainID;
    } else {
      message.chainID = 0;
    }
    return message;
  },
};

const baseQueryGetChainRegistrationResponse: object = {};

export const QueryGetChainRegistrationResponse = {
  encode(
    message: QueryGetChainRegistrationResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.chainRegistration !== undefined) {
      ChainRegistration.encode(
        message.chainRegistration,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetChainRegistrationResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetChainRegistrationResponse,
    } as QueryGetChainRegistrationResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.chainRegistration = ChainRegistration.decode(
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

  fromJSON(object: any): QueryGetChainRegistrationResponse {
    const message = {
      ...baseQueryGetChainRegistrationResponse,
    } as QueryGetChainRegistrationResponse;
    if (
      object.chainRegistration !== undefined &&
      object.chainRegistration !== null
    ) {
      message.chainRegistration = ChainRegistration.fromJSON(
        object.chainRegistration
      );
    } else {
      message.chainRegistration = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetChainRegistrationResponse): unknown {
    const obj: any = {};
    message.chainRegistration !== undefined &&
      (obj.chainRegistration = message.chainRegistration
        ? ChainRegistration.toJSON(message.chainRegistration)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetChainRegistrationResponse>
  ): QueryGetChainRegistrationResponse {
    const message = {
      ...baseQueryGetChainRegistrationResponse,
    } as QueryGetChainRegistrationResponse;
    if (
      object.chainRegistration !== undefined &&
      object.chainRegistration !== null
    ) {
      message.chainRegistration = ChainRegistration.fromPartial(
        object.chainRegistration
      );
    } else {
      message.chainRegistration = undefined;
    }
    return message;
  },
};

const baseQueryAllChainRegistrationRequest: object = {};

export const QueryAllChainRegistrationRequest = {
  encode(
    message: QueryAllChainRegistrationRequest,
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
  ): QueryAllChainRegistrationRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllChainRegistrationRequest,
    } as QueryAllChainRegistrationRequest;
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

  fromJSON(object: any): QueryAllChainRegistrationRequest {
    const message = {
      ...baseQueryAllChainRegistrationRequest,
    } as QueryAllChainRegistrationRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllChainRegistrationRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllChainRegistrationRequest>
  ): QueryAllChainRegistrationRequest {
    const message = {
      ...baseQueryAllChainRegistrationRequest,
    } as QueryAllChainRegistrationRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllChainRegistrationResponse: object = {};

export const QueryAllChainRegistrationResponse = {
  encode(
    message: QueryAllChainRegistrationResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.chainRegistration) {
      ChainRegistration.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryAllChainRegistrationResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllChainRegistrationResponse,
    } as QueryAllChainRegistrationResponse;
    message.chainRegistration = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.chainRegistration.push(
            ChainRegistration.decode(reader, reader.uint32())
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

  fromJSON(object: any): QueryAllChainRegistrationResponse {
    const message = {
      ...baseQueryAllChainRegistrationResponse,
    } as QueryAllChainRegistrationResponse;
    message.chainRegistration = [];
    if (
      object.chainRegistration !== undefined &&
      object.chainRegistration !== null
    ) {
      for (const e of object.chainRegistration) {
        message.chainRegistration.push(ChainRegistration.fromJSON(e));
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllChainRegistrationResponse): unknown {
    const obj: any = {};
    if (message.chainRegistration) {
      obj.chainRegistration = message.chainRegistration.map((e) =>
        e ? ChainRegistration.toJSON(e) : undefined
      );
    } else {
      obj.chainRegistration = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllChainRegistrationResponse>
  ): QueryAllChainRegistrationResponse {
    const message = {
      ...baseQueryAllChainRegistrationResponse,
    } as QueryAllChainRegistrationResponse;
    message.chainRegistration = [];
    if (
      object.chainRegistration !== undefined &&
      object.chainRegistration !== null
    ) {
      for (const e of object.chainRegistration) {
        message.chainRegistration.push(ChainRegistration.fromPartial(e));
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

const baseQueryGetCoinMetaRollbackProtectionRequest: object = { index: "" };

export const QueryGetCoinMetaRollbackProtectionRequest = {
  encode(
    message: QueryGetCoinMetaRollbackProtectionRequest,
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
  ): QueryGetCoinMetaRollbackProtectionRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetCoinMetaRollbackProtectionRequest,
    } as QueryGetCoinMetaRollbackProtectionRequest;
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

  fromJSON(object: any): QueryGetCoinMetaRollbackProtectionRequest {
    const message = {
      ...baseQueryGetCoinMetaRollbackProtectionRequest,
    } as QueryGetCoinMetaRollbackProtectionRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = String(object.index);
    } else {
      message.index = "";
    }
    return message;
  },

  toJSON(message: QueryGetCoinMetaRollbackProtectionRequest): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetCoinMetaRollbackProtectionRequest>
  ): QueryGetCoinMetaRollbackProtectionRequest {
    const message = {
      ...baseQueryGetCoinMetaRollbackProtectionRequest,
    } as QueryGetCoinMetaRollbackProtectionRequest;
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = "";
    }
    return message;
  },
};

const baseQueryGetCoinMetaRollbackProtectionResponse: object = {};

export const QueryGetCoinMetaRollbackProtectionResponse = {
  encode(
    message: QueryGetCoinMetaRollbackProtectionResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.coinMetaRollbackProtection !== undefined) {
      CoinMetaRollbackProtection.encode(
        message.coinMetaRollbackProtection,
        writer.uint32(10).fork()
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): QueryGetCoinMetaRollbackProtectionResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryGetCoinMetaRollbackProtectionResponse,
    } as QueryGetCoinMetaRollbackProtectionResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.coinMetaRollbackProtection = CoinMetaRollbackProtection.decode(
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

  fromJSON(object: any): QueryGetCoinMetaRollbackProtectionResponse {
    const message = {
      ...baseQueryGetCoinMetaRollbackProtectionResponse,
    } as QueryGetCoinMetaRollbackProtectionResponse;
    if (
      object.coinMetaRollbackProtection !== undefined &&
      object.coinMetaRollbackProtection !== null
    ) {
      message.coinMetaRollbackProtection = CoinMetaRollbackProtection.fromJSON(
        object.coinMetaRollbackProtection
      );
    } else {
      message.coinMetaRollbackProtection = undefined;
    }
    return message;
  },

  toJSON(message: QueryGetCoinMetaRollbackProtectionResponse): unknown {
    const obj: any = {};
    message.coinMetaRollbackProtection !== undefined &&
      (obj.coinMetaRollbackProtection = message.coinMetaRollbackProtection
        ? CoinMetaRollbackProtection.toJSON(message.coinMetaRollbackProtection)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryGetCoinMetaRollbackProtectionResponse>
  ): QueryGetCoinMetaRollbackProtectionResponse {
    const message = {
      ...baseQueryGetCoinMetaRollbackProtectionResponse,
    } as QueryGetCoinMetaRollbackProtectionResponse;
    if (
      object.coinMetaRollbackProtection !== undefined &&
      object.coinMetaRollbackProtection !== null
    ) {
      message.coinMetaRollbackProtection = CoinMetaRollbackProtection.fromPartial(
        object.coinMetaRollbackProtection
      );
    } else {
      message.coinMetaRollbackProtection = undefined;
    }
    return message;
  },
};

const baseQueryAllCoinMetaRollbackProtectionRequest: object = {};

export const QueryAllCoinMetaRollbackProtectionRequest = {
  encode(
    message: QueryAllCoinMetaRollbackProtectionRequest,
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
  ): QueryAllCoinMetaRollbackProtectionRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllCoinMetaRollbackProtectionRequest,
    } as QueryAllCoinMetaRollbackProtectionRequest;
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

  fromJSON(object: any): QueryAllCoinMetaRollbackProtectionRequest {
    const message = {
      ...baseQueryAllCoinMetaRollbackProtectionRequest,
    } as QueryAllCoinMetaRollbackProtectionRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },

  toJSON(message: QueryAllCoinMetaRollbackProtectionRequest): unknown {
    const obj: any = {};
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageRequest.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllCoinMetaRollbackProtectionRequest>
  ): QueryAllCoinMetaRollbackProtectionRequest {
    const message = {
      ...baseQueryAllCoinMetaRollbackProtectionRequest,
    } as QueryAllCoinMetaRollbackProtectionRequest;
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination);
    } else {
      message.pagination = undefined;
    }
    return message;
  },
};

const baseQueryAllCoinMetaRollbackProtectionResponse: object = {};

export const QueryAllCoinMetaRollbackProtectionResponse = {
  encode(
    message: QueryAllCoinMetaRollbackProtectionResponse,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.coinMetaRollbackProtection) {
      CoinMetaRollbackProtection.encode(v!, writer.uint32(10).fork()).ldelim();
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
  ): QueryAllCoinMetaRollbackProtectionResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseQueryAllCoinMetaRollbackProtectionResponse,
    } as QueryAllCoinMetaRollbackProtectionResponse;
    message.coinMetaRollbackProtection = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.coinMetaRollbackProtection.push(
            CoinMetaRollbackProtection.decode(reader, reader.uint32())
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

  fromJSON(object: any): QueryAllCoinMetaRollbackProtectionResponse {
    const message = {
      ...baseQueryAllCoinMetaRollbackProtectionResponse,
    } as QueryAllCoinMetaRollbackProtectionResponse;
    message.coinMetaRollbackProtection = [];
    if (
      object.coinMetaRollbackProtection !== undefined &&
      object.coinMetaRollbackProtection !== null
    ) {
      for (const e of object.coinMetaRollbackProtection) {
        message.coinMetaRollbackProtection.push(
          CoinMetaRollbackProtection.fromJSON(e)
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

  toJSON(message: QueryAllCoinMetaRollbackProtectionResponse): unknown {
    const obj: any = {};
    if (message.coinMetaRollbackProtection) {
      obj.coinMetaRollbackProtection = message.coinMetaRollbackProtection.map(
        (e) => (e ? CoinMetaRollbackProtection.toJSON(e) : undefined)
      );
    } else {
      obj.coinMetaRollbackProtection = [];
    }
    message.pagination !== undefined &&
      (obj.pagination = message.pagination
        ? PageResponse.toJSON(message.pagination)
        : undefined);
    return obj;
  },

  fromPartial(
    object: DeepPartial<QueryAllCoinMetaRollbackProtectionResponse>
  ): QueryAllCoinMetaRollbackProtectionResponse {
    const message = {
      ...baseQueryAllCoinMetaRollbackProtectionResponse,
    } as QueryAllCoinMetaRollbackProtectionResponse;
    message.coinMetaRollbackProtection = [];
    if (
      object.coinMetaRollbackProtection !== undefined &&
      object.coinMetaRollbackProtection !== null
    ) {
      for (const e of object.coinMetaRollbackProtection) {
        message.coinMetaRollbackProtection.push(
          CoinMetaRollbackProtection.fromPartial(e)
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
  /** Queries a chainRegistration by index. */
  ChainRegistration(
    request: QueryGetChainRegistrationRequest
  ): Promise<QueryGetChainRegistrationResponse>;
  /** Queries a list of chainRegistration items. */
  ChainRegistrationAll(
    request: QueryAllChainRegistrationRequest
  ): Promise<QueryAllChainRegistrationResponse>;
  /** Queries a coinMetaRollbackProtection by index. */
  CoinMetaRollbackProtection(
    request: QueryGetCoinMetaRollbackProtectionRequest
  ): Promise<QueryGetCoinMetaRollbackProtectionResponse>;
  /** Queries a list of coinMetaRollbackProtection items. */
  CoinMetaRollbackProtectionAll(
    request: QueryAllCoinMetaRollbackProtectionRequest
  ): Promise<QueryAllCoinMetaRollbackProtectionResponse>;
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  Config(request: QueryGetConfigRequest): Promise<QueryGetConfigResponse> {
    const data = QueryGetConfigRequest.encode(request).finish();
    const promise = this.rpc.request(
      "certusone.wormholechain.tokenbridge.Query",
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
      "certusone.wormholechain.tokenbridge.Query",
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
      "certusone.wormholechain.tokenbridge.Query",
      "ReplayProtectionAll",
      data
    );
    return promise.then((data) =>
      QueryAllReplayProtectionResponse.decode(new Reader(data))
    );
  }

  ChainRegistration(
    request: QueryGetChainRegistrationRequest
  ): Promise<QueryGetChainRegistrationResponse> {
    const data = QueryGetChainRegistrationRequest.encode(request).finish();
    const promise = this.rpc.request(
      "certusone.wormholechain.tokenbridge.Query",
      "ChainRegistration",
      data
    );
    return promise.then((data) =>
      QueryGetChainRegistrationResponse.decode(new Reader(data))
    );
  }

  ChainRegistrationAll(
    request: QueryAllChainRegistrationRequest
  ): Promise<QueryAllChainRegistrationResponse> {
    const data = QueryAllChainRegistrationRequest.encode(request).finish();
    const promise = this.rpc.request(
      "certusone.wormholechain.tokenbridge.Query",
      "ChainRegistrationAll",
      data
    );
    return promise.then((data) =>
      QueryAllChainRegistrationResponse.decode(new Reader(data))
    );
  }

  CoinMetaRollbackProtection(
    request: QueryGetCoinMetaRollbackProtectionRequest
  ): Promise<QueryGetCoinMetaRollbackProtectionResponse> {
    const data = QueryGetCoinMetaRollbackProtectionRequest.encode(
      request
    ).finish();
    const promise = this.rpc.request(
      "certusone.wormholechain.tokenbridge.Query",
      "CoinMetaRollbackProtection",
      data
    );
    return promise.then((data) =>
      QueryGetCoinMetaRollbackProtectionResponse.decode(new Reader(data))
    );
  }

  CoinMetaRollbackProtectionAll(
    request: QueryAllCoinMetaRollbackProtectionRequest
  ): Promise<QueryAllCoinMetaRollbackProtectionResponse> {
    const data = QueryAllCoinMetaRollbackProtectionRequest.encode(
      request
    ).finish();
    const promise = this.rpc.request(
      "certusone.wormholechain.tokenbridge.Query",
      "CoinMetaRollbackProtectionAll",
      data
    );
    return promise.then((data) =>
      QueryAllCoinMetaRollbackProtectionResponse.decode(new Reader(data))
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
