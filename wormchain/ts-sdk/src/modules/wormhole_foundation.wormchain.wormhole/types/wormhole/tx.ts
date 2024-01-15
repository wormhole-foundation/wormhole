//@ts-nocheck
/* eslint-disable */
import { Reader, util, configure, Writer } from "protobufjs/minimal";
import * as Long from "long";

export const protobufPackage = "wormhole_foundation.wormchain.wormhole";

export interface EmptyResponse {}

export interface MsgCreateAllowlistEntryRequest {
  /** signer should be a guardian validator in a current set or future set. */
  signer: string;
  /** the address to allowlist */
  address: string;
  /** optional human readable name for the entry */
  name: string;
}

export interface MsgDeleteAllowlistEntryRequest {
  /** signer should be a guardian validator in a current set or future set. */
  signer: string;
  /** the address allowlist to remove */
  address: string;
}

export interface MsgAllowlistResponse {}

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
  /** Checksum is the sha256 hash of the stored code */
  checksum: Uint8Array;
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
  /** vaa must be governance msg with payload containing keccak256 hash(hash(hash(BigEndian(CodeID)), Label), Msg) */
  vaa: Uint8Array;
}

export interface MsgInstantiateContractResponse {
  /** Address is the bech32 address of the new contract instance. */
  address: string;
  /** Data contains base64-encoded bytes to returned from the contract */
  data: Uint8Array;
}

export interface MsgAddWasmInstantiateAllowlist {
  /** Signer is the actor that signed the messages */
  signer: string;
  /** Address is the bech32 address of the contract that can call wasm instantiate without a VAA */
  address: string;
  /** CodeID is the reference to the stored WASM code that can be instantiated */
  code_id: number;
  /** vaa is the WormchainAddWasmInstantiateAllowlist governance message */
  vaa: Uint8Array;
}

export interface MsgDeleteWasmInstantiateAllowlist {
  /** signer should be a guardian validator in a current set or future set. */
  signer: string;
  /** the <contract, code_id> pair to remove */
  address: string;
  code_id: number;
  /** vaa is the WormchainDeleteWasmInstantiateAllowlist governance message */
  vaa: Uint8Array;
}

export interface MsgWasmInstantiateAllowlistResponse {}

/** MsgMigrateContract runs a code upgrade/ downgrade for a smart contract */
export interface MsgMigrateContract {
  /** Sender is the actor that signs the messages */
  signer: string;
  /** Contract is the address of the smart contract */
  contract: string;
  /** CodeID references the new WASM code */
  code_id: number;
  /** Msg json encoded message to be passed to the contract on migration */
  msg: Uint8Array;
  /** vaa must be governance msg with payload containing keccak256 hash(hash(hash(BigEndian(CodeID)), Contract), Msg) */
  vaa: Uint8Array;
}

/** MsgMigrateContractResponse returns contract migration result data. */
export interface MsgMigrateContractResponse {
  /**
   * Data contains same raw bytes returned as data from the wasm contract.
   * (May be empty)
   */
  data: Uint8Array;
}

export interface MsgExecuteGatewayGovernanceVaa {
  /** Sender is the actor that signs the messages */
  signer: string;
  /** vaa must be governance msg with valid module, action, and payload */
  vaa: Uint8Array;
}

const baseEmptyResponse: object = {};

export const EmptyResponse = {
  encode(_: EmptyResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): EmptyResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseEmptyResponse } as EmptyResponse;
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

  fromJSON(_: any): EmptyResponse {
    const message = { ...baseEmptyResponse } as EmptyResponse;
    return message;
  },

  toJSON(_: EmptyResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<EmptyResponse>): EmptyResponse {
    const message = { ...baseEmptyResponse } as EmptyResponse;
    return message;
  },
};

const baseMsgCreateAllowlistEntryRequest: object = {
  signer: "",
  address: "",
  name: "",
};

