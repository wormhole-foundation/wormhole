//@ts-nocheck
/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "wormchain.wormhole";

export interface GuardianKey {
  key: Uint8Array;
}

export interface GuardianValidator {
  guardianKey: Uint8Array;
  validatorAddr: Uint8Array;
}

export interface GuardianSet {
  index: number;
  keys: Uint8Array[];
  expirationTime: number;
}

export interface ValidatorAllowedAddress {
  /** the validator/guardian that controls this entry */
  validatorAddress: string;
  /** the allowlisted account */
  allowedAddress: string;
  /** human readable name */
  name: string;
}

export interface WasmInstantiateAllowedContractCodeId {
  /** bech32 address of the contract that can call wasm instantiate without a VAA */
  contractAddress: string;
  /** reference to the stored WASM code that can be instantiated */
  codeId: number;
}

export interface IbcComposabilityMwContract {
  /**
   * bech32 address of the contract that is used by the ibc composability
   * middleware
   */
  contractAddress: string;
}

function createBaseGuardianKey(): GuardianKey {
  return { key: new Uint8Array() };
}

export const GuardianKey = {
  encode(message: GuardianKey, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key.length !== 0) {
      writer.uint32(10).bytes(message.key);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GuardianKey {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGuardianKey();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GuardianKey {
    return { key: isSet(object.key) ? bytesFromBase64(object.key) : new Uint8Array() };
  },

  toJSON(message: GuardianKey): unknown {
    const obj: any = {};
    message.key !== undefined
      && (obj.key = base64FromBytes(message.key !== undefined ? message.key : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GuardianKey>, I>>(object: I): GuardianKey {
    const message = createBaseGuardianKey();
    message.key = object.key ?? new Uint8Array();
    return message;
  },
};

function createBaseGuardianValidator(): GuardianValidator {
  return { guardianKey: new Uint8Array(), validatorAddr: new Uint8Array() };
}

export const GuardianValidator = {
  encode(message: GuardianValidator, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.guardianKey.length !== 0) {
      writer.uint32(10).bytes(message.guardianKey);
    }
    if (message.validatorAddr.length !== 0) {
      writer.uint32(18).bytes(message.validatorAddr);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GuardianValidator {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGuardianValidator();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.guardianKey = reader.bytes();
          break;
        case 2:
          message.validatorAddr = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GuardianValidator {
    return {
      guardianKey: isSet(object.guardianKey) ? bytesFromBase64(object.guardianKey) : new Uint8Array(),
      validatorAddr: isSet(object.validatorAddr) ? bytesFromBase64(object.validatorAddr) : new Uint8Array(),
    };
  },

  toJSON(message: GuardianValidator): unknown {
    const obj: any = {};
    message.guardianKey !== undefined
      && (obj.guardianKey = base64FromBytes(
        message.guardianKey !== undefined ? message.guardianKey : new Uint8Array(),
      ));
    message.validatorAddr !== undefined
      && (obj.validatorAddr = base64FromBytes(
        message.validatorAddr !== undefined ? message.validatorAddr : new Uint8Array(),
      ));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GuardianValidator>, I>>(object: I): GuardianValidator {
    const message = createBaseGuardianValidator();
    message.guardianKey = object.guardianKey ?? new Uint8Array();
    message.validatorAddr = object.validatorAddr ?? new Uint8Array();
    return message;
  },
};

function createBaseGuardianSet(): GuardianSet {
  return { index: 0, keys: [], expirationTime: 0 };
}

export const GuardianSet = {
  encode(message: GuardianSet, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.index !== 0) {
      writer.uint32(8).uint32(message.index);
    }
    for (const v of message.keys) {
      writer.uint32(18).bytes(v!);
    }
    if (message.expirationTime !== 0) {
      writer.uint32(24).uint64(message.expirationTime);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GuardianSet {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGuardianSet();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.index = reader.uint32();
          break;
        case 2:
          message.keys.push(reader.bytes());
          break;
        case 3:
          message.expirationTime = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GuardianSet {
    return {
      index: isSet(object.index) ? Number(object.index) : 0,
      keys: Array.isArray(object?.keys) ? object.keys.map((e: any) => bytesFromBase64(e)) : [],
      expirationTime: isSet(object.expirationTime) ? Number(object.expirationTime) : 0,
    };
  },

  toJSON(message: GuardianSet): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = Math.round(message.index));
    if (message.keys) {
      obj.keys = message.keys.map((e) => base64FromBytes(e !== undefined ? e : new Uint8Array()));
    } else {
      obj.keys = [];
    }
    message.expirationTime !== undefined && (obj.expirationTime = Math.round(message.expirationTime));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GuardianSet>, I>>(object: I): GuardianSet {
    const message = createBaseGuardianSet();
    message.index = object.index ?? 0;
    message.keys = object.keys?.map((e) => e) || [];
    message.expirationTime = object.expirationTime ?? 0;
    return message;
  },
};

function createBaseValidatorAllowedAddress(): ValidatorAllowedAddress {
  return { validatorAddress: "", allowedAddress: "", name: "" };
}

export const ValidatorAllowedAddress = {
  encode(message: ValidatorAllowedAddress, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.validatorAddress !== "") {
      writer.uint32(10).string(message.validatorAddress);
    }
    if (message.allowedAddress !== "") {
      writer.uint32(18).string(message.allowedAddress);
    }
    if (message.name !== "") {
      writer.uint32(26).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ValidatorAllowedAddress {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseValidatorAllowedAddress();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validatorAddress = reader.string();
          break;
        case 2:
          message.allowedAddress = reader.string();
          break;
        case 3:
          message.name = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ValidatorAllowedAddress {
    return {
      validatorAddress: isSet(object.validatorAddress) ? String(object.validatorAddress) : "",
      allowedAddress: isSet(object.allowedAddress) ? String(object.allowedAddress) : "",
      name: isSet(object.name) ? String(object.name) : "",
    };
  },

  toJSON(message: ValidatorAllowedAddress): unknown {
    const obj: any = {};
    message.validatorAddress !== undefined && (obj.validatorAddress = message.validatorAddress);
    message.allowedAddress !== undefined && (obj.allowedAddress = message.allowedAddress);
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ValidatorAllowedAddress>, I>>(object: I): ValidatorAllowedAddress {
    const message = createBaseValidatorAllowedAddress();
    message.validatorAddress = object.validatorAddress ?? "";
    message.allowedAddress = object.allowedAddress ?? "";
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseWasmInstantiateAllowedContractCodeId(): WasmInstantiateAllowedContractCodeId {
  return { contractAddress: "", codeId: 0 };
}

export const WasmInstantiateAllowedContractCodeId = {
  encode(message: WasmInstantiateAllowedContractCodeId, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.contractAddress !== "") {
      writer.uint32(10).string(message.contractAddress);
    }
    if (message.codeId !== 0) {
      writer.uint32(16).uint64(message.codeId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): WasmInstantiateAllowedContractCodeId {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseWasmInstantiateAllowedContractCodeId();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.contractAddress = reader.string();
          break;
        case 2:
          message.codeId = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): WasmInstantiateAllowedContractCodeId {
    return {
      contractAddress: isSet(object.contractAddress) ? String(object.contractAddress) : "",
      codeId: isSet(object.codeId) ? Number(object.codeId) : 0,
    };
  },

  toJSON(message: WasmInstantiateAllowedContractCodeId): unknown {
    const obj: any = {};
    message.contractAddress !== undefined && (obj.contractAddress = message.contractAddress);
    message.codeId !== undefined && (obj.codeId = Math.round(message.codeId));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<WasmInstantiateAllowedContractCodeId>, I>>(
    object: I,
  ): WasmInstantiateAllowedContractCodeId {
    const message = createBaseWasmInstantiateAllowedContractCodeId();
    message.contractAddress = object.contractAddress ?? "";
    message.codeId = object.codeId ?? 0;
    return message;
  },
};

function createBaseIbcComposabilityMwContract(): IbcComposabilityMwContract {
  return { contractAddress: "" };
}

export const IbcComposabilityMwContract = {
  encode(message: IbcComposabilityMwContract, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.contractAddress !== "") {
      writer.uint32(10).string(message.contractAddress);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IbcComposabilityMwContract {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIbcComposabilityMwContract();
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

  fromJSON(object: any): IbcComposabilityMwContract {
    return { contractAddress: isSet(object.contractAddress) ? String(object.contractAddress) : "" };
  },

  toJSON(message: IbcComposabilityMwContract): unknown {
    const obj: any = {};
    message.contractAddress !== undefined && (obj.contractAddress = message.contractAddress);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<IbcComposabilityMwContract>, I>>(object: I): IbcComposabilityMwContract {
    const message = createBaseIbcComposabilityMwContract();
    message.contractAddress = object.contractAddress ?? "";
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
