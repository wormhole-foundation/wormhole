//@ts-nocheck
/* eslint-disable */
import { Reader, util, configure, Writer } from "protobufjs/minimal";
import * as Long from "long";

export const protobufPackage = "wormhole_foundation.wormchain.wormhole";

export interface MsgExecuteGovernanceVAA {
  vaa: Uint8Array;
  signer: string;
}

export interface MsgExecuteGovernanceVAAResponse {}

export interface MsgRegisterAccountAsGuardian {
  signer: string;
  signature: Uint8Array;
}

export interface MsgRegisterAccountAsGuardianResponse {}

/** Same as from x/wasmd but with vaa auth */
export interface MsgStoreCode {
  /** Signer is the that actor that signed the messages */
  signer: string;
  /** WASMByteCode can be raw or gzip compressed */
  wasm_byte_code: Uint8Array;
  /** vaa must be governance msg with payload containing sha3 256 hash of `wasm_byte_code` */
  vaa: Uint8Array;
}

export interface MsgStoreCodeResponse {
  /** CodeID is the reference to the stored WASM code */
  code_id: number;
}

/** Same as from x/wasmd but with vaa auth */
export interface MsgInstantiateContract {
  /** Signer is the that actor that signed the messages */
  signer: string;
  /** CodeID is the reference to the stored WASM code */
  code_id: number;
  /** Label is optional metadata to be stored with a contract instance. */
  label: string;
  /** Msg json encoded message to be passed to the contract on instantiation */
  msg: Uint8Array;
  /** vaa must be governance msg with payload containing sha3 256 hash of `bigEndian(code_id) || label || msg` */
  vaa: Uint8Array;
}

export interface MsgInstantiateContractResponse {
  /** Address is the bech32 address of the new contract instance. */
  address: string;
  /** Data contains base64-encoded bytes to returned from the contract */
  data: Uint8Array;
}

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

const baseMsgRegisterAccountAsGuardian: object = { signer: "" };

export const MsgRegisterAccountAsGuardian = {
  encode(
    message: MsgRegisterAccountAsGuardian,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.signature.length !== 0) {
      writer.uint32(26).bytes(message.signature);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgRegisterAccountAsGuardian {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgRegisterAccountAsGuardian,
    } as MsgRegisterAccountAsGuardian;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 3:
          message.signature = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgRegisterAccountAsGuardian {
    const message = {
      ...baseMsgRegisterAccountAsGuardian,
    } as MsgRegisterAccountAsGuardian;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = String(object.signer);
    } else {
      message.signer = "";
    }
    if (object.signature !== undefined && object.signature !== null) {
      message.signature = bytesFromBase64(object.signature);
    }
    return message;
  },

  toJSON(message: MsgRegisterAccountAsGuardian): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.signature !== undefined &&
      (obj.signature = base64FromBytes(
        message.signature !== undefined ? message.signature : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgRegisterAccountAsGuardian>
  ): MsgRegisterAccountAsGuardian {
    const message = {
      ...baseMsgRegisterAccountAsGuardian,
    } as MsgRegisterAccountAsGuardian;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = object.signer;
    } else {
      message.signer = "";
    }
    if (object.signature !== undefined && object.signature !== null) {
      message.signature = object.signature;
    } else {
      message.signature = new Uint8Array();
    }
    return message;
  },
};

const baseMsgRegisterAccountAsGuardianResponse: object = {};

export const MsgRegisterAccountAsGuardianResponse = {
  encode(
    _: MsgRegisterAccountAsGuardianResponse,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgRegisterAccountAsGuardianResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgRegisterAccountAsGuardianResponse,
    } as MsgRegisterAccountAsGuardianResponse;
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

  fromJSON(_: any): MsgRegisterAccountAsGuardianResponse {
    const message = {
      ...baseMsgRegisterAccountAsGuardianResponse,
    } as MsgRegisterAccountAsGuardianResponse;
    return message;
  },

  toJSON(_: MsgRegisterAccountAsGuardianResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<MsgRegisterAccountAsGuardianResponse>
  ): MsgRegisterAccountAsGuardianResponse {
    const message = {
      ...baseMsgRegisterAccountAsGuardianResponse,
    } as MsgRegisterAccountAsGuardianResponse;
    return message;
  },
};

const baseMsgStoreCode: object = { signer: "" };

