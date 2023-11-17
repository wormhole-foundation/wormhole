//@ts-nocheck
/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "wormhole_foundation.wormchain.wormhole";

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
  validator_address: string;
  /** the allowlisted account */
  allowed_address: string;
  /** human readable name */
  name: string;
}

export interface WasmInstantiateAllowedContractCodeId {
  /** bech32 address of the contract that can call wasm instantiate without a VAA */
  contract_address: string;
  /** reference to the stored WASM code that can be instantiated */
  code_id: number;
}

export interface IbcComposabilityMwContract {
  /** bech32 address of the contract that is used by the ibc composability middleware */
  contract_address: string;
}

const baseGuardianKey: object = {};

export const GuardianKey = {
  encode(message: GuardianKey, writer: Writer = Writer.create()): Writer {
    if (message.key.length !== 0) {
      writer.uint32(10).bytes(message.key);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GuardianKey {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGuardianKey } as GuardianKey;
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
    const message = { ...baseGuardianKey } as GuardianKey;
    if (object.key !== undefined && object.key !== null) {
      message.key = bytesFromBase64(object.key);
    }
    return message;
  },

  toJSON(message: GuardianKey): unknown {
    const obj: any = {};
    message.key !== undefined &&
      (obj.key = base64FromBytes(
        message.key !== undefined ? message.key : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(object: DeepPartial<GuardianKey>): GuardianKey {
    const message = { ...baseGuardianKey } as GuardianKey;
    if (object.key !== undefined && object.key !== null) {
      message.key = object.key;
    } else {
      message.key = new Uint8Array();
    }
    return message;
  },
};

const baseGuardianValidator: object = {};

export const GuardianValidator = {
  encode(message: GuardianValidator, writer: Writer = Writer.create()): Writer {
    if (message.guardianKey.length !== 0) {
      writer.uint32(10).bytes(message.guardianKey);
    }
    if (message.validatorAddr.length !== 0) {
      writer.uint32(18).bytes(message.validatorAddr);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): GuardianValidator {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGuardianValidator } as GuardianValidator;
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
    const message = { ...baseGuardianValidator } as GuardianValidator;
    if (object.guardianKey !== undefined && object.guardianKey !== null) {
      message.guardianKey = bytesFromBase64(object.guardianKey);
    }
    if (object.validatorAddr !== undefined && object.validatorAddr !== null) {
      message.validatorAddr = bytesFromBase64(object.validatorAddr);
    }
    return message;
  },

  toJSON(message: GuardianValidator): unknown {
    const obj: any = {};
    message.guardianKey !== undefined &&
      (obj.guardianKey = base64FromBytes(
        message.guardianKey !== undefined
          ? message.guardianKey
          : new Uint8Array()
      ));
    message.validatorAddr !== undefined &&
      (obj.validatorAddr = base64FromBytes(
        message.validatorAddr !== undefined
          ? message.validatorAddr
          : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(object: DeepPartial<GuardianValidator>): GuardianValidator {
    const message = { ...baseGuardianValidator } as GuardianValidator;
    if (object.guardianKey !== undefined && object.guardianKey !== null) {
      message.guardianKey = object.guardianKey;
    } else {
      message.guardianKey = new Uint8Array();
    }
    if (object.validatorAddr !== undefined && object.validatorAddr !== null) {
      message.validatorAddr = object.validatorAddr;
    } else {
      message.validatorAddr = new Uint8Array();
    }
    return message;
  },
};

const baseGuardianSet: object = { index: 0, expirationTime: 0 };

export const GuardianSet = {
  encode(message: GuardianSet, writer: Writer = Writer.create()): Writer {
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

  decode(input: Reader | Uint8Array, length?: number): GuardianSet {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseGuardianSet } as GuardianSet;
    message.keys = [];
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
    const message = { ...baseGuardianSet } as GuardianSet;
    message.keys = [];
    if (object.index !== undefined && object.index !== null) {
      message.index = Number(object.index);
    } else {
      message.index = 0;
    }
    if (object.keys !== undefined && object.keys !== null) {
      for (const e of object.keys) {
        message.keys.push(bytesFromBase64(e));
      }
    }
    if (object.expirationTime !== undefined && object.expirationTime !== null) {
      message.expirationTime = Number(object.expirationTime);
    } else {
      message.expirationTime = 0;
    }
    return message;
  },

  toJSON(message: GuardianSet): unknown {
    const obj: any = {};
    message.index !== undefined && (obj.index = message.index);
    if (message.keys) {
      obj.keys = message.keys.map((e) =>
        base64FromBytes(e !== undefined ? e : new Uint8Array())
      );
    } else {
      obj.keys = [];
    }
    message.expirationTime !== undefined &&
      (obj.expirationTime = message.expirationTime);
    return obj;
  },

  fromPartial(object: DeepPartial<GuardianSet>): GuardianSet {
    const message = { ...baseGuardianSet } as GuardianSet;
    message.keys = [];
    if (object.index !== undefined && object.index !== null) {
      message.index = object.index;
    } else {
      message.index = 0;
    }
    if (object.keys !== undefined && object.keys !== null) {
      for (const e of object.keys) {
        message.keys.push(e);
      }
    }
    if (object.expirationTime !== undefined && object.expirationTime !== null) {
      message.expirationTime = object.expirationTime;
    } else {
      message.expirationTime = 0;
    }
    return message;
  },
};

const baseValidatorAllowedAddress: object = {
  validator_address: "",
  allowed_address: "",
  name: "",
};

export const ValidatorAllowedAddress = {
  encode(
    message: ValidatorAllowedAddress,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.validator_address !== "") {
      writer.uint32(10).string(message.validator_address);
    }
    if (message.allowed_address !== "") {
      writer.uint32(18).string(message.allowed_address);
    }
    if (message.name !== "") {
      writer.uint32(26).string(message.name);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): ValidatorAllowedAddress {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseValidatorAllowedAddress,
    } as ValidatorAllowedAddress;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator_address = reader.string();
          break;
        case 2:
          message.allowed_address = reader.string();
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
    const message = {
      ...baseValidatorAllowedAddress,
    } as ValidatorAllowedAddress;
    if (
      object.validator_address !== undefined &&
      object.validator_address !== null
    ) {
      message.validator_address = String(object.validator_address);
    } else {
      message.validator_address = "";
    }
    if (
      object.allowed_address !== undefined &&
      object.allowed_address !== null
    ) {
      message.allowed_address = String(object.allowed_address);
    } else {
      message.allowed_address = "";
    }
    if (object.name !== undefined && object.name !== null) {
      message.name = String(object.name);
    } else {
      message.name = "";
    }
    return message;
  },

  toJSON(message: ValidatorAllowedAddress): unknown {
    const obj: any = {};
    message.validator_address !== undefined &&
      (obj.validator_address = message.validator_address);
    message.allowed_address !== undefined &&
      (obj.allowed_address = message.allowed_address);
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(
    object: DeepPartial<ValidatorAllowedAddress>
  ): ValidatorAllowedAddress {
    const message = {
      ...baseValidatorAllowedAddress,
    } as ValidatorAllowedAddress;
    if (
      object.validator_address !== undefined &&
      object.validator_address !== null
    ) {
      message.validator_address = object.validator_address;
    } else {
      message.validator_address = "";
    }
    if (
      object.allowed_address !== undefined &&
      object.allowed_address !== null
    ) {
      message.allowed_address = object.allowed_address;
    } else {
      message.allowed_address = "";
    }
    if (object.name !== undefined && object.name !== null) {
      message.name = object.name;
    } else {
      message.name = "";
    }
    return message;
  },
};

const baseWasmInstantiateAllowedContractCodeId: object = {
  contract_address: "",
  code_id: 0,
};

export const WasmInstantiateAllowedContractCodeId = {
  encode(
    message: WasmInstantiateAllowedContractCodeId,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.contract_address !== "") {
      writer.uint32(10).string(message.contract_address);
    }
    if (message.code_id !== 0) {
      writer.uint32(16).uint64(message.code_id);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): WasmInstantiateAllowedContractCodeId {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseWasmInstantiateAllowedContractCodeId,
    } as WasmInstantiateAllowedContractCodeId;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.contract_address = reader.string();
          break;
        case 2:
          message.code_id = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): WasmInstantiateAllowedContractCodeId {
    const message = {
      ...baseWasmInstantiateAllowedContractCodeId,
    } as WasmInstantiateAllowedContractCodeId;
    if (
      object.contract_address !== undefined &&
      object.contract_address !== null
    ) {
      message.contract_address = String(object.contract_address);
    } else {
      message.contract_address = "";
    }
    if (object.code_id !== undefined && object.code_id !== null) {
      message.code_id = Number(object.code_id);
    } else {
      message.code_id = 0;
    }
    return message;
  },

  toJSON(message: WasmInstantiateAllowedContractCodeId): unknown {
    const obj: any = {};
    message.contract_address !== undefined &&
      (obj.contract_address = message.contract_address);
    message.code_id !== undefined && (obj.code_id = message.code_id);
    return obj;
  },

  fromPartial(
    object: DeepPartial<WasmInstantiateAllowedContractCodeId>
  ): WasmInstantiateAllowedContractCodeId {
    const message = {
      ...baseWasmInstantiateAllowedContractCodeId,
    } as WasmInstantiateAllowedContractCodeId;
    if (
      object.contract_address !== undefined &&
      object.contract_address !== null
    ) {
      message.contract_address = object.contract_address;
    } else {
      message.contract_address = "";
    }
    if (object.code_id !== undefined && object.code_id !== null) {
      message.code_id = object.code_id;
    } else {
      message.code_id = 0;
    }
    return message;
  },
};

const baseIbcComposabilityMwContract: object = { contract_address: "" };

export const IbcComposabilityMwContract = {
  encode(
    message: IbcComposabilityMwContract,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.contract_address !== "") {
      writer.uint32(10).string(message.contract_address);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): IbcComposabilityMwContract {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseIbcComposabilityMwContract,
    } as IbcComposabilityMwContract;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.contract_address = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): IbcComposabilityMwContract {
    const message = {
      ...baseIbcComposabilityMwContract,
    } as IbcComposabilityMwContract;
    if (
      object.contract_address !== undefined &&
      object.contract_address !== null
    ) {
      message.contract_address = String(object.contract_address);
    } else {
      message.contract_address = "";
    }
    return message;
  },

  toJSON(message: IbcComposabilityMwContract): unknown {
    const obj: any = {};
    message.contract_address !== undefined &&
      (obj.contract_address = message.contract_address);
    return obj;
  },

  fromPartial(
    object: DeepPartial<IbcComposabilityMwContract>
  ): IbcComposabilityMwContract {
    const message = {
      ...baseIbcComposabilityMwContract,
    } as IbcComposabilityMwContract;
    if (
      object.contract_address !== undefined &&
      object.contract_address !== null
    ) {
      message.contract_address = object.contract_address;
    } else {
      message.contract_address = "";
    }
    return message;
  },
};

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

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

if (util.Long !== Long) {
  util.Long = Long as any;
  configure();
}