export const MsgCreateAllowlistEntryRequest = {
  encode(
    message: MsgCreateAllowlistEntryRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    if (message.name !== "") {
      writer.uint32(26).string(message.name);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgCreateAllowlistEntryRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgCreateAllowlistEntryRequest,
    } as MsgCreateAllowlistEntryRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 2:
          message.address = reader.string();
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

  fromJSON(object: any): MsgCreateAllowlistEntryRequest {
    const message = {
      ...baseMsgCreateAllowlistEntryRequest,
    } as MsgCreateAllowlistEntryRequest;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = String(object.signer);
    } else {
      message.signer = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = String(object.address);
    } else {
      message.address = "";
    }
    if (object.name !== undefined && object.name !== null) {
      message.name = String(object.name);
    } else {
      message.name = "";
    }
    return message;
  },

  toJSON(message: MsgCreateAllowlistEntryRequest): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.address !== undefined && (obj.address = message.address);
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgCreateAllowlistEntryRequest>
  ): MsgCreateAllowlistEntryRequest {
    const message = {
      ...baseMsgCreateAllowlistEntryRequest,
    } as MsgCreateAllowlistEntryRequest;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = object.signer;
    } else {
      message.signer = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = object.address;
    } else {
      message.address = "";
    }
    if (object.name !== undefined && object.name !== null) {
      message.name = object.name;
    } else {
      message.name = "";
    }
    return message;
  },
};

const baseMsgDeleteAllowlistEntryRequest: object = { signer: "", address: "" };

export const MsgDeleteAllowlistEntryRequest = {
  encode(
    message: MsgDeleteAllowlistEntryRequest,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgDeleteAllowlistEntryRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgDeleteAllowlistEntryRequest,
    } as MsgDeleteAllowlistEntryRequest;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 2:
          message.address = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgDeleteAllowlistEntryRequest {
    const message = {
      ...baseMsgDeleteAllowlistEntryRequest,
    } as MsgDeleteAllowlistEntryRequest;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = String(object.signer);
    } else {
      message.signer = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = String(object.address);
    } else {
      message.address = "";
    }
    return message;
  },

  toJSON(message: MsgDeleteAllowlistEntryRequest): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.address !== undefined && (obj.address = message.address);
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgDeleteAllowlistEntryRequest>
  ): MsgDeleteAllowlistEntryRequest {
    const message = {
      ...baseMsgDeleteAllowlistEntryRequest,
    } as MsgDeleteAllowlistEntryRequest;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = object.signer;
    } else {
      message.signer = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = object.address;
    } else {
      message.address = "";
    }
    return message;
  },
};

const baseMsgAllowlistResponse: object = {};