export const MsgStoreCode = {
  encode(message: MsgStoreCode, writer: Writer = Writer.create()): Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.wasm_byte_code.length !== 0) {
      writer.uint32(18).bytes(message.wasm_byte_code);
    }
    if (message.vaa.length !== 0) {
      writer.uint32(26).bytes(message.vaa);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgStoreCode {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgStoreCode } as MsgStoreCode;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 2:
          message.wasm_byte_code = reader.bytes();
          break;
        case 3:
          message.vaa = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgStoreCode {
    const message = { ...baseMsgStoreCode } as MsgStoreCode;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = String(object.signer);
    } else {
      message.signer = "";
    }
    if (object.wasm_byte_code !== undefined && object.wasm_byte_code !== null) {
      message.wasm_byte_code = bytesFromBase64(object.wasm_byte_code);
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = bytesFromBase64(object.vaa);
    }
    return message;
  },

  toJSON(message: MsgStoreCode): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.wasm_byte_code !== undefined &&
      (obj.wasm_byte_code = base64FromBytes(
        message.wasm_byte_code !== undefined
          ? message.wasm_byte_code
          : new Uint8Array()
      ));
    message.vaa !== undefined &&
      (obj.vaa = base64FromBytes(
        message.vaa !== undefined ? message.vaa : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(object: DeepPartial<MsgStoreCode>): MsgStoreCode {
    const message = { ...baseMsgStoreCode } as MsgStoreCode;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = object.signer;
    } else {
      message.signer = "";
    }
    if (object.wasm_byte_code !== undefined && object.wasm_byte_code !== null) {
      message.wasm_byte_code = object.wasm_byte_code;
    } else {
      message.wasm_byte_code = new Uint8Array();
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = object.vaa;
    } else {
      message.vaa = new Uint8Array();
    }
    return message;
  },
};

const baseMsgStoreCodeResponse: object = { code_id: 0 };

export const MsgStoreCodeResponse = {
  encode(
    message: MsgStoreCodeResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.code_id !== 0) {
      writer.uint32(8).uint64(message.code_id);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgStoreCodeResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgStoreCodeResponse } as MsgStoreCodeResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.code_id = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgStoreCodeResponse {
    const message = { ...baseMsgStoreCodeResponse } as MsgStoreCodeResponse;
    if (object.code_id !== undefined && object.code_id !== null) {
      message.code_id = Number(object.code_id);
    } else {
      message.code_id = 0;
    }
    return message;
  },

  toJSON(message: MsgStoreCodeResponse): unknown {
    const obj: any = {};
    message.code_id !== undefined && (obj.code_id = message.code_id);
    return obj;
  },

  fromPartial(object: DeepPartial<MsgStoreCodeResponse>): MsgStoreCodeResponse {
    const message = { ...baseMsgStoreCodeResponse } as MsgStoreCodeResponse;
    if (object.code_id !== undefined && object.code_id !== null) {
      message.code_id = object.code_id;
    } else {
      message.code_id = 0;
    }
    return message;
  },
};

const baseMsgInstantiateContract: object = {
  signer: "",
  code_id: 0,
  label: "",
};

export const MsgInstantiateContract = {
  encode(
    message: MsgInstantiateContract,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.code_id !== 0) {
      writer.uint32(24).uint64(message.code_id);
    }
    if (message.label !== "") {
      writer.uint32(34).string(message.label);
    }
    if (message.msg.length !== 0) {
      writer.uint32(42).bytes(message.msg);
    }
    if (message.vaa.length !== 0) {
      writer.uint32(50).bytes(message.vaa);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgInstantiateContract {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgInstantiateContract } as MsgInstantiateContract;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 3:
          message.code_id = longToNumber(reader.uint64() as Long);
          break;
        case 4:
          message.label = reader.string();
          break;
        case 5:
          message.msg = reader.bytes();
          break;
        case 6:
          message.vaa = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgInstantiateContract {
    const message = { ...baseMsgInstantiateContract } as MsgInstantiateContract;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = String(object.signer);
    } else {
      message.signer = "";
    }
    if (object.code_id !== undefined && object.code_id !== null) {
      message.code_id = Number(object.code_id);
    } else {
      message.code_id = 0;
    }
    if (object.label !== undefined && object.label !== null) {
      message.label = String(object.label);
    } else {
      message.label = "";
    }
    if (object.msg !== undefined && object.msg !== null) {
      message.msg = bytesFromBase64(object.msg);
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = bytesFromBase64(object.vaa);
    }
    return message;
  },

  toJSON(message: MsgInstantiateContract): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.code_id !== undefined && (obj.code_id = message.code_id);
    message.label !== undefined && (obj.label = message.label);
    message.msg !== undefined &&
      (obj.msg = base64FromBytes(
        message.msg !== undefined ? message.msg : new Uint8Array()
      ));
    message.vaa !== undefined &&
      (obj.vaa = base64FromBytes(
        message.vaa !== undefined ? message.vaa : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgInstantiateContract>
  ): MsgInstantiateContract {
    const message = { ...baseMsgInstantiateContract } as MsgInstantiateContract;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = object.signer;
    } else {
      message.signer = "";
    }
    if (object.code_id !== undefined && object.code_id !== null) {
      message.code_id = object.code_id;
    } else {
      message.code_id = 0;
    }
    if (object.label !== undefined && object.label !== null) {
      message.label = object.label;
    } else {
      message.label = "";
    }
    if (object.msg !== undefined && object.msg !== null) {
      message.msg = object.msg;
    } else {
      message.msg = new Uint8Array();
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = object.vaa;
    } else {
      message.vaa = new Uint8Array();
    }
    return message;
  },
};

