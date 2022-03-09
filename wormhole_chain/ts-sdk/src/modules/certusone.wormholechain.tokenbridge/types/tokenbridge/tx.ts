//@ts-nocheck
/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { Coin } from "../cosmos/base/v1beta1/coin";

export const protobufPackage = "certusone.wormholechain.tokenbridge";

export interface MsgExecuteGovernanceVAA {
  creator: string;
  vaa: Uint8Array;
}

export interface MsgExecuteGovernanceVAAResponse {}

export interface MsgExecuteVAA {
  creator: string;
  vaa: Uint8Array;
}

export interface MsgExecuteVAAResponse {}

export interface MsgAttestToken {
  creator: string;
  denom: string;
}

export interface MsgAttestTokenResponse {}

export interface MsgTransfer {
  creator: string;
  amount: Coin | undefined;
  toChain: number;
  toAddress: Uint8Array;
  fee: string;
}

export interface MsgTransferResponse {}

const baseMsgExecuteGovernanceVAA: object = { creator: "" };

export const MsgExecuteGovernanceVAA = {
  encode(
    message: MsgExecuteGovernanceVAA,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.vaa.length !== 0) {
      writer.uint32(18).bytes(message.vaa);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgExecuteGovernanceVAA {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgExecuteGovernanceVAA,
    } as MsgExecuteGovernanceVAA;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.vaa = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgExecuteGovernanceVAA {
    const message = {
      ...baseMsgExecuteGovernanceVAA,
    } as MsgExecuteGovernanceVAA;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = bytesFromBase64(object.vaa);
    }
    return message;
  },

  toJSON(message: MsgExecuteGovernanceVAA): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.vaa !== undefined &&
      (obj.vaa = base64FromBytes(
        message.vaa !== undefined ? message.vaa : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgExecuteGovernanceVAA>
  ): MsgExecuteGovernanceVAA {
    const message = {
      ...baseMsgExecuteGovernanceVAA,
    } as MsgExecuteGovernanceVAA;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = object.vaa;
    } else {
      message.vaa = new Uint8Array();
    }
    return message;
  },
};

const baseMsgExecuteGovernanceVAAResponse: object = {};

export const MsgExecuteGovernanceVAAResponse = {
  encode(
    _: MsgExecuteGovernanceVAAResponse,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgExecuteGovernanceVAAResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgExecuteGovernanceVAAResponse,
    } as MsgExecuteGovernanceVAAResponse;
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

  fromJSON(_: any): MsgExecuteGovernanceVAAResponse {
    const message = {
      ...baseMsgExecuteGovernanceVAAResponse,
    } as MsgExecuteGovernanceVAAResponse;
    return message;
  },

  toJSON(_: MsgExecuteGovernanceVAAResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<MsgExecuteGovernanceVAAResponse>
  ): MsgExecuteGovernanceVAAResponse {
    const message = {
      ...baseMsgExecuteGovernanceVAAResponse,
    } as MsgExecuteGovernanceVAAResponse;
    return message;
  },
};

const baseMsgExecuteVAA: object = { creator: "" };

