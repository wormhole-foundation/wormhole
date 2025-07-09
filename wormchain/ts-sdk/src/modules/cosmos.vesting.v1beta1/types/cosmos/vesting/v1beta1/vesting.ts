//@ts-nocheck
/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { BaseAccount } from "../../auth/v1beta1/auth";
import { Coin } from "../../base/v1beta1/coin";

export const protobufPackage = "cosmos.vesting.v1beta1";

/**
 * BaseVestingAccount implements the VestingAccount interface. It contains all
 * the necessary fields needed for any vesting account implementation.
 */
export interface BaseVestingAccount {
  baseAccount: BaseAccount | undefined;
  originalVesting: Coin[];
  delegatedFree: Coin[];
  delegatedVesting: Coin[];
  /** Vesting end time, as unix timestamp (in seconds). */
  endTime: number;
}

/**
 * ContinuousVestingAccount implements the VestingAccount interface. It
 * continuously vests by unlocking coins linearly with respect to time.
 */
export interface ContinuousVestingAccount {
  baseVestingAccount:
    | BaseVestingAccount
    | undefined;
  /** Vesting start time, as unix timestamp (in seconds). */
  startTime: number;
}

/**
 * DelayedVestingAccount implements the VestingAccount interface. It vests all
 * coins after a specific time, but non prior. In other words, it keeps them
 * locked until a specified time.
 */
export interface DelayedVestingAccount {
  baseVestingAccount: BaseVestingAccount | undefined;
}

/** Period defines a length of time and amount of coins that will vest. */
export interface Period {
  /** Period duration in seconds. */
  length: number;
  amount: Coin[];
}

/**
 * PeriodicVestingAccount implements the VestingAccount interface. It
 * periodically vests by unlocking coins during each specified period.
 */
export interface PeriodicVestingAccount {
  baseVestingAccount: BaseVestingAccount | undefined;
  startTime: number;
  vestingPeriods: Period[];
}

/**
 * PermanentLockedAccount implements the VestingAccount interface. It does
 * not ever release coins, locking them indefinitely. Coins in this account can
 * still be used for delegating and for governance votes even while locked.
 *
 * Since: cosmos-sdk 0.43
 */
export interface PermanentLockedAccount {
  baseVestingAccount: BaseVestingAccount | undefined;
}

function createBaseBaseVestingAccount(): BaseVestingAccount {
  return { baseAccount: undefined, originalVesting: [], delegatedFree: [], delegatedVesting: [], endTime: 0 };
}

