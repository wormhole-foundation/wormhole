//@ts-nocheck
/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "certusone.wormholechain.tokenbridge";

export interface EventChainRegistered {
  chainID: number;
  emitterAddress: Uint8Array;
}

export interface EventAssetRegistrationUpdate {
  tokenChain: number;
  tokenAddress: Uint8Array;
  name: string;
  symbol: string;
  decimals: number;
}

export interface EventTransferReceived {
  tokenChain: number;
  tokenAddress: Uint8Array;
  to: string;
  feeRecipient: string;
  amount: string;
  fee: string;
  localDenom: string;
}

const baseEventChainRegistered: object = { chainID: 0 };

export const EventChainRegistered = {
  encode(
    message: EventChainRegistered,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.chainID !== 0) {
      writer.uint32(8).uint32(message.chainID);
    }
    if (message.emitterAddress.length !== 0) {
      writer.uint32(18).bytes(message.emitterAddress);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): EventChainRegistered {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseEventChainRegistered } as EventChainRegistered;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.chainID = reader.uint32();
          break;
        case 2:
          message.emitterAddress = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): EventChainRegistered {
    const message = { ...baseEventChainRegistered } as EventChainRegistered;
    if (object.chainID !== undefined && object.chainID !== null) {
      message.chainID = Number(object.chainID);
    } else {
      message.chainID = 0;
    }
    if (object.emitterAddress !== undefined && object.emitterAddress !== null) {
      message.emitterAddress = bytesFromBase64(object.emitterAddress);
    }
    return message;
  },

  toJSON(message: EventChainRegistered): unknown {
    const obj: any = {};
    message.chainID !== undefined && (obj.chainID = message.chainID);
    message.emitterAddress !== undefined &&
      (obj.emitterAddress = base64FromBytes(
        message.emitterAddress !== undefined
          ? message.emitterAddress
          : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(object: DeepPartial<EventChainRegistered>): EventChainRegistered {
    const message = { ...baseEventChainRegistered } as EventChainRegistered;
    if (object.chainID !== undefined && object.chainID !== null) {
      message.chainID = object.chainID;
    } else {
      message.chainID = 0;
    }
    if (object.emitterAddress !== undefined && object.emitterAddress !== null) {
      message.emitterAddress = object.emitterAddress;
    } else {
      message.emitterAddress = new Uint8Array();
    }
    return message;
  },
};

const baseEventAssetRegistrationUpdate: object = {
  tokenChain: 0,
  name: "",
  symbol: "",
  decimals: 0,
};

export const EventAssetRegistrationUpdate = {
  encode(
    message: EventAssetRegistrationUpdate,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.tokenChain !== 0) {
      writer.uint32(8).uint32(message.tokenChain);
    }
    if (message.tokenAddress.length !== 0) {
      writer.uint32(18).bytes(message.tokenAddress);
    }
    if (message.name !== "") {
      writer.uint32(26).string(message.name);
    }
    if (message.symbol !== "") {
      writer.uint32(34).string(message.symbol);
    }
    if (message.decimals !== 0) {
      writer.uint32(40).uint32(message.decimals);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): EventAssetRegistrationUpdate {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseEventAssetRegistrationUpdate,
    } as EventAssetRegistrationUpdate;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tokenChain = reader.uint32();
          break;
        case 2:
          message.tokenAddress = reader.bytes();
          break;
        case 3:
          message.name = reader.string();
          break;
        case 4:
          message.symbol = reader.string();
          break;
        case 5:
          message.decimals = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): EventAssetRegistrationUpdate {
    const message = {
      ...baseEventAssetRegistrationUpdate,
    } as EventAssetRegistrationUpdate;
    if (object.tokenChain !== undefined && object.tokenChain !== null) {
      message.tokenChain = Number(object.tokenChain);
    } else {
      message.tokenChain = 0;
    }
    if (object.tokenAddress !== undefined && object.tokenAddress !== null) {
      message.tokenAddress = bytesFromBase64(object.tokenAddress);
    }
    if (object.name !== undefined && object.name !== null) {
      message.name = String(object.name);
    } else {
      message.name = "";
    }
    if (object.symbol !== undefined && object.symbol !== null) {
      message.symbol = String(object.symbol);
    } else {
      message.symbol = "";
    }
    if (object.decimals !== undefined && object.decimals !== null) {
      message.decimals = Number(object.decimals);
    } else {
      message.decimals = 0;
    }
    return message;
  },

  toJSON(message: EventAssetRegistrationUpdate): unknown {
    const obj: any = {};
    message.tokenChain !== undefined && (obj.tokenChain = message.tokenChain);
    message.tokenAddress !== undefined &&
      (obj.tokenAddress = base64FromBytes(
        message.tokenAddress !== undefined
          ? message.tokenAddress
          : new Uint8Array()
      ));
    message.name !== undefined && (obj.name = message.name);
    message.symbol !== undefined && (obj.symbol = message.symbol);
    message.decimals !== undefined && (obj.decimals = message.decimals);
    return obj;
  },

  fromPartial(
    object: DeepPartial<EventAssetRegistrationUpdate>
  ): EventAssetRegistrationUpdate {
    const message = {
      ...baseEventAssetRegistrationUpdate,
    } as EventAssetRegistrationUpdate;
    if (object.tokenChain !== undefined && object.tokenChain !== null) {
      message.tokenChain = object.tokenChain;
    } else {
      message.tokenChain = 0;
    }
    if (object.tokenAddress !== undefined && object.tokenAddress !== null) {
      message.tokenAddress = object.tokenAddress;
    } else {
      message.tokenAddress = new Uint8Array();
    }
    if (object.name !== undefined && object.name !== null) {
      message.name = object.name;
    } else {
      message.name = "";
    }
    if (object.symbol !== undefined && object.symbol !== null) {
      message.symbol = object.symbol;
    } else {
      message.symbol = "";
    }
    if (object.decimals !== undefined && object.decimals !== null) {
      message.decimals = object.decimals;
    } else {
      message.decimals = 0;
    }
    return message;
  },
};

const baseEventTransferReceived: object = {
  tokenChain: 0,
  to: "",
  feeRecipient: "",
  amount: "",
  fee: "",
  localDenom: "",
};

export const EventTransferReceived = {
  encode(
    message: EventTransferReceived,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.tokenChain !== 0) {
      writer.uint32(8).uint32(message.tokenChain);
    }
    if (message.tokenAddress.length !== 0) {
      writer.uint32(18).bytes(message.tokenAddress);
    }
    if (message.to !== "") {
      writer.uint32(26).string(message.to);
    }
    if (message.feeRecipient !== "") {
      writer.uint32(34).string(message.feeRecipient);
    }
    if (message.amount !== "") {
      writer.uint32(42).string(message.amount);
    }
    if (message.fee !== "") {
      writer.uint32(50).string(message.fee);
    }
    if (message.localDenom !== "") {
      writer.uint32(58).string(message.localDenom);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): EventTransferReceived {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseEventTransferReceived } as EventTransferReceived;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tokenChain = reader.uint32();
          break;
        case 2:
          message.tokenAddress = reader.bytes();
          break;
        case 3:
          message.to = reader.string();
          break;
        case 4:
          message.feeRecipient = reader.string();
          break;
        case 5:
          message.amount = reader.string();
          break;
        case 6:
          message.fee = reader.string();
          break;
        case 7:
          message.localDenom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): EventTransferReceived {
    const message = { ...baseEventTransferReceived } as EventTransferReceived;
    if (object.tokenChain !== undefined && object.tokenChain !== null) {
      message.tokenChain = Number(object.tokenChain);
    } else {
      message.tokenChain = 0;
    }
    if (object.tokenAddress !== undefined && object.tokenAddress !== null) {
      message.tokenAddress = bytesFromBase64(object.tokenAddress);
    }
    if (object.to !== undefined && object.to !== null) {
      message.to = String(object.to);
    } else {
      message.to = "";
    }
    if (object.feeRecipient !== undefined && object.feeRecipient !== null) {
      message.feeRecipient = String(object.feeRecipient);
    } else {
      message.feeRecipient = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = String(object.amount);
    } else {
      message.amount = "";
    }
    if (object.fee !== undefined && object.fee !== null) {
      message.fee = String(object.fee);
    } else {
      message.fee = "";
    }
    if (object.localDenom !== undefined && object.localDenom !== null) {
      message.localDenom = String(object.localDenom);
    } else {
      message.localDenom = "";
    }
    return message;
  },

  toJSON(message: EventTransferReceived): unknown {
    const obj: any = {};
    message.tokenChain !== undefined && (obj.tokenChain = message.tokenChain);
    message.tokenAddress !== undefined &&
      (obj.tokenAddress = base64FromBytes(
        message.tokenAddress !== undefined
          ? message.tokenAddress
          : new Uint8Array()
      ));
    message.to !== undefined && (obj.to = message.to);
    message.feeRecipient !== undefined &&
      (obj.feeRecipient = message.feeRecipient);
    message.amount !== undefined && (obj.amount = message.amount);
    message.fee !== undefined && (obj.fee = message.fee);
    message.localDenom !== undefined && (obj.localDenom = message.localDenom);
    return obj;
  },

  fromPartial(
    object: DeepPartial<EventTransferReceived>
  ): EventTransferReceived {
    const message = { ...baseEventTransferReceived } as EventTransferReceived;
    if (object.tokenChain !== undefined && object.tokenChain !== null) {
      message.tokenChain = object.tokenChain;
    } else {
      message.tokenChain = 0;
    }
    if (object.tokenAddress !== undefined && object.tokenAddress !== null) {
      message.tokenAddress = object.tokenAddress;
    } else {
      message.tokenAddress = new Uint8Array();
    }
    if (object.to !== undefined && object.to !== null) {
      message.to = object.to;
    } else {
      message.to = "";
    }
    if (object.feeRecipient !== undefined && object.feeRecipient !== null) {
      message.feeRecipient = object.feeRecipient;
    } else {
      message.feeRecipient = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount;
    } else {
      message.amount = "";
    }
    if (object.fee !== undefined && object.fee !== null) {
      message.fee = object.fee;
    } else {
      message.fee = "";
    }
    if (object.localDenom !== undefined && object.localDenom !== null) {
      message.localDenom = object.localDenom;
    } else {
      message.localDenom = "";
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