export const MsgExecuteVAA = {
  encode(message: MsgExecuteVAA, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.vaa.length !== 0) {
      writer.uint32(18).bytes(message.vaa);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgExecuteVAA {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgExecuteVAA } as MsgExecuteVAA;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.vaa = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgExecuteVAA {
    const message = { ...baseMsgExecuteVAA } as MsgExecuteVAA;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = bytesFromBase64(object.vaa);
    }
    return message;
  },

  toJSON(message: MsgExecuteVAA): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.vaa !== undefined &&
      (obj.vaa = base64FromBytes(
        message.vaa !== undefined ? message.vaa : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(object: DeepPartial<MsgExecuteVAA>): MsgExecuteVAA {
    const message = { ...baseMsgExecuteVAA } as MsgExecuteVAA;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = object.vaa;
    } else {
      message.vaa = new Uint8Array();
    }
    return message;
  },
};

const baseMsgExecuteVAAResponse: object = {};

export const MsgExecuteVAAResponse = {
  encode(_: MsgExecuteVAAResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgExecuteVAAResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgExecuteVAAResponse } as MsgExecuteVAAResponse;
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

  fromJSON(_: any): MsgExecuteVAAResponse {
    const message = { ...baseMsgExecuteVAAResponse } as MsgExecuteVAAResponse;
    return message;
  },

  toJSON(_: MsgExecuteVAAResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgExecuteVAAResponse>): MsgExecuteVAAResponse {
    const message = { ...baseMsgExecuteVAAResponse } as MsgExecuteVAAResponse;
    return message;
  },
};

const baseMsgAttestToken: object = { creator: "", denom: "" };

export const MsgAttestToken = {
  encode(message: MsgAttestToken, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.denom !== "") {
      writer.uint32(18).string(message.denom);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgAttestToken {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgAttestToken } as MsgAttestToken;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.denom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgAttestToken {
    const message = { ...baseMsgAttestToken } as MsgAttestToken;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = String(object.denom);
    } else {
      message.denom = "";
    }
    return message;
  },

  toJSON(message: MsgAttestToken): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.denom !== undefined && (obj.denom = message.denom);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgAttestToken>): MsgAttestToken {
    const message = { ...baseMsgAttestToken } as MsgAttestToken;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = object.denom;
    } else {
      message.denom = "";
    }
    return message;
  },
};

const baseMsgAttestTokenResponse: object = {};

export const MsgAttestTokenResponse = {
  encode(_: MsgAttestTokenResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgAttestTokenResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgAttestTokenResponse } as MsgAttestTokenResponse;
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

  fromJSON(_: any): MsgAttestTokenResponse {
    const message = { ...baseMsgAttestTokenResponse } as MsgAttestTokenResponse;
    return message;
  },

  toJSON(_: MsgAttestTokenResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgAttestTokenResponse>): MsgAttestTokenResponse {
    const message = { ...baseMsgAttestTokenResponse } as MsgAttestTokenResponse;
    return message;
  },
};

const baseMsgTransfer: object = { creator: "", toChain: 0, fee: "" };

export const MsgTransfer = {
  encode(message: MsgTransfer, writer: Writer = Writer.create()): Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.amount !== undefined) {
      Coin.encode(message.amount, writer.uint32(18).fork()).ldelim();
    }
    if (message.toChain !== 0) {
      writer.uint32(24).uint32(message.toChain);
    }
    if (message.toAddress.length !== 0) {
      writer.uint32(34).bytes(message.toAddress);
    }
    if (message.fee !== "") {
      writer.uint32(42).string(message.fee);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgTransfer {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgTransfer } as MsgTransfer;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.amount = Coin.decode(reader, reader.uint32());
          break;
        case 3:
          message.toChain = reader.uint32();
          break;
        case 4:
          message.toAddress = reader.bytes();
          break;
        case 5:
          message.fee = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgTransfer {
    const message = { ...baseMsgTransfer } as MsgTransfer;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = String(object.creator);
    } else {
      message.creator = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = Coin.fromJSON(object.amount);
    } else {
      message.amount = undefined;
    }
    if (object.toChain !== undefined && object.toChain !== null) {
      message.toChain = Number(object.toChain);
    } else {
      message.toChain = 0;
    }
    if (object.toAddress !== undefined && object.toAddress !== null) {
      message.toAddress = bytesFromBase64(object.toAddress);
    }
    if (object.fee !== undefined && object.fee !== null) {
      message.fee = String(object.fee);
    } else {
      message.fee = "";
    }
    return message;
  },

  toJSON(message: MsgTransfer): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.amount !== undefined &&
      (obj.amount = message.amount ? Coin.toJSON(message.amount) : undefined);
    message.toChain !== undefined && (obj.toChain = message.toChain);
    message.toAddress !== undefined &&
      (obj.toAddress = base64FromBytes(
        message.toAddress !== undefined ? message.toAddress : new Uint8Array()
      ));
    message.fee !== undefined && (obj.fee = message.fee);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgTransfer>): MsgTransfer {
    const message = { ...baseMsgTransfer } as MsgTransfer;
    if (object.creator !== undefined && object.creator !== null) {
      message.creator = object.creator;
    } else {
      message.creator = "";
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = Coin.fromPartial(object.amount);
    } else {
      message.amount = undefined;
    }
    if (object.toChain !== undefined && object.toChain !== null) {
      message.toChain = object.toChain;
    } else {
      message.toChain = 0;
    }
    if (object.toAddress !== undefined && object.toAddress !== null) {
      message.toAddress = object.toAddress;
    } else {
      message.toAddress = new Uint8Array();
    }
    if (object.fee !== undefined && object.fee !== null) {
      message.fee = object.fee;
    } else {
      message.fee = "";
    }
    return message;
  },
};

