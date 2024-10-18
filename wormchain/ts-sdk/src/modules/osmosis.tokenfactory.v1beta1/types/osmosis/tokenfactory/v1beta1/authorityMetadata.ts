//@ts-nocheck
/* eslint-disable */
import _m0 from "protobufjs/minimal";

export const protobufPackage = "osmosis.tokenfactory.v1beta1";

/**
 * DenomAuthorityMetadata specifies metadata for addresses that have specific
 * capabilities over a token factory denom. Right now there is only one Admin
 * permission, but is planned to be extended to the future.
 */
export interface DenomAuthorityMetadata {
  /** Can be empty for no admin, or a valid osmosis address */
  admin: string;
}

function createBaseDenomAuthorityMetadata(): DenomAuthorityMetadata {
  return { admin: "" };
}

export const DenomAuthorityMetadata = {
  encode(message: DenomAuthorityMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.admin !== "") {
      writer.uint32(10).string(message.admin);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DenomAuthorityMetadata {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDenomAuthorityMetadata();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.admin = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DenomAuthorityMetadata {
    return { admin: isSet(object.admin) ? String(object.admin) : "" };
  },

  toJSON(message: DenomAuthorityMetadata): unknown {
    const obj: any = {};
    message.admin !== undefined && (obj.admin = message.admin);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DenomAuthorityMetadata>, I>>(object: I): DenomAuthorityMetadata {
    const message = createBaseDenomAuthorityMetadata();
    message.admin = object.admin ?? "";
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
