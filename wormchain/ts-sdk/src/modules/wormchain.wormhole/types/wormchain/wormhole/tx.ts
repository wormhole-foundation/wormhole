//@ts-nocheck
/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { GuardianSet } from "./guardian";

export const protobufPackage = "wormchain.wormhole";

export interface EmptyResponse {
}

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

export interface MsgAllowlistResponse {
}

export interface MsgExecuteGovernanceVAA {
  vaa: Uint8Array;
  signer: string;
}

export interface MsgExecuteGovernanceVAAResponse {
}

export interface MsgRegisterAccountAsGuardian {
  signer: string;
  signature: Uint8Array;
}

export interface MsgRegisterAccountAsGuardianResponse {
}

/** Same as from x/wasmd but with vaa auth */
export interface MsgStoreCode {
  /** Signer is the that actor that signed the messages */
  signer: string;
  /** WASMByteCode can be raw or gzip compressed */
  wasmByteCode: Uint8Array;
  /**
   * vaa must be governance msg with payload containing sha3 256 hash of
   * `wasm_byte_code`
   */
  vaa: Uint8Array;
}

export interface MsgStoreCodeResponse {
  /** CodeID is the reference to the stored WASM code */
  codeId: number;
  /** Checksum is the sha256 hash of the stored code */
  checksum: Uint8Array;
}

/** Same as from x/wasmd but with vaa auth */
export interface MsgInstantiateContract {
  /** Signer is the that actor that signed the messages */
  signer: string;
  /** CodeID is the reference to the stored WASM code */
  codeId: number;
  /** Label is optional metadata to be stored with a contract instance. */
  label: string;
  /** Msg json encoded message to be passed to the contract on instantiation */
  msg: Uint8Array;
  /**
   * vaa must be governance msg with payload containing keccak256
   * hash(hash(hash(BigEndian(CodeID)), Label), Msg)
   */
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
  /**
   * Address is the bech32 address of the contract that can call wasm
   * instantiate without a VAA
   */
  address: string;
  /** CodeID is the reference to the stored WASM code that can be instantiated */
  codeId: number;
  /** vaa is the WormchainAddWasmInstantiateAllowlist governance message */
  vaa: Uint8Array;
}

export interface MsgDeleteWasmInstantiateAllowlist {
  /** signer should be a guardian validator in a current set or future set. */
  signer: string;
  /** the <contract, code_id> pair to remove */
  address: string;
  codeId: number;
  /** vaa is the WormchainDeleteWasmInstantiateAllowlist governance message */
  vaa: Uint8Array;
}

export interface MsgWasmInstantiateAllowlistResponse {
}

/** MsgMigrateContract runs a code upgrade/ downgrade for a smart contract */
export interface MsgMigrateContract {
  /** Sender is the actor that signs the messages */
  signer: string;
  /** Contract is the address of the smart contract */
  contract: string;
  /** CodeID references the new WASM code */
  codeId: number;
  /** Msg json encoded message to be passed to the contract on migration */
  msg: Uint8Array;
  /**
   * vaa must be governance msg with payload containing keccak256
   * hash(hash(hash(BigEndian(CodeID)), Contract), Msg)
   */
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

/** GuardianSetUpdateProposal defines a guardian set update governance proposal */
export interface MsgGuardianSetUpdateProposal {
  /** authority is the address that controls the module (defaults to x/gov unless overwritten). */
  authority: string;
  newGuardianSet: GuardianSet | undefined;
}

/**
 * GovernanceWormholeMessageProposal defines a governance proposal to emit a
 * generic message in the governance message format.
 */
export interface MsgGovernanceWormholeMessageProposal {
  /** authority is the address that controls the module (defaults to x/gov unless overwritten). */
  authority: string;
  action: number;
  module: Uint8Array;
  targetChain: number;
  payload: Uint8Array;
}

function createBaseEmptyResponse(): EmptyResponse {
  return {};
}

export const EmptyResponse = {
  encode(_: EmptyResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): EmptyResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEmptyResponse();
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
    return {};
  },