const baseMsgTransferResponse: object = {};

export const MsgTransferResponse = {
  encode(_: MsgTransferResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgTransferResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgTransferResponse } as MsgTransferResponse;
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

  fromJSON(_: any): MsgTransferResponse {
    const message = { ...baseMsgTransferResponse } as MsgTransferResponse;
    return message;
  },

  toJSON(_: MsgTransferResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgTransferResponse>): MsgTransferResponse {
    const message = { ...baseMsgTransferResponse } as MsgTransferResponse;
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  ExecuteGovernanceVAA(
    request: MsgExecuteGovernanceVAA
  ): Promise<MsgExecuteGovernanceVAAResponse>;
  ExecuteVAA(request: MsgExecuteVAA): Promise<MsgExecuteVAAResponse>;
  AttestToken(request: MsgAttestToken): Promise<MsgAttestTokenResponse>;
  /** this line is used by starport scaffolding # proto/tx/rpc */
  Transfer(request: MsgTransfer): Promise<MsgTransferResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
  }
  ExecuteGovernanceVAA(
    request: MsgExecuteGovernanceVAA
  ): Promise<MsgExecuteGovernanceVAAResponse> {
    const data = MsgExecuteGovernanceVAA.encode(request).finish();
    const promise = this.rpc.request(
      "certusone.wormholechain.tokenbridge.Msg",
      "ExecuteGovernanceVAA",
      data
    );
    return promise.then((data) =>
      MsgExecuteGovernanceVAAResponse.decode(new Reader(data))
    );
  }

  ExecuteVAA(request: MsgExecuteVAA): Promise<MsgExecuteVAAResponse> {
    const data = MsgExecuteVAA.encode(request).finish();
    const promise = this.rpc.request(
      "certusone.wormholechain.tokenbridge.Msg",
      "ExecuteVAA",
      data
    );
    return promise.then((data) =>
      MsgExecuteVAAResponse.decode(new Reader(data))
    );
  }

  AttestToken(request: MsgAttestToken): Promise<MsgAttestTokenResponse> {
    const data = MsgAttestToken.encode(request).finish();
    const promise = this.rpc.request(
      "certusone.wormholechain.tokenbridge.Msg",
      "AttestToken",
      data
    );
    return promise.then((data) =>
      MsgAttestTokenResponse.decode(new Reader(data))
    );
  }

  Transfer(request: MsgTransfer): Promise<MsgTransferResponse> {
    const data = MsgTransfer.encode(request).finish();
    const promise = this.rpc.request(
      "certusone.wormholechain.tokenbridge.Msg",
      "Transfer",
      data
    );
    return promise.then((data) => MsgTransferResponse.decode(new Reader(data)));
  }
}

interface Rpc {
  request(
    service: string,
    method: string,
    data: Uint8Array
  ): Promise<Uint8Array>;
}

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