const baseMsgInstantiateContractResponse: object = { address: "" };

export const MsgInstantiateContractResponse = {
  encode(
    message: MsgInstantiateContractResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.data.length !== 0) {
      writer.uint32(18).bytes(message.data);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgInstantiateContractResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgInstantiateContractResponse,
    } as MsgInstantiateContractResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.address = reader.string();
          break;
        case 2:
          message.data = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgInstantiateContractResponse {
    const message = {
      ...baseMsgInstantiateContractResponse,
    } as MsgInstantiateContractResponse;
    if (object.address !== undefined && object.address !== null) {
      message.address = String(object.address);
    } else {
      message.address = "";
    }
    if (object.data !== undefined && object.data !== null) {
      message.data = bytesFromBase64(object.data);
    }
    return message;
  },

  toJSON(message: MsgInstantiateContractResponse): unknown {
    const obj: any = {};
    message.address !== undefined && (obj.address = message.address);
    message.data !== undefined &&
      (obj.data = base64FromBytes(
        message.data !== undefined ? message.data : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgInstantiateContractResponse>
  ): MsgInstantiateContractResponse {
    const message = {
      ...baseMsgInstantiateContractResponse,
    } as MsgInstantiateContractResponse;
    if (object.address !== undefined && object.address !== null) {
      message.address = object.address;
    } else {
      message.address = "";
    }
    if (object.data !== undefined && object.data !== null) {
      message.data = object.data;
    } else {
      message.data = new Uint8Array();
    }
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  ExecuteGovernanceVAA(
    request: MsgExecuteGovernanceVAA
  ): Promise<MsgExecuteGovernanceVAAResponse>;
  RegisterAccountAsGuardian(
    request: MsgRegisterAccountAsGuardian
  ): Promise<MsgRegisterAccountAsGuardianResponse>;
  /** StoreCode to submit Wasm code to the system */
  StoreCode(request: MsgStoreCode): Promise<MsgStoreCodeResponse>;
  /** Instantiate creates a new smart contract instance for the given code id. */
  InstantiateContract(
    request: MsgInstantiateContract
  ): Promise<MsgInstantiateContractResponse>;
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
      "wormhole_foundation.wormchain.wormhole.Msg",
      "ExecuteGovernanceVAA",
      data
    );
    return promise.then((data) =>
      MsgExecuteGovernanceVAAResponse.decode(new Reader(data))
    );
  }

  RegisterAccountAsGuardian(
    request: MsgRegisterAccountAsGuardian
  ): Promise<MsgRegisterAccountAsGuardianResponse> {
    const data = MsgRegisterAccountAsGuardian.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Msg",
      "RegisterAccountAsGuardian",
      data
    );
    return promise.then((data) =>
      MsgRegisterAccountAsGuardianResponse.decode(new Reader(data))
    );
  }

  StoreCode(request: MsgStoreCode): Promise<MsgStoreCodeResponse> {
    const data = MsgStoreCode.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Msg",
      "StoreCode",
      data
    );
    return promise.then((data) =>
      MsgStoreCodeResponse.decode(new Reader(data))
    );
  }

  InstantiateContract(
    request: MsgInstantiateContract
  ): Promise<MsgInstantiateContractResponse> {
    const data = MsgInstantiateContract.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Msg",
      "InstantiateContract",
      data
    );
    return promise.then((data) =>
      MsgInstantiateContractResponse.decode(new Reader(data))
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