export const BaseVestingAccount = {
  encode(message: BaseVestingAccount, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.baseAccount !== undefined) {
      BaseAccount.encode(message.baseAccount, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.originalVesting) {
      Coin.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.delegatedFree) {
      Coin.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.delegatedVesting) {
      Coin.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    if (message.endTime !== 0) {
      writer.uint32(40).int64(message.endTime);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BaseVestingAccount {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBaseVestingAccount();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.baseAccount = BaseAccount.decode(reader, reader.uint32());
          break;
        case 2:
          message.originalVesting.push(Coin.decode(reader, reader.uint32()));
          break;
        case 3:
          message.delegatedFree.push(Coin.decode(reader, reader.uint32()));
          break;
        case 4:
          message.delegatedVesting.push(Coin.decode(reader, reader.uint32()));
          break;
        case 5:
          message.endTime = longToNumber(reader.int64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): BaseVestingAccount {
    return {
      baseAccount: isSet(object.baseAccount) ? BaseAccount.fromJSON(object.baseAccount) : undefined,
      originalVesting: Array.isArray(object?.originalVesting)
        ? object.originalVesting.map((e: any) => Coin.fromJSON(e))
        : [],
      delegatedFree: Array.isArray(object?.delegatedFree) ? object.delegatedFree.map((e: any) => Coin.fromJSON(e)) : [],
      delegatedVesting: Array.isArray(object?.delegatedVesting)
        ? object.delegatedVesting.map((e: any) => Coin.fromJSON(e))
        : [],
      endTime: isSet(object.endTime) ? Number(object.endTime) : 0,
    };
  },

  toJSON(message: BaseVestingAccount): unknown {
    const obj: any = {};
    message.baseAccount !== undefined
      && (obj.baseAccount = message.baseAccount ? BaseAccount.toJSON(message.baseAccount) : undefined);
    if (message.originalVesting) {
      obj.originalVesting = message.originalVesting.map((e) => e ? Coin.toJSON(e) : undefined);
    } else {
      obj.originalVesting = [];
    }
    if (message.delegatedFree) {
      obj.delegatedFree = message.delegatedFree.map((e) => e ? Coin.toJSON(e) : undefined);
    } else {
      obj.delegatedFree = [];
    }
    if (message.delegatedVesting) {
      obj.delegatedVesting = message.delegatedVesting.map((e) => e ? Coin.toJSON(e) : undefined);
    } else {
      obj.delegatedVesting = [];
    }
    message.endTime !== undefined && (obj.endTime = Math.round(message.endTime));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<BaseVestingAccount>, I>>(object: I): BaseVestingAccount {
    const message = createBaseBaseVestingAccount();
    message.baseAccount = (object.baseAccount !== undefined && object.baseAccount !== null)
      ? BaseAccount.fromPartial(object.baseAccount)
      : undefined;
    message.originalVesting = object.originalVesting?.map((e) => Coin.fromPartial(e)) || [];
    message.delegatedFree = object.delegatedFree?.map((e) => Coin.fromPartial(e)) || [];
    message.delegatedVesting = object.delegatedVesting?.map((e) => Coin.fromPartial(e)) || [];
    message.endTime = object.endTime ?? 0;
    return message;
  },
};

function createBaseContinuousVestingAccount(): ContinuousVestingAccount {
  return { baseVestingAccount: undefined, startTime: 0 };
}

export const ContinuousVestingAccount = {
  encode(message: ContinuousVestingAccount, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.baseVestingAccount !== undefined) {
      BaseVestingAccount.encode(message.baseVestingAccount, writer.uint32(10).fork()).ldelim();
    }
    if (message.startTime !== 0) {
      writer.uint32(16).int64(message.startTime);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ContinuousVestingAccount {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseContinuousVestingAccount();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.baseVestingAccount = BaseVestingAccount.decode(reader, reader.uint32());
          break;
        case 2:
          message.startTime = longToNumber(reader.int64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ContinuousVestingAccount {
    return {
      baseVestingAccount: isSet(object.baseVestingAccount)
        ? BaseVestingAccount.fromJSON(object.baseVestingAccount)
        : undefined,
      startTime: isSet(object.startTime) ? Number(object.startTime) : 0,
    };
  },

  toJSON(message: ContinuousVestingAccount): unknown {
    const obj: any = {};
    message.baseVestingAccount !== undefined && (obj.baseVestingAccount = message.baseVestingAccount
      ? BaseVestingAccount.toJSON(message.baseVestingAccount)
      : undefined);
    message.startTime !== undefined && (obj.startTime = Math.round(message.startTime));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ContinuousVestingAccount>, I>>(object: I): ContinuousVestingAccount {
    const message = createBaseContinuousVestingAccount();
    message.baseVestingAccount = (object.baseVestingAccount !== undefined && object.baseVestingAccount !== null)
      ? BaseVestingAccount.fromPartial(object.baseVestingAccount)
      : undefined;
    message.startTime = object.startTime ?? 0;
    return message;
  },
};

function createBaseDelayedVestingAccount(): DelayedVestingAccount {
  return { baseVestingAccount: undefined };
}

export const DelayedVestingAccount = {
  encode(message: DelayedVestingAccount, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.baseVestingAccount !== undefined) {
      BaseVestingAccount.encode(message.baseVestingAccount, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DelayedVestingAccount {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDelayedVestingAccount();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.baseVestingAccount = BaseVestingAccount.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DelayedVestingAccount {
    return {
      baseVestingAccount: isSet(object.baseVestingAccount)
        ? BaseVestingAccount.fromJSON(object.baseVestingAccount)
        : undefined,
    };
  },

  toJSON(message: DelayedVestingAccount): unknown {
    const obj: any = {};
    message.baseVestingAccount !== undefined && (obj.baseVestingAccount = message.baseVestingAccount
      ? BaseVestingAccount.toJSON(message.baseVestingAccount)
      : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DelayedVestingAccount>, I>>(object: I): DelayedVestingAccount {
    const message = createBaseDelayedVestingAccount();
    message.baseVestingAccount = (object.baseVestingAccount !== undefined && object.baseVestingAccount !== null)
      ? BaseVestingAccount.fromPartial(object.baseVestingAccount)
      : undefined;
    return message;
  },
};

function createBasePeriod(): Period {
  return { length: 0, amount: [] };
}

export const Period = {
  encode(message: Period, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.length !== 0) {
      writer.uint32(8).int64(message.length);
    }
    for (const v of message.amount) {
      Coin.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Period {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePeriod();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.length = longToNumber(reader.int64() as Long);
          break;
        case 2:
          message.amount.push(Coin.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Period {
    return {
      length: isSet(object.length) ? Number(object.length) : 0,
      amount: Array.isArray(object?.amount) ? object.amount.map((e: any) => Coin.fromJSON(e)) : [],
    };
  },

  toJSON(message: Period): unknown {
    const obj: any = {};
    message.length !== undefined && (obj.length = Math.round(message.length));
    if (message.amount) {
      obj.amount = message.amount.map((e) => e ? Coin.toJSON(e) : undefined);
    } else {
      obj.amount = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Period>, I>>(object: I): Period {
    const message = createBasePeriod();
    message.length = object.length ?? 0;
    message.amount = object.amount?.map((e) => Coin.fromPartial(e)) || [];
    return message;
  },
};

function createBasePeriodicVestingAccount(): PeriodicVestingAccount {
  return { baseVestingAccount: undefined, startTime: 0, vestingPeriods: [] };
}

export const PeriodicVestingAccount = {
  encode(message: PeriodicVestingAccount, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.baseVestingAccount !== undefined) {
      BaseVestingAccount.encode(message.baseVestingAccount, writer.uint32(10).fork()).ldelim();
    }
    if (message.startTime !== 0) {
      writer.uint32(16).int64(message.startTime);
    }
    for (const v of message.vestingPeriods) {
      Period.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PeriodicVestingAccount {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePeriodicVestingAccount();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.baseVestingAccount = BaseVestingAccount.decode(reader, reader.uint32());
          break;
        case 2:
          message.startTime = longToNumber(reader.int64() as Long);
          break;
        case 3:
          message.vestingPeriods.push(Period.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PeriodicVestingAccount {
    return {
      baseVestingAccount: isSet(object.baseVestingAccount)
        ? BaseVestingAccount.fromJSON(object.baseVestingAccount)
        : undefined,
      startTime: isSet(object.startTime) ? Number(object.startTime) : 0,
      vestingPeriods: Array.isArray(object?.vestingPeriods)
        ? object.vestingPeriods.map((e: any) => Period.fromJSON(e))
        : [],
    };
  },

  toJSON(message: PeriodicVestingAccount): unknown {
    const obj: any = {};
    message.baseVestingAccount !== undefined && (obj.baseVestingAccount = message.baseVestingAccount
      ? BaseVestingAccount.toJSON(message.baseVestingAccount)
      : undefined);
    message.startTime !== undefined && (obj.startTime = Math.round(message.startTime));
    if (message.vestingPeriods) {
      obj.vestingPeriods = message.vestingPeriods.map((e) => e ? Period.toJSON(e) : undefined);
    } else {
      obj.vestingPeriods = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PeriodicVestingAccount>, I>>(object: I): PeriodicVestingAccount {
    const message = createBasePeriodicVestingAccount();
    message.baseVestingAccount = (object.baseVestingAccount !== undefined && object.baseVestingAccount !== null)
      ? BaseVestingAccount.fromPartial(object.baseVestingAccount)
      : undefined;
    message.startTime = object.startTime ?? 0;
    message.vestingPeriods = object.vestingPeriods?.map((e) => Period.fromPartial(e)) || [];
    return message;
  },
};

function createBasePermanentLockedAccount(): PermanentLockedAccount {
  return { baseVestingAccount: undefined };
}

export const PermanentLockedAccount = {
  encode(message: PermanentLockedAccount, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.baseVestingAccount !== undefined) {
      BaseVestingAccount.encode(message.baseVestingAccount, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PermanentLockedAccount {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePermanentLockedAccount();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.baseVestingAccount = BaseVestingAccount.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PermanentLockedAccount {
    return {
      baseVestingAccount: isSet(object.baseVestingAccount)
        ? BaseVestingAccount.fromJSON(object.baseVestingAccount)
        : undefined,
    };
  },

  toJSON(message: PermanentLockedAccount): unknown {
    const obj: any = {};
    message.baseVestingAccount !== undefined && (obj.baseVestingAccount = message.baseVestingAccount
      ? BaseVestingAccount.toJSON(message.baseVestingAccount)
      : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<PermanentLockedAccount>, I>>(object: I): PermanentLockedAccount {
    const message = createBasePermanentLockedAccount();
    message.baseVestingAccount = (object.baseVestingAccount !== undefined && object.baseVestingAccount !== null)
      ? BaseVestingAccount.fromPartial(object.baseVestingAccount)
      : undefined;
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
