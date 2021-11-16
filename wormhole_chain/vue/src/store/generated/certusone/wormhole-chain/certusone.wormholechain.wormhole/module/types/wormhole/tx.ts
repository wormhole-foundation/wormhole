/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";

export const protobufPackage = "certusone.wormholechain.wormhole";

export interface MsgExecuteGovernanceVAA {
  vaa: Uint8Array;
  signer: string;
}

export interface MsgExecuteGovernanceVAAResponse {}

const baseMsgExecuteGovernanceVAA: object = { signer: "" };

export const MsgExecuteGovernanceVAA = {
  encode(
    message: MsgExecuteGovernanceVAA,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.vaa.length !== 0) {
      writer.uint32(10).bytes(message.vaa);
    }
    if (message.signer !== "") {
      writer.uint32(18).string(message.signer);
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
          message.vaa = reader.bytes();
          break;
        case 2:
          message.signer = reader.string();
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
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = bytesFromBase64(object.vaa);
    }
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = String(object.signer);
    } else {
      message.signer = "";
    }
    return message;
  },

  toJSON(message: MsgExecuteGovernanceVAA): unknown {
    const obj: any = {};
    message.vaa !== undefined &&
      (obj.vaa = base64FromBytes(
        message.vaa !== undefined ? message.vaa : new Uint8Array()
      ));
    message.signer !== undefined && (obj.signer = message.signer);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgExecuteGovernanceVAA>
  ): MsgExecuteGovernanceVAA {
    const message = {
      ...baseMsgExecuteGovernanceVAA,
    } as MsgExecuteGovernanceVAA;
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = object.vaa;
    } else {
      message.vaa = new Uint8Array();
    }
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = object.signer;
    } else {
      message.signer = "";
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

/** Msg defines the Msg service. */
export interface Msg {
  /** this line is used by starport scaffolding # proto/tx/rpc */
  ExecuteGovernanceVAA(
    request: MsgExecuteGovernanceVAA
  ): Promise<MsgExecuteGovernanceVAAResponse>;
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
      "certusone.wormholechain.wormhole.Msg",
      "ExecuteGovernanceVAA",
      data
    );
    return promise.then((data) =>
      MsgExecuteGovernanceVAAResponse.decode(new Reader(data))
    );
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
