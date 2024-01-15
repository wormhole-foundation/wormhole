//@ts-nocheck
/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
import { Any } from "../../../google/protobuf/any";
import { Coin } from "../../../cosmos/base/v1beta1/coin";

export const protobufPackage = "cosmwasm.wasm.v1";

/**
 * ContractExecutionAuthorization defines authorization for wasm execute.
 * Since: wasmd 0.30
 */
export interface ContractExecutionAuthorization {
  /** Grants for contract executions */
  grants: ContractGrant[];
}

/**
 * ContractMigrationAuthorization defines authorization for wasm contract
 * migration. Since: wasmd 0.30
 */
export interface ContractMigrationAuthorization {
  /** Grants for contract migrations */
  grants: ContractGrant[];
}

/**
 * ContractGrant a granted permission for a single contract
 * Since: wasmd 0.30
 */
export interface ContractGrant {
  /** Contract is the bech32 address of the smart contract */
  contract: string;
  /**
   * Limit defines execution limits that are enforced and updated when the grant
   * is applied. When the limit lapsed the grant is removed.
   */
  limit: Any | undefined;
  /**
   * Filter define more fine-grained control on the message payload passed
   * to the contract in the operation. When no filter applies on execution, the
   * operation is prohibited.
   */
  filter: Any | undefined;
}

/**
 * MaxCallsLimit limited number of calls to the contract. No funds transferable.
 * Since: wasmd 0.30
 */
export interface MaxCallsLimit {
  /** Remaining number that is decremented on each execution */
  remaining: number;
}

/**
 * MaxFundsLimit defines the maximal amounts that can be sent to the contract.
 * Since: wasmd 0.30
 */
export interface MaxFundsLimit {
  /** Amounts is the maximal amount of tokens transferable to the contract. */
  amounts: Coin[];
}

/**
 * CombinedLimit defines the maximal amounts that can be sent to a contract and
 * the maximal number of calls executable. Both need to remain >0 to be valid.
 * Since: wasmd 0.30
 */
export interface CombinedLimit {
  /** Remaining number that is decremented on each execution */
  calls_remaining: number;
  /** Amounts is the maximal amount of tokens transferable to the contract. */
  amounts: Coin[];
}

/**
 * AllowAllMessagesFilter is a wildcard to allow any type of contract payload
 * message.
 * Since: wasmd 0.30
 */
export interface AllowAllMessagesFilter {}

/**
 * AcceptedMessageKeysFilter accept only the specific contract message keys in
 * the json object to be executed.
 * Since: wasmd 0.30
 */
export interface AcceptedMessageKeysFilter {
  /** Messages is the list of unique keys */
  keys: string[];
}

/**
 * AcceptedMessagesFilter accept only the specific raw contract messages to be
 * executed.
 * Since: wasmd 0.30
 */
export interface AcceptedMessagesFilter {
  /** Messages is the list of raw contract messages */
  messages: Uint8Array[];
}

const baseContractExecutionAuthorization: object = {};

