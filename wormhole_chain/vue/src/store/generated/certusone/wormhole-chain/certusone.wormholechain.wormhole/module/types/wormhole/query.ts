/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { GuardianSet } from "../wormhole/guardian_set";
import {
  PageRequest,
  PageResponse,
} from "../cosmos/base/query/v1beta1/pagination";
import { Config } from "../wormhole/config";

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
