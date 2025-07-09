//@ts-nocheck
/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "cosmos.base.query.v1beta1";

/**
 * PageRequest is to be embedded in gRPC request messages for efficient
 * pagination. Ex:
 *
 *  message SomeRequest {
 *          Foo some_parameter = 1;
 *          PageRequest pagination = 2;
 *  }
 */
export interface PageRequest {
  /**
   * key is a value returned in PageResponse.next_key to begin
   * querying the next page most efficiently. Only one of offset or key
   * should be set.
   */
  key: Uint8Array;
  /**
   * offset is a numeric offset that can be used when key is unavailable.
   * It is less efficient than using key. Only one of offset or key should
   * be set.
   */
  offset: number;
  /**
   * limit is the total number of results to be returned in the result page.
   * If left empty it will default to a value to be set by each app.
   */
  limit: number;
  /**
   * count_total is set to true  to indicate that the result set should include
   * a count of the total number of items available for pagination in UIs.
   * count_total is only respected when offset is used. It is ignored when key
   * is set.
   */
  countTotal: boolean;
  /**
   * reverse is set to true if results are to be returned in the descending order.
   *
   * Since: cosmos-sdk 0.43
   */
  reverse: boolean;
}

/**
 * PageResponse is to be embedded in gRPC response messages where the
 * corresponding request message has used PageRequest.
 *
 *  message SomeResponse {
 *          repeated Bar results = 1;
 *          PageResponse page = 2;
 *  }
 */
export interface PageResponse {
  /**
   * next_key is the key to be passed to PageRequest.key to
   * query the next page most efficiently. It will be empty if
   * there are no more results.
   */
  nextKey: Uint8Array;
  /**
   * total is total number of results available if PageRequest.count_total
   * was set, its value is undefined otherwise
   */
  total: number;
}

function createBasePageRequest(): PageRequest {
  return { key: new Uint8Array(), offset: 0, limit: 0, countTotal: false, reverse: false };
}

export const PageRequest = {
  encode(message: PageRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key.length !== 0) {
      writer.uint32(10).bytes(message.key);
    }
    if (message.offset !== 0) {
      writer.uint32(16).uint64(message.offset);
    }
    if (message.limit !== 0) {
      writer.uint32(24).uint64(message.limit);
    }
    if (message.countTotal === true) {
      writer.uint32(32).bool(message.countTotal);
    }
    if (message.reverse === true) {
      writer.uint32(40).bool(message.reverse);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PageRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePageRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.bytes();
          break;
        case 2:
          message.offset = longToNumber(reader.uint64() as Long);
          break;
        case 3:
          message.limit = longToNumber(reader.uint64() as Long);
          break;
        case 4:
          message.countTotal = reader.bool();
          break;
        case 5:
          message.reverse = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PageRequest {
    return {
      key: isSet(object.key) ? bytesFromBase64(object.key) : new Uint8Array(),
      offset: isSet(object.offset) ? Number(object.offset) : 0,
      limit: isSet(object.limit) ? Number(object.limit) : 0,
      countTotal: isSet(object.countTotal) ? Boolean(object.countTotal) : false,
      reverse: isSet(object.reverse) ? Boolean(object.reverse) : false,
    };
  },

  toJSON(message: PageRequest): unknown {
    const obj: any = {};
    message.key !== undefined
      && (obj.key = base64FromBytes(message.key !== undefined ? message.key : new Uint8Array()));
    message.offset !== undefined && (obj.offset = Math.round(message.offset));
    message.limit !== undefined && (obj.limit = Math.round(message.limit));
    message.countTotal !== undefined && (obj.countTotal = message.countTotal);
    message.reverse !== undefined && (obj.reverse = message.reverse);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PageRequest>, I>>(object: I): PageRequest {
    const message = createBasePageRequest();
    message.key = object.key ?? new Uint8Array();
    message.offset = object.offset ?? 0;
    message.limit = object.limit ?? 0;
    message.countTotal = object.countTotal ?? false;
    message.reverse = object.reverse ?? false;
    return message;
  },
};

function createBasePageResponse(): PageResponse {
  return { nextKey: new Uint8Array(), total: 0 };
}

export const PageResponse = {
  encode(message: PageResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.nextKey.length !== 0) {
      writer.uint32(10).bytes(message.nextKey);
    }
    if (message.total !== 0) {
      writer.uint32(16).uint64(message.total);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PageResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePageResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.nextKey = reader.bytes();
          break;
        case 2:
          message.total = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PageResponse {
    return {
      nextKey: isSet(object.nextKey) ? bytesFromBase64(object.nextKey) : new Uint8Array(),
      total: isSet(object.total) ? Number(object.total) : 0,
    };
  },

  toJSON(message: PageResponse): unknown {
    const obj: any = {};
    message.nextKey !== undefined
      && (obj.nextKey = base64FromBytes(message.nextKey !== undefined ? message.nextKey : new Uint8Array()));
    message.total !== undefined && (obj.total = Math.round(message.total));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PageResponse>, I>>(object: I): PageResponse {
    const message = createBasePageResponse();
    message.nextKey = object.nextKey ?? new Uint8Array();
    message.total = object.total ?? 0;
    return message;
  },
};

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

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