export const MsgAllowlistResponse = {
  encode(_: MsgAllowlistResponse, writer: Writer = Writer.create()): Writer {
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgAllowlistResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgAllowlistResponse } as MsgAllowlistResponse;
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

  fromJSON(_: any): MsgAllowlistResponse {
    const message = { ...baseMsgAllowlistResponse } as MsgAllowlistResponse;
    return message;
  },

  toJSON(_: MsgAllowlistResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<MsgAllowlistResponse>): MsgAllowlistResponse {
    const message = { ...baseMsgAllowlistResponse } as MsgAllowlistResponse;
    return message;
  },
};

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
    if (message.checksum.length !== 0) {
      writer.uint32(18).bytes(message.checksum);
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
        case 2:
          message.checksum = reader.bytes();
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
    if (object.checksum !== undefined && object.checksum !== null) {
      message.checksum = bytesFromBase64(object.checksum);
    }
    return message;
  },

  toJSON(message: MsgStoreCodeResponse): unknown {
    const obj: any = {};
    message.code_id !== undefined && (obj.code_id = message.code_id);
    message.checksum !== undefined &&
      (obj.checksum = base64FromBytes(
        message.checksum !== undefined ? message.checksum : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(object: DeepPartial<MsgStoreCodeResponse>): MsgStoreCodeResponse {
    const message = { ...baseMsgStoreCodeResponse } as MsgStoreCodeResponse;
    if (object.code_id !== undefined && object.code_id !== null) {
      message.code_id = object.code_id;
    } else {
      message.code_id = 0;
    }
    if (object.checksum !== undefined && object.checksum !== null) {
      message.checksum = object.checksum;
    } else {
      message.checksum = new Uint8Array();
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

const baseMsgAddWasmInstantiateAllowlist: object = {
  signer: "",
  address: "",
  code_id: 0,
};

export const MsgAddWasmInstantiateAllowlist = {
  encode(
    message: MsgAddWasmInstantiateAllowlist,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    if (message.code_id !== 0) {
      writer.uint32(24).uint64(message.code_id);
    }
    if (message.vaa.length !== 0) {
      writer.uint32(34).bytes(message.vaa);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgAddWasmInstantiateAllowlist {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgAddWasmInstantiateAllowlist,
    } as MsgAddWasmInstantiateAllowlist;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 2:
          message.address = reader.string();
          break;
        case 3:
          message.code_id = longToNumber(reader.uint64() as Long);
          break;
        case 4:
          message.vaa = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgAddWasmInstantiateAllowlist {
    const message = {
      ...baseMsgAddWasmInstantiateAllowlist,
    } as MsgAddWasmInstantiateAllowlist;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = String(object.signer);
    } else {
      message.signer = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = String(object.address);
    } else {
      message.address = "";
    }
    if (object.code_id !== undefined && object.code_id !== null) {
      message.code_id = Number(object.code_id);
    } else {
      message.code_id = 0;
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = bytesFromBase64(object.vaa);
    }
    return message;
  },

  toJSON(message: MsgAddWasmInstantiateAllowlist): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.address !== undefined && (obj.address = message.address);
    message.code_id !== undefined && (obj.code_id = message.code_id);
    message.vaa !== undefined &&
      (obj.vaa = base64FromBytes(
        message.vaa !== undefined ? message.vaa : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgAddWasmInstantiateAllowlist>
  ): MsgAddWasmInstantiateAllowlist {
    const message = {
      ...baseMsgAddWasmInstantiateAllowlist,
    } as MsgAddWasmInstantiateAllowlist;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = object.signer;
    } else {
      message.signer = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = object.address;
    } else {
      message.address = "";
    }
    if (object.code_id !== undefined && object.code_id !== null) {
      message.code_id = object.code_id;
    } else {
      message.code_id = 0;
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = object.vaa;
    } else {
      message.vaa = new Uint8Array();
    }
    return message;
  },
};

const baseMsgDeleteWasmInstantiateAllowlist: object = {
  signer: "",
  address: "",
  code_id: 0,
};

export const MsgDeleteWasmInstantiateAllowlist = {
  encode(
    message: MsgDeleteWasmInstantiateAllowlist,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    if (message.code_id !== 0) {
      writer.uint32(24).uint64(message.code_id);
    }
    if (message.vaa.length !== 0) {
      writer.uint32(34).bytes(message.vaa);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgDeleteWasmInstantiateAllowlist {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgDeleteWasmInstantiateAllowlist,
    } as MsgDeleteWasmInstantiateAllowlist;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 2:
          message.address = reader.string();
          break;
        case 3:
          message.code_id = longToNumber(reader.uint64() as Long);
          break;
        case 4:
          message.vaa = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgDeleteWasmInstantiateAllowlist {
    const message = {
      ...baseMsgDeleteWasmInstantiateAllowlist,
    } as MsgDeleteWasmInstantiateAllowlist;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = String(object.signer);
    } else {
      message.signer = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = String(object.address);
    } else {
      message.address = "";
    }
    if (object.code_id !== undefined && object.code_id !== null) {
      message.code_id = Number(object.code_id);
    } else {
      message.code_id = 0;
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = bytesFromBase64(object.vaa);
    }
    return message;
  },

  toJSON(message: MsgDeleteWasmInstantiateAllowlist): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.address !== undefined && (obj.address = message.address);
    message.code_id !== undefined && (obj.code_id = message.code_id);
    message.vaa !== undefined &&
      (obj.vaa = base64FromBytes(
        message.vaa !== undefined ? message.vaa : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgDeleteWasmInstantiateAllowlist>
  ): MsgDeleteWasmInstantiateAllowlist {
    const message = {
      ...baseMsgDeleteWasmInstantiateAllowlist,
    } as MsgDeleteWasmInstantiateAllowlist;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = object.signer;
    } else {
      message.signer = "";
    }
    if (object.address !== undefined && object.address !== null) {
      message.address = object.address;
    } else {
      message.address = "";
    }
    if (object.code_id !== undefined && object.code_id !== null) {
      message.code_id = object.code_id;
    } else {
      message.code_id = 0;
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = object.vaa;
    } else {
      message.vaa = new Uint8Array();
    }
    return message;
  },
};

const baseMsgWasmInstantiateAllowlistResponse: object = {};

export const MsgWasmInstantiateAllowlistResponse = {
  encode(
    _: MsgWasmInstantiateAllowlistResponse,
    writer: Writer = Writer.create()
  ): Writer {
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgWasmInstantiateAllowlistResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgWasmInstantiateAllowlistResponse,
    } as MsgWasmInstantiateAllowlistResponse;
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

  fromJSON(_: any): MsgWasmInstantiateAllowlistResponse {
    const message = {
      ...baseMsgWasmInstantiateAllowlistResponse,
    } as MsgWasmInstantiateAllowlistResponse;
    return message;
  },

  toJSON(_: MsgWasmInstantiateAllowlistResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(
    _: DeepPartial<MsgWasmInstantiateAllowlistResponse>
  ): MsgWasmInstantiateAllowlistResponse {
    const message = {
      ...baseMsgWasmInstantiateAllowlistResponse,
    } as MsgWasmInstantiateAllowlistResponse;
    return message;
  },
};

const baseMsgMigrateContract: object = { signer: "", contract: "", code_id: 0 };

export const MsgMigrateContract = {
  encode(
    message: MsgMigrateContract,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.contract !== "") {
      writer.uint32(18).string(message.contract);
    }
    if (message.code_id !== 0) {
      writer.uint32(24).uint64(message.code_id);
    }
    if (message.msg.length !== 0) {
      writer.uint32(34).bytes(message.msg);
    }
    if (message.vaa.length !== 0) {
      writer.uint32(50).bytes(message.vaa);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): MsgMigrateContract {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseMsgMigrateContract } as MsgMigrateContract;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 2:
          message.contract = reader.string();
          break;
        case 3:
          message.code_id = longToNumber(reader.uint64() as Long);
          break;
        case 4:
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

  fromJSON(object: any): MsgMigrateContract {
    const message = { ...baseMsgMigrateContract } as MsgMigrateContract;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = String(object.signer);
    } else {
      message.signer = "";
    }
    if (object.contract !== undefined && object.contract !== null) {
      message.contract = String(object.contract);
    } else {
      message.contract = "";
    }
    if (object.code_id !== undefined && object.code_id !== null) {
      message.code_id = Number(object.code_id);
    } else {
      message.code_id = 0;
    }
    if (object.msg !== undefined && object.msg !== null) {
      message.msg = bytesFromBase64(object.msg);
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = bytesFromBase64(object.vaa);
    }
    return message;
  },

  toJSON(message: MsgMigrateContract): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.contract !== undefined && (obj.contract = message.contract);
    message.code_id !== undefined && (obj.code_id = message.code_id);
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

  fromPartial(object: DeepPartial<MsgMigrateContract>): MsgMigrateContract {
    const message = { ...baseMsgMigrateContract } as MsgMigrateContract;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = object.signer;
    } else {
      message.signer = "";
    }
    if (object.contract !== undefined && object.contract !== null) {
      message.contract = object.contract;
    } else {
      message.contract = "";
    }
    if (object.code_id !== undefined && object.code_id !== null) {
      message.code_id = object.code_id;
    } else {
      message.code_id = 0;
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

const baseMsgMigrateContractResponse: object = {};

export const MsgMigrateContractResponse = {
  encode(
    message: MsgMigrateContractResponse,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.data.length !== 0) {
      writer.uint32(10).bytes(message.data);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgMigrateContractResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgMigrateContractResponse,
    } as MsgMigrateContractResponse;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.data = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgMigrateContractResponse {
    const message = {
      ...baseMsgMigrateContractResponse,
    } as MsgMigrateContractResponse;
    if (object.data !== undefined && object.data !== null) {
      message.data = bytesFromBase64(object.data);
    }
    return message;
  },

  toJSON(message: MsgMigrateContractResponse): unknown {
    const obj: any = {};
    message.data !== undefined &&
      (obj.data = base64FromBytes(
        message.data !== undefined ? message.data : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgMigrateContractResponse>
  ): MsgMigrateContractResponse {
    const message = {
      ...baseMsgMigrateContractResponse,
    } as MsgMigrateContractResponse;
    if (object.data !== undefined && object.data !== null) {
      message.data = object.data;
    } else {
      message.data = new Uint8Array();
    }
    return message;
  },
};

const baseMsgExecuteGatewayGovernanceVaa: object = { signer: "" };

export const MsgExecuteGatewayGovernanceVaa = {
  encode(
    message: MsgExecuteGatewayGovernanceVaa,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.vaa.length !== 0) {
      writer.uint32(18).bytes(message.vaa);
    }
    return writer;
  },

  decode(
    input: Reader | Uint8Array,
    length?: number
  ): MsgExecuteGatewayGovernanceVaa {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = {
      ...baseMsgExecuteGatewayGovernanceVaa,
    } as MsgExecuteGatewayGovernanceVaa;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
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

  fromJSON(object: any): MsgExecuteGatewayGovernanceVaa {
    const message = {
      ...baseMsgExecuteGatewayGovernanceVaa,
    } as MsgExecuteGatewayGovernanceVaa;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = String(object.signer);
    } else {
      message.signer = "";
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = bytesFromBase64(object.vaa);
    }
    return message;
  },

  toJSON(message: MsgExecuteGatewayGovernanceVaa): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.vaa !== undefined &&
      (obj.vaa = base64FromBytes(
        message.vaa !== undefined ? message.vaa : new Uint8Array()
      ));
    return obj;
  },

  fromPartial(
    object: DeepPartial<MsgExecuteGatewayGovernanceVaa>
  ): MsgExecuteGatewayGovernanceVaa {
    const message = {
      ...baseMsgExecuteGatewayGovernanceVaa,
    } as MsgExecuteGatewayGovernanceVaa;
    if (object.signer !== undefined && object.signer !== null) {
      message.signer = object.signer;
    } else {
      message.signer = "";
    }
    if (object.vaa !== undefined && object.vaa !== null) {
      message.vaa = object.vaa;
    } else {
      message.vaa = new Uint8Array();
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
  CreateAllowlistEntry(
    request: MsgCreateAllowlistEntryRequest
  ): Promise<MsgAllowlistResponse>;
  DeleteAllowlistEntry(
    request: MsgDeleteAllowlistEntryRequest
  ): Promise<MsgAllowlistResponse>;
  /** StoreCode to submit Wasm code to the system */
  StoreCode(request: MsgStoreCode): Promise<MsgStoreCodeResponse>;
  /** Instantiate creates a new smart contract instance for the given code id. */
  InstantiateContract(
    request: MsgInstantiateContract
  ): Promise<MsgInstantiateContractResponse>;
  AddWasmInstantiateAllowlist(
    request: MsgAddWasmInstantiateAllowlist
  ): Promise<MsgWasmInstantiateAllowlistResponse>;
  DeleteWasmInstantiateAllowlist(
    request: MsgDeleteWasmInstantiateAllowlist
  ): Promise<MsgWasmInstantiateAllowlistResponse>;
  MigrateContract(
    request: MsgMigrateContract
  ): Promise<MsgMigrateContractResponse>;
  /** this line is used by starport scaffolding # proto/tx/rpc */
  ExecuteGatewayGovernanceVaa(
    request: MsgExecuteGatewayGovernanceVaa
  ): Promise<EmptyResponse>;
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

  CreateAllowlistEntry(
    request: MsgCreateAllowlistEntryRequest
  ): Promise<MsgAllowlistResponse> {
    const data = MsgCreateAllowlistEntryRequest.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Msg",
      "CreateAllowlistEntry",
      data
    );
    return promise.then((data) =>
      MsgAllowlistResponse.decode(new Reader(data))
    );
  }

  DeleteAllowlistEntry(
    request: MsgDeleteAllowlistEntryRequest
  ): Promise<MsgAllowlistResponse> {
    const data = MsgDeleteAllowlistEntryRequest.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Msg",
      "DeleteAllowlistEntry",
      data
    );
    return promise.then((data) =>
      MsgAllowlistResponse.decode(new Reader(data))
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

  AddWasmInstantiateAllowlist(
    request: MsgAddWasmInstantiateAllowlist
  ): Promise<MsgWasmInstantiateAllowlistResponse> {
    const data = MsgAddWasmInstantiateAllowlist.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Msg",
      "AddWasmInstantiateAllowlist",
      data
    );
    return promise.then((data) =>
      MsgWasmInstantiateAllowlistResponse.decode(new Reader(data))
    );
  }

  DeleteWasmInstantiateAllowlist(
    request: MsgDeleteWasmInstantiateAllowlist
  ): Promise<MsgWasmInstantiateAllowlistResponse> {
    const data = MsgDeleteWasmInstantiateAllowlist.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Msg",
      "DeleteWasmInstantiateAllowlist",
      data
    );
    return promise.then((data) =>
      MsgWasmInstantiateAllowlistResponse.decode(new Reader(data))
    );
  }

  MigrateContract(
    request: MsgMigrateContract
  ): Promise<MsgMigrateContractResponse> {
    const data = MsgMigrateContract.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Msg",
      "MigrateContract",
      data
    );
    return promise.then((data) =>
      MsgMigrateContractResponse.decode(new Reader(data))
    );
  }

  ExecuteGatewayGovernanceVaa(
    request: MsgExecuteGatewayGovernanceVaa
  ): Promise<EmptyResponse> {
    const data = MsgExecuteGatewayGovernanceVaa.encode(request).finish();
    const promise = this.rpc.request(
      "wormhole_foundation.wormchain.wormhole.Msg",
      "ExecuteGatewayGovernanceVaa",
      data
    );
    return promise.then((data) => EmptyResponse.decode(new Reader(data)));
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
