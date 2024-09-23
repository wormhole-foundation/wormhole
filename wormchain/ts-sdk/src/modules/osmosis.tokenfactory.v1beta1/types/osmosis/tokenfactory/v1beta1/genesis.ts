//@ts-nocheck
/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { DenomAuthorityMetadata } from "./authorityMetadata";
import { Params } from "./params";

export const protobufPackage = "osmosis.tokenfactory.v1beta1";

/** GenesisState defines the tokenfactory module's genesis state. */
export interface GenesisState {
  /** params defines the paramaters of the module. */
  params: Params | undefined;
  factoryDenoms: GenesisDenom[];
}

/**
 * GenesisDenom defines a tokenfactory denom that is defined within genesis
 * state. The structure contains DenomAuthorityMetadata which defines the
 * denom's admin.
 */
export interface GenesisDenom {
  denom: string;
  authorityMetadata: DenomAuthorityMetadata | undefined;
}

function createBaseGenesisState(): GenesisState {
  return { params: undefined, factoryDenoms: [] };
}

export const GenesisState = {
  encode(message: GenesisState, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.factoryDenoms) {
      GenesisDenom.encode(v!, writer.uint32(18).fork()).ldelim();
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
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.factoryDenoms.push(GenesisDenom.decode(reader, reader.uint32()));
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
      params: isSet(object.params) ? Params.fromJSON(object.params) : undefined,
      factoryDenoms: Array.isArray(object?.factoryDenoms)
        ? object.factoryDenoms.map((e: any) => GenesisDenom.fromJSON(e))
        : [],
    };
  },

  toJSON(message: GenesisState): unknown {
    const obj: any = {};
    message.params !== undefined && (obj.params = message.params ? Params.toJSON(message.params) : undefined);
    if (message.factoryDenoms) {
      obj.factoryDenoms = message.factoryDenoms.map((e) => e ? GenesisDenom.toJSON(e) : undefined);
    } else {
      obj.factoryDenoms = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GenesisState>, I>>(object: I): GenesisState {
    const message = createBaseGenesisState();
    message.params = (object.params !== undefined && object.params !== null)
      ? Params.fromPartial(object.params)
      : undefined;
    message.factoryDenoms = object.factoryDenoms?.map((e) => GenesisDenom.fromPartial(e)) || [];
    return message;
  },
};

function createBaseGenesisDenom(): GenesisDenom {
  return { denom: "", authorityMetadata: undefined };
}

export const GenesisDenom = {
  encode(message: GenesisDenom, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.denom !== "") {
      writer.uint32(10).string(message.denom);
    }
    if (message.authorityMetadata !== undefined) {
      DenomAuthorityMetadata.encode(message.authorityMetadata, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GenesisDenom {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisDenom();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.denom = reader.string();
          break;
        case 2:
          message.authorityMetadata = DenomAuthorityMetadata.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GenesisDenom {
    return {
      denom: isSet(object.denom) ? String(object.denom) : "",
      authorityMetadata: isSet(object.authorityMetadata)
        ? DenomAuthorityMetadata.fromJSON(object.authorityMetadata)
        : undefined,
    };
  },

  toJSON(message: GenesisDenom): unknown {
    const obj: any = {};
    message.denom !== undefined && (obj.denom = message.denom);
    message.authorityMetadata !== undefined && (obj.authorityMetadata = message.authorityMetadata
      ? DenomAuthorityMetadata.toJSON(message.authorityMetadata)
      : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GenesisDenom>, I>>(object: I): GenesisDenom {
    const message = createBaseGenesisDenom();
    message.denom = object.denom ?? "";
    message.authorityMetadata = (object.authorityMetadata !== undefined && object.authorityMetadata !== null)
      ? DenomAuthorityMetadata.fromPartial(object.authorityMetadata)
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