  toJSON(_: EmptyResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<EmptyResponse>, I>>(_: I): EmptyResponse {
    const message = createBaseEmptyResponse();
    return message;
  },
};

function createBaseMsgCreateAllowlistEntryRequest(): MsgCreateAllowlistEntryRequest {
  return { signer: "", address: "", name: "" };
}

export const MsgCreateAllowlistEntryRequest = {
  encode(message: MsgCreateAllowlistEntryRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgCreateAllowlistEntryRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateAllowlistEntryRequest();
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
    return {
      signer: isSet(object.signer) ? String(object.signer) : "",
      address: isSet(object.address) ? String(object.address) : "",
      name: isSet(object.name) ? String(object.name) : "",
    };
  },

  toJSON(message: MsgCreateAllowlistEntryRequest): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.address !== undefined && (obj.address = message.address);
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgCreateAllowlistEntryRequest>, I>>(
    object: I,
  ): MsgCreateAllowlistEntryRequest {
    const message = createBaseMsgCreateAllowlistEntryRequest();
    message.signer = object.signer ?? "";
    message.address = object.address ?? "";
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseMsgDeleteAllowlistEntryRequest(): MsgDeleteAllowlistEntryRequest {
  return { signer: "", address: "" };
}

export const MsgDeleteAllowlistEntryRequest = {
  encode(message: MsgDeleteAllowlistEntryRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgDeleteAllowlistEntryRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDeleteAllowlistEntryRequest();
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
    return {
      signer: isSet(object.signer) ? String(object.signer) : "",
      address: isSet(object.address) ? String(object.address) : "",
    };
  },

  toJSON(message: MsgDeleteAllowlistEntryRequest): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.address !== undefined && (obj.address = message.address);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgDeleteAllowlistEntryRequest>, I>>(
    object: I,
  ): MsgDeleteAllowlistEntryRequest {
    const message = createBaseMsgDeleteAllowlistEntryRequest();
    message.signer = object.signer ?? "";
    message.address = object.address ?? "";
    return message;
  },
};

function createBaseMsgAllowlistResponse(): MsgAllowlistResponse {
  return {};
}

export const MsgAllowlistResponse = {
  encode(_: MsgAllowlistResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgAllowlistResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAllowlistResponse();
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
    return {};
  },

  toJSON(_: MsgAllowlistResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgAllowlistResponse>, I>>(_: I): MsgAllowlistResponse {
    const message = createBaseMsgAllowlistResponse();
    return message;
  },
};

function createBaseMsgExecuteGovernanceVAA(): MsgExecuteGovernanceVAA {
  return { vaa: new Uint8Array(), signer: "" };
}

export const MsgExecuteGovernanceVAA = {
  encode(message: MsgExecuteGovernanceVAA, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.vaa.length !== 0) {
      writer.uint32(10).bytes(message.vaa);
    }
    if (message.signer !== "") {
      writer.uint32(18).string(message.signer);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgExecuteGovernanceVAA {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgExecuteGovernanceVAA();
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
    return {
      vaa: isSet(object.vaa) ? bytesFromBase64(object.vaa) : new Uint8Array(),
      signer: isSet(object.signer) ? String(object.signer) : "",
    };
  },

  toJSON(message: MsgExecuteGovernanceVAA): unknown {
    const obj: any = {};
    message.vaa !== undefined
      && (obj.vaa = base64FromBytes(message.vaa !== undefined ? message.vaa : new Uint8Array()));
    message.signer !== undefined && (obj.signer = message.signer);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgExecuteGovernanceVAA>, I>>(object: I): MsgExecuteGovernanceVAA {
    const message = createBaseMsgExecuteGovernanceVAA();
    message.vaa = object.vaa ?? new Uint8Array();
    message.signer = object.signer ?? "";
    return message;
  },
};

function createBaseMsgExecuteGovernanceVAAResponse(): MsgExecuteGovernanceVAAResponse {
  return {};
}

export const MsgExecuteGovernanceVAAResponse = {
  encode(_: MsgExecuteGovernanceVAAResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgExecuteGovernanceVAAResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgExecuteGovernanceVAAResponse();
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
    return {};
  },

  toJSON(_: MsgExecuteGovernanceVAAResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgExecuteGovernanceVAAResponse>, I>>(_: I): MsgExecuteGovernanceVAAResponse {
    const message = createBaseMsgExecuteGovernanceVAAResponse();
    return message;
  },
};

function createBaseMsgRegisterAccountAsGuardian(): MsgRegisterAccountAsGuardian {
  return { signer: "", signature: new Uint8Array() };
}

export const MsgRegisterAccountAsGuardian = {
  encode(message: MsgRegisterAccountAsGuardian, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.signature.length !== 0) {
      writer.uint32(26).bytes(message.signature);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgRegisterAccountAsGuardian {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterAccountAsGuardian();
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
    return {
      signer: isSet(object.signer) ? String(object.signer) : "",
      signature: isSet(object.signature) ? bytesFromBase64(object.signature) : new Uint8Array(),
    };
  },

  toJSON(message: MsgRegisterAccountAsGuardian): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.signature !== undefined
      && (obj.signature = base64FromBytes(message.signature !== undefined ? message.signature : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgRegisterAccountAsGuardian>, I>>(object: I): MsgRegisterAccountAsGuardian {
    const message = createBaseMsgRegisterAccountAsGuardian();
    message.signer = object.signer ?? "";
    message.signature = object.signature ?? new Uint8Array();
    return message;
  },
};

function createBaseMsgRegisterAccountAsGuardianResponse(): MsgRegisterAccountAsGuardianResponse {
  return {};
}

export const MsgRegisterAccountAsGuardianResponse = {
  encode(_: MsgRegisterAccountAsGuardianResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgRegisterAccountAsGuardianResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterAccountAsGuardianResponse();
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
    return {};
  },

  toJSON(_: MsgRegisterAccountAsGuardianResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgRegisterAccountAsGuardianResponse>, I>>(
    _: I,
  ): MsgRegisterAccountAsGuardianResponse {
    const message = createBaseMsgRegisterAccountAsGuardianResponse();
    return message;
  },
};

function createBaseMsgStoreCode(): MsgStoreCode {
  return { signer: "", wasmByteCode: new Uint8Array(), vaa: new Uint8Array() };
}

export const MsgStoreCode = {
  encode(message: MsgStoreCode, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.wasmByteCode.length !== 0) {
      writer.uint32(18).bytes(message.wasmByteCode);
    }
    if (message.vaa.length !== 0) {
      writer.uint32(26).bytes(message.vaa);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgStoreCode {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgStoreCode();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 2:
          message.wasmByteCode = reader.bytes();
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
    return {
      signer: isSet(object.signer) ? String(object.signer) : "",
      wasmByteCode: isSet(object.wasmByteCode) ? bytesFromBase64(object.wasmByteCode) : new Uint8Array(),
      vaa: isSet(object.vaa) ? bytesFromBase64(object.vaa) : new Uint8Array(),
    };
  },

  toJSON(message: MsgStoreCode): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.wasmByteCode !== undefined
      && (obj.wasmByteCode = base64FromBytes(
        message.wasmByteCode !== undefined ? message.wasmByteCode : new Uint8Array(),
      ));
    message.vaa !== undefined
      && (obj.vaa = base64FromBytes(message.vaa !== undefined ? message.vaa : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgStoreCode>, I>>(object: I): MsgStoreCode {
    const message = createBaseMsgStoreCode();
    message.signer = object.signer ?? "";
    message.wasmByteCode = object.wasmByteCode ?? new Uint8Array();
    message.vaa = object.vaa ?? new Uint8Array();
    return message;
  },
};

function createBaseMsgStoreCodeResponse(): MsgStoreCodeResponse {
  return { codeId: 0, checksum: new Uint8Array() };
}

export const MsgStoreCodeResponse = {
  encode(message: MsgStoreCodeResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.codeId !== 0) {
      writer.uint32(8).uint64(message.codeId);
    }
    if (message.checksum.length !== 0) {
      writer.uint32(18).bytes(message.checksum);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgStoreCodeResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgStoreCodeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.codeId = longToNumber(reader.uint64() as Long);
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
    return {
      codeId: isSet(object.codeId) ? Number(object.codeId) : 0,
      checksum: isSet(object.checksum) ? bytesFromBase64(object.checksum) : new Uint8Array(),
    };
  },

  toJSON(message: MsgStoreCodeResponse): unknown {
    const obj: any = {};
    message.codeId !== undefined && (obj.codeId = Math.round(message.codeId));
    message.checksum !== undefined
      && (obj.checksum = base64FromBytes(message.checksum !== undefined ? message.checksum : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgStoreCodeResponse>, I>>(object: I): MsgStoreCodeResponse {
    const message = createBaseMsgStoreCodeResponse();
    message.codeId = object.codeId ?? 0;
    message.checksum = object.checksum ?? new Uint8Array();
    return message;
  },
};

function createBaseMsgInstantiateContract(): MsgInstantiateContract {
  return { signer: "", codeId: 0, label: "", msg: new Uint8Array(), vaa: new Uint8Array() };
}

export const MsgInstantiateContract = {
  encode(message: MsgInstantiateContract, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.codeId !== 0) {
      writer.uint32(24).uint64(message.codeId);
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

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgInstantiateContract {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgInstantiateContract();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 3:
          message.codeId = longToNumber(reader.uint64() as Long);
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
    return {
      signer: isSet(object.signer) ? String(object.signer) : "",
      codeId: isSet(object.codeId) ? Number(object.codeId) : 0,
      label: isSet(object.label) ? String(object.label) : "",
      msg: isSet(object.msg) ? bytesFromBase64(object.msg) : new Uint8Array(),
      vaa: isSet(object.vaa) ? bytesFromBase64(object.vaa) : new Uint8Array(),
    };
  },

  toJSON(message: MsgInstantiateContract): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.codeId !== undefined && (obj.codeId = Math.round(message.codeId));
    message.label !== undefined && (obj.label = message.label);
    message.msg !== undefined
      && (obj.msg = base64FromBytes(message.msg !== undefined ? message.msg : new Uint8Array()));
    message.vaa !== undefined
      && (obj.vaa = base64FromBytes(message.vaa !== undefined ? message.vaa : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgInstantiateContract>, I>>(object: I): MsgInstantiateContract {
    const message = createBaseMsgInstantiateContract();
    message.signer = object.signer ?? "";
    message.codeId = object.codeId ?? 0;
    message.label = object.label ?? "";
    message.msg = object.msg ?? new Uint8Array();
    message.vaa = object.vaa ?? new Uint8Array();
    return message;
  },
};

function createBaseMsgInstantiateContractResponse(): MsgInstantiateContractResponse {
  return { address: "", data: new Uint8Array() };
}

export const MsgInstantiateContractResponse = {
  encode(message: MsgInstantiateContractResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.data.length !== 0) {
      writer.uint32(18).bytes(message.data);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgInstantiateContractResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgInstantiateContractResponse();
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
    return {
      address: isSet(object.address) ? String(object.address) : "",
      data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(),
    };
  },

  toJSON(message: MsgInstantiateContractResponse): unknown {
    const obj: any = {};
    message.address !== undefined && (obj.address = message.address);
    message.data !== undefined
      && (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgInstantiateContractResponse>, I>>(
    object: I,
  ): MsgInstantiateContractResponse {
    const message = createBaseMsgInstantiateContractResponse();
    message.address = object.address ?? "";
    message.data = object.data ?? new Uint8Array();
    return message;
  },
};

function createBaseMsgAddWasmInstantiateAllowlist(): MsgAddWasmInstantiateAllowlist {
  return { signer: "", address: "", codeId: 0, vaa: new Uint8Array() };
}

export const MsgAddWasmInstantiateAllowlist = {
  encode(message: MsgAddWasmInstantiateAllowlist, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    if (message.codeId !== 0) {
      writer.uint32(24).uint64(message.codeId);
    }
    if (message.vaa.length !== 0) {
      writer.uint32(34).bytes(message.vaa);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgAddWasmInstantiateAllowlist {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddWasmInstantiateAllowlist();
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
          message.codeId = longToNumber(reader.uint64() as Long);
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
    return {
      signer: isSet(object.signer) ? String(object.signer) : "",
      address: isSet(object.address) ? String(object.address) : "",
      codeId: isSet(object.codeId) ? Number(object.codeId) : 0,
      vaa: isSet(object.vaa) ? bytesFromBase64(object.vaa) : new Uint8Array(),
    };
  },

  toJSON(message: MsgAddWasmInstantiateAllowlist): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.address !== undefined && (obj.address = message.address);
    message.codeId !== undefined && (obj.codeId = Math.round(message.codeId));
    message.vaa !== undefined
      && (obj.vaa = base64FromBytes(message.vaa !== undefined ? message.vaa : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgAddWasmInstantiateAllowlist>, I>>(
    object: I,
  ): MsgAddWasmInstantiateAllowlist {
    const message = createBaseMsgAddWasmInstantiateAllowlist();
    message.signer = object.signer ?? "";
    message.address = object.address ?? "";
    message.codeId = object.codeId ?? 0;
    message.vaa = object.vaa ?? new Uint8Array();
    return message;
  },
};

function createBaseMsgDeleteWasmInstantiateAllowlist(): MsgDeleteWasmInstantiateAllowlist {
  return { signer: "", address: "", codeId: 0, vaa: new Uint8Array() };
}

export const MsgDeleteWasmInstantiateAllowlist = {
  encode(message: MsgDeleteWasmInstantiateAllowlist, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    if (message.codeId !== 0) {
      writer.uint32(24).uint64(message.codeId);
    }
    if (message.vaa.length !== 0) {
      writer.uint32(34).bytes(message.vaa);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgDeleteWasmInstantiateAllowlist {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDeleteWasmInstantiateAllowlist();
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
          message.codeId = longToNumber(reader.uint64() as Long);
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
    return {
      signer: isSet(object.signer) ? String(object.signer) : "",
      address: isSet(object.address) ? String(object.address) : "",
      codeId: isSet(object.codeId) ? Number(object.codeId) : 0,
      vaa: isSet(object.vaa) ? bytesFromBase64(object.vaa) : new Uint8Array(),
    };
  },

  toJSON(message: MsgDeleteWasmInstantiateAllowlist): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.address !== undefined && (obj.address = message.address);
    message.codeId !== undefined && (obj.codeId = Math.round(message.codeId));
    message.vaa !== undefined
      && (obj.vaa = base64FromBytes(message.vaa !== undefined ? message.vaa : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgDeleteWasmInstantiateAllowlist>, I>>(
    object: I,
  ): MsgDeleteWasmInstantiateAllowlist {
    const message = createBaseMsgDeleteWasmInstantiateAllowlist();
    message.signer = object.signer ?? "";
    message.address = object.address ?? "";
    message.codeId = object.codeId ?? 0;
    message.vaa = object.vaa ?? new Uint8Array();
    return message;
  },
};

function createBaseMsgWasmInstantiateAllowlistResponse(): MsgWasmInstantiateAllowlistResponse {
  return {};
}

export const MsgWasmInstantiateAllowlistResponse = {
  encode(_: MsgWasmInstantiateAllowlistResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgWasmInstantiateAllowlistResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgWasmInstantiateAllowlistResponse();
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
    return {};
  },

  toJSON(_: MsgWasmInstantiateAllowlistResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgWasmInstantiateAllowlistResponse>, I>>(
    _: I,
  ): MsgWasmInstantiateAllowlistResponse {
    const message = createBaseMsgWasmInstantiateAllowlistResponse();
    return message;
  },
};

function createBaseMsgMigrateContract(): MsgMigrateContract {
  return { signer: "", contract: "", codeId: 0, msg: new Uint8Array(), vaa: new Uint8Array() };
}

export const MsgMigrateContract = {
  encode(message: MsgMigrateContract, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.contract !== "") {
      writer.uint32(18).string(message.contract);
    }
    if (message.codeId !== 0) {
      writer.uint32(24).uint64(message.codeId);
    }
    if (message.msg.length !== 0) {
      writer.uint32(34).bytes(message.msg);
    }
    if (message.vaa.length !== 0) {
      writer.uint32(50).bytes(message.vaa);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgMigrateContract {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgMigrateContract();
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
          message.codeId = longToNumber(reader.uint64() as Long);
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
    return {
      signer: isSet(object.signer) ? String(object.signer) : "",
      contract: isSet(object.contract) ? String(object.contract) : "",
      codeId: isSet(object.codeId) ? Number(object.codeId) : 0,
      msg: isSet(object.msg) ? bytesFromBase64(object.msg) : new Uint8Array(),
      vaa: isSet(object.vaa) ? bytesFromBase64(object.vaa) : new Uint8Array(),
    };
  },

  toJSON(message: MsgMigrateContract): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.contract !== undefined && (obj.contract = message.contract);
    message.codeId !== undefined && (obj.codeId = Math.round(message.codeId));
    message.msg !== undefined
      && (obj.msg = base64FromBytes(message.msg !== undefined ? message.msg : new Uint8Array()));
    message.vaa !== undefined
      && (obj.vaa = base64FromBytes(message.vaa !== undefined ? message.vaa : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgMigrateContract>, I>>(object: I): MsgMigrateContract {
    const message = createBaseMsgMigrateContract();
    message.signer = object.signer ?? "";
    message.contract = object.contract ?? "";
    message.codeId = object.codeId ?? 0;
    message.msg = object.msg ?? new Uint8Array();
    message.vaa = object.vaa ?? new Uint8Array();
    return message;
  },
};

function createBaseMsgMigrateContractResponse(): MsgMigrateContractResponse {
  return { data: new Uint8Array() };
}

export const MsgMigrateContractResponse = {
  encode(message: MsgMigrateContractResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.data.length !== 0) {
      writer.uint32(10).bytes(message.data);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgMigrateContractResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgMigrateContractResponse();
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
    return { data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array() };
  },

  toJSON(message: MsgMigrateContractResponse): unknown {
    const obj: any = {};
    message.data !== undefined
      && (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgMigrateContractResponse>, I>>(object: I): MsgMigrateContractResponse {
    const message = createBaseMsgMigrateContractResponse();
    message.data = object.data ?? new Uint8Array();
    return message;
  },
};

function createBaseMsgExecuteGatewayGovernanceVaa(): MsgExecuteGatewayGovernanceVaa {
  return { signer: "", vaa: new Uint8Array() };
}

export const MsgExecuteGatewayGovernanceVaa = {
  encode(message: MsgExecuteGatewayGovernanceVaa, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.vaa.length !== 0) {
      writer.uint32(18).bytes(message.vaa);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgExecuteGatewayGovernanceVaa {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgExecuteGatewayGovernanceVaa();
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
    return {
      signer: isSet(object.signer) ? String(object.signer) : "",
      vaa: isSet(object.vaa) ? bytesFromBase64(object.vaa) : new Uint8Array(),
    };
  },

  toJSON(message: MsgExecuteGatewayGovernanceVaa): unknown {
    const obj: any = {};
    message.signer !== undefined && (obj.signer = message.signer);
    message.vaa !== undefined
      && (obj.vaa = base64FromBytes(message.vaa !== undefined ? message.vaa : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgExecuteGatewayGovernanceVaa>, I>>(
    object: I,
  ): MsgExecuteGatewayGovernanceVaa {
    const message = createBaseMsgExecuteGatewayGovernanceVaa();
    message.signer = object.signer ?? "";
    message.vaa = object.vaa ?? new Uint8Array();
    return message;
  },
};

function createBaseMsgGuardianSetUpdateProposal(): MsgGuardianSetUpdateProposal {
  return { authority: "", newGuardianSet: undefined };
}

export const MsgGuardianSetUpdateProposal = {
  encode(message: MsgGuardianSetUpdateProposal, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.newGuardianSet !== undefined) {
      GuardianSet.encode(message.newGuardianSet, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgGuardianSetUpdateProposal {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgGuardianSetUpdateProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.newGuardianSet = GuardianSet.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgGuardianSetUpdateProposal {
    return {
      authority: isSet(object.authority) ? String(object.authority) : "",
      newGuardianSet: isSet(object.newGuardianSet) ? GuardianSet.fromJSON(object.newGuardianSet) : undefined,
    };
  },

  toJSON(message: MsgGuardianSetUpdateProposal): unknown {
    const obj: any = {};
    message.authority !== undefined && (obj.authority = message.authority);
    message.newGuardianSet !== undefined
      && (obj.newGuardianSet = message.newGuardianSet ? GuardianSet.toJSON(message.newGuardianSet) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgGuardianSetUpdateProposal>, I>>(object: I): MsgGuardianSetUpdateProposal {
    const message = createBaseMsgGuardianSetUpdateProposal();
    message.authority = object.authority ?? "";
    message.newGuardianSet = (object.newGuardianSet !== undefined && object.newGuardianSet !== null)
      ? GuardianSet.fromPartial(object.newGuardianSet)
      : undefined;
    return message;
  },
};

function createBaseMsgGovernanceWormholeMessageProposal(): MsgGovernanceWormholeMessageProposal {
  return { authority: "", action: 0, module: new Uint8Array(), targetChain: 0, payload: new Uint8Array() };
}

export const MsgGovernanceWormholeMessageProposal = {
  encode(message: MsgGovernanceWormholeMessageProposal, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.action !== 0) {
      writer.uint32(16).uint32(message.action);
    }
    if (message.module.length !== 0) {
      writer.uint32(26).bytes(message.module);
    }
    if (message.targetChain !== 0) {
      writer.uint32(32).uint32(message.targetChain);
    }
    if (message.payload.length !== 0) {
      writer.uint32(42).bytes(message.payload);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MsgGovernanceWormholeMessageProposal {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgGovernanceWormholeMessageProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.action = reader.uint32();
          break;
        case 3:
          message.module = reader.bytes();
          break;
        case 4:
          message.targetChain = reader.uint32();
          break;
        case 5:
          message.payload = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): MsgGovernanceWormholeMessageProposal {
    return {
      authority: isSet(object.authority) ? String(object.authority) : "",
      action: isSet(object.action) ? Number(object.action) : 0,
      module: isSet(object.module) ? bytesFromBase64(object.module) : new Uint8Array(),
      targetChain: isSet(object.targetChain) ? Number(object.targetChain) : 0,
      payload: isSet(object.payload) ? bytesFromBase64(object.payload) : new Uint8Array(),
    };
  },

  toJSON(message: MsgGovernanceWormholeMessageProposal): unknown {
    const obj: any = {};
    message.authority !== undefined && (obj.authority = message.authority);
    message.action !== undefined && (obj.action = Math.round(message.action));
    message.module !== undefined
      && (obj.module = base64FromBytes(message.module !== undefined ? message.module : new Uint8Array()));
    message.targetChain !== undefined && (obj.targetChain = Math.round(message.targetChain));
    message.payload !== undefined
      && (obj.payload = base64FromBytes(message.payload !== undefined ? message.payload : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<MsgGovernanceWormholeMessageProposal>, I>>(
    object: I,
  ): MsgGovernanceWormholeMessageProposal {
    const message = createBaseMsgGovernanceWormholeMessageProposal();
    message.authority = object.authority ?? "";
    message.action = object.action ?? 0;
    message.module = object.module ?? new Uint8Array();
    message.targetChain = object.targetChain ?? 0;
    message.payload = object.payload ?? new Uint8Array();
    return message;
  },
};

/** Msg defines the Msg service. */
export interface Msg {
  ExecuteGovernanceVAA(request: MsgExecuteGovernanceVAA): Promise<MsgExecuteGovernanceVAAResponse>;
  RegisterAccountAsGuardian(request: MsgRegisterAccountAsGuardian): Promise<MsgRegisterAccountAsGuardianResponse>;
  CreateAllowlistEntry(request: MsgCreateAllowlistEntryRequest): Promise<MsgAllowlistResponse>;
  DeleteAllowlistEntry(request: MsgDeleteAllowlistEntryRequest): Promise<MsgAllowlistResponse>;
  /** StoreCode to submit Wasm code to the system */
  StoreCode(request: MsgStoreCode): Promise<MsgStoreCodeResponse>;
  /** Instantiate creates a new smart contract instance for the given code id. */
  InstantiateContract(request: MsgInstantiateContract): Promise<MsgInstantiateContractResponse>;
  AddWasmInstantiateAllowlist(request: MsgAddWasmInstantiateAllowlist): Promise<MsgWasmInstantiateAllowlistResponse>;
  DeleteWasmInstantiateAllowlist(
    request: MsgDeleteWasmInstantiateAllowlist,
  ): Promise<MsgWasmInstantiateAllowlistResponse>;
  MigrateContract(request: MsgMigrateContract): Promise<MsgMigrateContractResponse>;
  ExecuteGatewayGovernanceVaa(request: MsgExecuteGatewayGovernanceVaa): Promise<EmptyResponse>;
  /** GuardianSetUpdateProposal processes a proposal to update the guardian set */
  GuardianSetUpdateProposal(request: MsgGuardianSetUpdateProposal): Promise<EmptyResponse>;
  /** GovernanceWormholeMessageProposal processes a proposal to emit a generic message */
  GovernanceWormholeMessageProposal(request: MsgGovernanceWormholeMessageProposal): Promise<EmptyResponse>;
}

export class MsgClientImpl implements Msg {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
    this.ExecuteGovernanceVAA = this.ExecuteGovernanceVAA.bind(this);
    this.RegisterAccountAsGuardian = this.RegisterAccountAsGuardian.bind(this);
    this.CreateAllowlistEntry = this.CreateAllowlistEntry.bind(this);
    this.DeleteAllowlistEntry = this.DeleteAllowlistEntry.bind(this);
    this.StoreCode = this.StoreCode.bind(this);
    this.InstantiateContract = this.InstantiateContract.bind(this);
    this.AddWasmInstantiateAllowlist = this.AddWasmInstantiateAllowlist.bind(this);
    this.DeleteWasmInstantiateAllowlist = this.DeleteWasmInstantiateAllowlist.bind(this);
    this.MigrateContract = this.MigrateContract.bind(this);
    this.ExecuteGatewayGovernanceVaa = this.ExecuteGatewayGovernanceVaa.bind(this);
    this.GuardianSetUpdateProposal = this.GuardianSetUpdateProposal.bind(this);
    this.GovernanceWormholeMessageProposal = this.GovernanceWormholeMessageProposal.bind(this);
  }
  ExecuteGovernanceVAA(request: MsgExecuteGovernanceVAA): Promise<MsgExecuteGovernanceVAAResponse> {
    const data = MsgExecuteGovernanceVAA.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Msg", "ExecuteGovernanceVAA", data);
    return promise.then((data) => MsgExecuteGovernanceVAAResponse.decode(new _m0.Reader(data)));
  }

  RegisterAccountAsGuardian(request: MsgRegisterAccountAsGuardian): Promise<MsgRegisterAccountAsGuardianResponse> {
    const data = MsgRegisterAccountAsGuardian.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Msg", "RegisterAccountAsGuardian", data);
    return promise.then((data) => MsgRegisterAccountAsGuardianResponse.decode(new _m0.Reader(data)));
  }

  CreateAllowlistEntry(request: MsgCreateAllowlistEntryRequest): Promise<MsgAllowlistResponse> {
    const data = MsgCreateAllowlistEntryRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Msg", "CreateAllowlistEntry", data);
    return promise.then((data) => MsgAllowlistResponse.decode(new _m0.Reader(data)));
  }

  DeleteAllowlistEntry(request: MsgDeleteAllowlistEntryRequest): Promise<MsgAllowlistResponse> {
    const data = MsgDeleteAllowlistEntryRequest.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Msg", "DeleteAllowlistEntry", data);
    return promise.then((data) => MsgAllowlistResponse.decode(new _m0.Reader(data)));
  }

  StoreCode(request: MsgStoreCode): Promise<MsgStoreCodeResponse> {
    const data = MsgStoreCode.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Msg", "StoreCode", data);
    return promise.then((data) => MsgStoreCodeResponse.decode(new _m0.Reader(data)));
  }

  InstantiateContract(request: MsgInstantiateContract): Promise<MsgInstantiateContractResponse> {
    const data = MsgInstantiateContract.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Msg", "InstantiateContract", data);
    return promise.then((data) => MsgInstantiateContractResponse.decode(new _m0.Reader(data)));
  }

  AddWasmInstantiateAllowlist(request: MsgAddWasmInstantiateAllowlist): Promise<MsgWasmInstantiateAllowlistResponse> {
    const data = MsgAddWasmInstantiateAllowlist.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Msg", "AddWasmInstantiateAllowlist", data);
    return promise.then((data) => MsgWasmInstantiateAllowlistResponse.decode(new _m0.Reader(data)));
  }

  DeleteWasmInstantiateAllowlist(
    request: MsgDeleteWasmInstantiateAllowlist,
  ): Promise<MsgWasmInstantiateAllowlistResponse> {
    const data = MsgDeleteWasmInstantiateAllowlist.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Msg", "DeleteWasmInstantiateAllowlist", data);
    return promise.then((data) => MsgWasmInstantiateAllowlistResponse.decode(new _m0.Reader(data)));
  }

  MigrateContract(request: MsgMigrateContract): Promise<MsgMigrateContractResponse> {
    const data = MsgMigrateContract.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Msg", "MigrateContract", data);
    return promise.then((data) => MsgMigrateContractResponse.decode(new _m0.Reader(data)));
  }

  ExecuteGatewayGovernanceVaa(request: MsgExecuteGatewayGovernanceVaa): Promise<EmptyResponse> {
    const data = MsgExecuteGatewayGovernanceVaa.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Msg", "ExecuteGatewayGovernanceVaa", data);
    return promise.then((data) => EmptyResponse.decode(new _m0.Reader(data)));
  }

  GuardianSetUpdateProposal(request: MsgGuardianSetUpdateProposal): Promise<EmptyResponse> {
    const data = MsgGuardianSetUpdateProposal.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Msg", "GuardianSetUpdateProposal", data);
    return promise.then((data) => EmptyResponse.decode(new _m0.Reader(data)));
  }

  GovernanceWormholeMessageProposal(request: MsgGovernanceWormholeMessageProposal): Promise<EmptyResponse> {
    const data = MsgGovernanceWormholeMessageProposal.encode(request).finish();
    const promise = this.rpc.request("wormchain.wormhole.Msg", "GovernanceWormholeMessageProposal", data);
    return promise.then((data) => EmptyResponse.decode(new _m0.Reader(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}

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