export const ContractExecutionAuthorization = {
  encode(
    message: ContractExecutionAuthorization,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.grants) {
      ContractGrant.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): ContractExecutionAuthorization {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseContractExecutionAuthorization,
    } as ContractExecutionAuthorization;
    message.grants = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.grants.push(ContractGrant.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ContractExecutionAuthorization {
    const message = {
      ...baseContractExecutionAuthorization,
    } as ContractExecutionAuthorization;
    message.grants = [];
    if (object.grants !== undefined && object.grants !== null) {
      for (const e of object.grants) {
        message.grants.push(ContractGrant.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: ContractExecutionAuthorization): unknown {
    const obj: any = {};
    if (message.grants) {
      obj.grants = message.grants.map((e) =>
        e ? ContractGrant.toJSON(e) : undefined
      );
    } else {
      obj.grants = [];
    }
    return obj;
  },

  fromPartial(
    object: DeepPartial<ContractExecutionAuthorization>
  ): ContractExecutionAuthorization {
    const message = {
      ...baseContractExecutionAuthorization,
    } as ContractExecutionAuthorization;
    message.grants = [];
    if (object.grants !== undefined && object.grants !== null) {
      for (const e of object.grants) {
        message.grants.push(ContractGrant.fromPartial(e));
      }
    }
    return message;
  },
};

const baseContractMigrationAuthorization: object = {};

export const ContractMigrationAuthorization = {
  encode(
    message: ContractMigrationAuthorization,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.grants) {
      ContractGrant.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): ContractMigrationAuthorization {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseContractMigrationAuthorization,
    } as ContractMigrationAuthorization;
    message.grants = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.grants.push(ContractGrant.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ContractMigrationAuthorization {
    const message = {
      ...baseContractMigrationAuthorization,
    } as ContractMigrationAuthorization;
    message.grants = [];
    if (object.grants !== undefined && object.grants !== null) {
      for (const e of object.grants) {
        message.grants.push(ContractGrant.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: ContractMigrationAuthorization): unknown {
    const obj: any = {};
    if (message.grants) {
      obj.grants = message.grants.map((e) =>
        e ? ContractGrant.toJSON(e) : undefined
      );
    } else {
      obj.grants = [];
    }
    return obj;
  },

  fromPartial(
    object: DeepPartial<ContractMigrationAuthorization>
  ): ContractMigrationAuthorization {
    const message = {
      ...baseContractMigrationAuthorization,
    } as ContractMigrationAuthorization;
    message.grants = [];
    if (object.grants !== undefined && object.grants !== null) {
      for (const e of object.grants) {
        message.grants.push(ContractGrant.fromPartial(e));
      }
    }
    return message;
  },
};

const baseContractGrant: object = { contract: "" };

export const ContractGrant = {
  encode(message: ContractGrant, writer: Writer = Writer.create()): Writer {
    if (message.contract !== "") {
      writer.uint32(10).string(message.contract);
    }
    if (message.limit !== undefined) {
      Any.encode(message.limit, writer.uint32(18).fork()).ldelim();
    }
    if (message.filter !== undefined) {
      Any.encode(message.filter, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): ContractGrant {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseContractGrant } as ContractGrant;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.contract = reader.string();
          break;
        case 2:
          message.limit = Any.decode(reader, reader.uint32());
          break;
        case 3:
          message.filter = Any.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ContractGrant {
    const message = { ...baseContractGrant } as ContractGrant;
    if (object.contract !== undefined && object.contract !== null) {
      message.contract = String(object.contract);
    } else {
      message.contract = "";
    }
    if (object.limit !== undefined && object.limit !== null) {
      message.limit = Any.fromJSON(object.limit);
    } else {
      message.limit = undefined;
    }
    if (object.filter !== undefined && object.filter !== null) {
      message.filter = Any.fromJSON(object.filter);
    } else {
      message.filter = undefined;
    }
    return message;
  },

  toJSON(message: ContractGrant): unknown {
    const obj: any = {};
    message.contract !== undefined && (obj.contract = message.contract);
    message.limit !== undefined &&
      (obj.limit = message.limit ? Any.toJSON(message.limit) : undefined);
    message.filter !== undefined &&
      (obj.filter = message.filter ? Any.toJSON(message.filter) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<ContractGrant>): ContractGrant {
    const message = { ...baseContractGrant } as ContractGrant;
    if (object.contract !== undefined && object.contract !== null) {
      message.contract = object.contract;
    } else {
      message.contract = "";
    }
    if (object.limit !== undefined && object.limit !== null) {
      message.limit = Any.fromPartial(object.limit);
    } else {
      message.limit = undefined;
    }
    if (object.filter !== undefined && object.filter !== null) {
      message.filter = Any.fromPartial(object.filter);
    } else {
      message.filter = undefined;
    }
    return message;
  },
};

const baseMaxCallsLimit: object = { remaining: 0 };

export const MaxCallsLimit = {
  encode(message: MaxCallsLimit, writer: Writer = Writer.create()): Writer {
    if (message.remaining !== 0) {
      writer.uint32(8).uint64(message.remaining);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MaxCallsLimit {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMaxCallsLimit } as MaxCallsLimit;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.remaining = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MaxCallsLimit {
    const message = { ...baseMaxCallsLimit } as MaxCallsLimit;
    if (object.remaining !== undefined && object.remaining !== null) {
      message.remaining = Number(object.remaining);
    } else {
      message.remaining = 0;
    }
    return message;
  },

  toJSON(message: MaxCallsLimit): unknown {
    const obj: any = {};
    message.remaining !== undefined && (obj.remaining = message.remaining);
    return obj;
  },

  fromPartial(object: DeepPartial<MaxCallsLimit>): MaxCallsLimit {
    const message = { ...baseMaxCallsLimit } as MaxCallsLimit;
    if (object.remaining !== undefined && object.remaining !== null) {
      message.remaining = object.remaining;
    } else {
      message.remaining = 0;
    }
    return message;
  },
};

const baseMaxFundsLimit: object = {};

export const MaxFundsLimit = {
  encode(message: MaxFundsLimit, writer: Writer = Writer.create()): Writer {
    for (const v of message.amounts) {
      Coin.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MaxFundsLimit {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMaxFundsLimit } as MaxFundsLimit;
    message.amounts = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.amounts.push(Coin.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MaxFundsLimit {
    const message = { ...baseMaxFundsLimit } as MaxFundsLimit;
    message.amounts = [];
    if (object.amounts !== undefined && object.amounts !== null) {
      for (const e of object.amounts) {
        message.amounts.push(Coin.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: MaxFundsLimit): unknown {
    const obj: any = {};
    if (message.amounts) {
      obj.amounts = message.amounts.map((e) =>
        e ? Coin.toJSON(e) : undefined
      );
    } else {
      obj.amounts = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<MaxFundsLimit>): MaxFundsLimit {
    const message = { ...baseMaxFundsLimit } as MaxFundsLimit;
    message.amounts = [];
    if (object.amounts !== undefined && object.amounts !== null) {
      for (const e of object.amounts) {
        message.amounts.push(Coin.fromPartial(e));
      }
    }
    return message;
  },
};

const baseCombinedLimit: object = { calls_remaining: 0 };

export const CombinedLimit = {
  encode(message: CombinedLimit, writer: Writer = Writer.create()): Writer {
    if (message.calls_remaining !== 0) {
      writer.uint32(8).uint64(message.calls_remaining);
    }
    for (const v of message.amounts) {
      Coin.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): CombinedLimit {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseCombinedLimit } as CombinedLimit;
    message.amounts = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.calls_remaining = longToNumber(reader.uint64() as Long);
          break;
        case 2:
          message.amounts.push(Coin.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): CombinedLimit {
    const message = { ...baseCombinedLimit } as CombinedLimit;
    message.amounts = [];
    if (
      object.calls_remaining !== undefined &&
      object.calls_remaining !== null
    ) {
      message.calls_remaining = Number(object.calls_remaining);
    } else {
      message.calls_remaining = 0;
    }
    if (object.amounts !== undefined && object.amounts !== null) {
      for (const e of object.amounts) {
        message.amounts.push(Coin.fromJSON(e));
      }
    }
    return message;
  },

  toJSON(message: CombinedLimit): unknown {
    const obj: any = {};
    message.calls_remaining !== undefined &&
      (obj.calls_remaining = message.calls_remaining);
    if (message.amounts) {
      obj.amounts = message.amounts.map((e) =>
        e ? Coin.toJSON(e) : undefined
      );
    } else {
      obj.amounts = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<CombinedLimit>): CombinedLimit {
    const message = { ...baseCombinedLimit } as CombinedLimit;
    message.amounts = [];
    if (
      object.calls_remaining !== undefined &&
      object.calls_remaining !== null
    ) {
      message.calls_remaining = object.calls_remaining;
    } else {
      message.calls_remaining = 0;
    }
    if (object.amounts !== undefined && object.amounts !== null) {
      for (const e of object.amounts) {
        message.amounts.push(Coin.fromPartial(e));
      }
    }
    return message;
  },
};

const baseAllowAllMessagesFilter: object = {};

export const AllowAllMessagesFilter = {
  encode(_: AllowAllMessagesFilter, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): AllowAllMessagesFilter {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseAllowAllMessagesFilter } as AllowAllMessagesFilter;
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

  fromJSON(_: any): AllowAllMessagesFilter {
    const message = { ...baseAllowAllMessagesFilter } as AllowAllMessagesFilter;
    return message;
  },

  toJSON(_: AllowAllMessagesFilter): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<AllowAllMessagesFilter>): AllowAllMessagesFilter {
    const message = { ...baseAllowAllMessagesFilter } as AllowAllMessagesFilter;
    return message;
  },
};

const baseAcceptedMessageKeysFilter: object = { keys: "" };

export const AcceptedMessageKeysFilter = {
  encode(
    message: AcceptedMessageKeysFilter,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.keys) {
      writer.uint32(10).string(v!);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): AcceptedMessageKeysFilter {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseAcceptedMessageKeysFilter,
    } as AcceptedMessageKeysFilter;
    message.keys = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.keys.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): AcceptedMessageKeysFilter {
    const message = {
      ...baseAcceptedMessageKeysFilter,
    } as AcceptedMessageKeysFilter;
    message.keys = [];
    if (object.keys !== undefined && object.keys !== null) {
      for (const e of object.keys) {
        message.keys.push(String(e));
      }
    }
    return message;
  },

  toJSON(message: AcceptedMessageKeysFilter): unknown {
    const obj: any = {};
    if (message.keys) {
      obj.keys = message.keys.map((e) => e);
    } else {
      obj.keys = [];
    }
    return obj;
  },

  fromPartial(
    object: DeepPartial<AcceptedMessageKeysFilter>
  ): AcceptedMessageKeysFilter {
    const message = {
      ...baseAcceptedMessageKeysFilter,
    } as AcceptedMessageKeysFilter;
    message.keys = [];
    if (object.keys !== undefined && object.keys !== null) {
      for (const e of object.keys) {
        message.keys.push(e);
      }
    }
    return message;
  },
};

const baseAcceptedMessagesFilter: object = {};

export const AcceptedMessagesFilter = {
  encode(
    message: AcceptedMessagesFilter,
    writer: Writer = Writer.create()
  ): Writer {
    for (const v of message.messages) {
      writer.uint32(10).bytes(v!);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): AcceptedMessagesFilter {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseAcceptedMessagesFilter } as AcceptedMessagesFilter;
    message.messages = [];
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.messages.push(reader.bytes());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): AcceptedMessagesFilter {
    const message = { ...baseAcceptedMessagesFilter } as AcceptedMessagesFilter;
    message.messages = [];
    if (object.messages !== undefined && object.messages !== null) {
      for (const e of object.messages) {
        message.messages.push(bytesFromBase64(e));
      }
    }
    return message;
  },

  toJSON(message: AcceptedMessagesFilter): unknown {
    const obj: any = {};
    if (message.messages) {
      obj.messages = message.messages.map((e) =>
        base64FromBytes(e !== undefined ? e : new Uint8Array())
      );
    } else {
      obj.messages = [];
    }
    return obj;
  },

  fromPartial(
    object: DeepPartial<AcceptedMessagesFilter>
  ): AcceptedMessagesFilter {
    const message = { ...baseAcceptedMessagesFilter } as AcceptedMessagesFilter;
    message.messages = [];
    if (object.messages !== undefined && object.messages !== null) {
      for (const e of object.messages) {
        message.messages.push(e);
      }
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
