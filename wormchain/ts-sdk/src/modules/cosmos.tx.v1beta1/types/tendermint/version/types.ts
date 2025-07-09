//@ts-nocheck
/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "tendermint.version";

/**
 * App includes the protocol and software version for the application.
 * This information is included in ResponseInfo. The App.Protocol can be
 * updated in ResponseEndBlock.
 */
export interface App {
  protocol: number;
  software: string;
}

/**
 * Consensus captures the consensus rules for processing a block in the blockchain,
 * including all blockchain data structures and the rules of the application's
 * state transition machine.
 */
export interface Consensus {
  block: number;
  app: number;
}

function createBaseApp(): App {
  return { protocol: 0, software: "" };
}

export const App = {
  encode(message: App, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.protocol !== 0) {
      writer.uint32(8).uint64(message.protocol);
    }
    if (message.software !== "") {
      writer.uint32(18).string(message.software);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): App {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApp();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.protocol = longToNumber(reader.uint64() as Long);
          break;
        case 2:
          message.software = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): App {
    return {
      protocol: isSet(object.protocol) ? Number(object.protocol) : 0,
      software: isSet(object.software) ? String(object.software) : "",
    };
  },

  toJSON(message: App): unknown {
    const obj: any = {};
    message.protocol !== undefined && (obj.protocol = Math.round(message.protocol));
    message.software !== undefined && (obj.software = message.software);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<App>, I>>(object: I): App {
    const message = createBaseApp();
    message.protocol = object.protocol ?? 0;
    message.software = object.software ?? "";
    return message;
  },
};

function createBaseConsensus(): Consensus {
  return { block: 0, app: 0 };
}

export const Consensus = {
  encode(message: Consensus, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.block !== 0) {
      writer.uint32(8).uint64(message.block);
    }
    if (message.app !== 0) {
      writer.uint32(16).uint64(message.app);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Consensus {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseConsensus();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.block = longToNumber(reader.uint64() as Long);
          break;
        case 2:
          message.app = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Consensus {
    return { block: isSet(object.block) ? Number(object.block) : 0, app: isSet(object.app) ? Number(object.app) : 0 };
  },

  toJSON(message: Consensus): unknown {
    const obj: any = {};
    message.block !== undefined && (obj.block = Math.round(message.block));
    message.app !== undefined && (obj.app = Math.round(message.app));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Consensus>, I>>(object: I): Consensus {
    const message = createBaseConsensus();
    message.block = object.block ?? 0;
    message.app = object.app ?? 0;
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
