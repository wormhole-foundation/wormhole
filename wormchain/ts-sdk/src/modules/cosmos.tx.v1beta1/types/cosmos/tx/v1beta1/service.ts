//@ts-nocheck
/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Block } from "../../../tendermint/types/block";
import { BlockID } from "../../../tendermint/types/types";
import { GasInfo, Result, TxResponse } from "../../base/abci/v1beta1/abci";
import { PageRequest, PageResponse } from "../../base/query/v1beta1/pagination";
import { Tx } from "./tx";

export const protobufPackage = "cosmos.tx.v1beta1";

/** OrderBy defines the sorting order */
export enum OrderBy {
  /** ORDER_BY_UNSPECIFIED - ORDER_BY_UNSPECIFIED specifies an unknown sorting order. OrderBy defaults to ASC in this case. */
  ORDER_BY_UNSPECIFIED = 0,
  /** ORDER_BY_ASC - ORDER_BY_ASC defines ascending order */
  ORDER_BY_ASC = 1,
  /** ORDER_BY_DESC - ORDER_BY_DESC defines descending order */
  ORDER_BY_DESC = 2,
  UNRECOGNIZED = -1,
}

export function orderByFromJSON(object: any): OrderBy {
  switch (object) {
    case 0:
    case "ORDER_BY_UNSPECIFIED":
      return OrderBy.ORDER_BY_UNSPECIFIED;
    case 1:
    case "ORDER_BY_ASC":
      return OrderBy.ORDER_BY_ASC;
    case 2:
    case "ORDER_BY_DESC":
      return OrderBy.ORDER_BY_DESC;
    case -1:
    case "UNRECOGNIZED":
    default:
      return OrderBy.UNRECOGNIZED;
  }
}

export function orderByToJSON(object: OrderBy): string {
  switch (object) {
    case OrderBy.ORDER_BY_UNSPECIFIED:
      return "ORDER_BY_UNSPECIFIED";
    case OrderBy.ORDER_BY_ASC:
      return "ORDER_BY_ASC";
    case OrderBy.ORDER_BY_DESC:
      return "ORDER_BY_DESC";
    case OrderBy.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** BroadcastMode specifies the broadcast mode for the TxService.Broadcast RPC method. */
export enum BroadcastMode {
  /** BROADCAST_MODE_UNSPECIFIED - zero-value for mode ordering */
  BROADCAST_MODE_UNSPECIFIED = 0,
  /**
   * BROADCAST_MODE_BLOCK - DEPRECATED: use BROADCAST_MODE_SYNC instead,
   * BROADCAST_MODE_BLOCK is not supported by the SDK from v0.47.x onwards.
   *
   * @deprecated
   */
  BROADCAST_MODE_BLOCK = 1,
  /**
   * BROADCAST_MODE_SYNC - BROADCAST_MODE_SYNC defines a tx broadcasting mode where the client waits for
   * a CheckTx execution response only.
   */
  BROADCAST_MODE_SYNC = 2,
  /**
   * BROADCAST_MODE_ASYNC - BROADCAST_MODE_ASYNC defines a tx broadcasting mode where the client returns
   * immediately.
   */
  BROADCAST_MODE_ASYNC = 3,
  UNRECOGNIZED = -1,
}

export function broadcastModeFromJSON(object: any): BroadcastMode {
  switch (object) {
    case 0:
    case "BROADCAST_MODE_UNSPECIFIED":
      return BroadcastMode.BROADCAST_MODE_UNSPECIFIED;
    case 1:
    case "BROADCAST_MODE_BLOCK":
      return BroadcastMode.BROADCAST_MODE_BLOCK;
    case 2:
    case "BROADCAST_MODE_SYNC":
      return BroadcastMode.BROADCAST_MODE_SYNC;
    case 3:
    case "BROADCAST_MODE_ASYNC":
      return BroadcastMode.BROADCAST_MODE_ASYNC;
    case -1:
    case "UNRECOGNIZED":
    default:
      return BroadcastMode.UNRECOGNIZED;
  }
}

export function broadcastModeToJSON(object: BroadcastMode): string {
  switch (object) {
    case BroadcastMode.BROADCAST_MODE_UNSPECIFIED:
      return "BROADCAST_MODE_UNSPECIFIED";
    case BroadcastMode.BROADCAST_MODE_BLOCK:
      return "BROADCAST_MODE_BLOCK";
    case BroadcastMode.BROADCAST_MODE_SYNC:
      return "BROADCAST_MODE_SYNC";
    case BroadcastMode.BROADCAST_MODE_ASYNC:
      return "BROADCAST_MODE_ASYNC";
    case BroadcastMode.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/**
 * GetTxsEventRequest is the request type for the Service.TxsByEvents
 * RPC method.
 */
export interface GetTxsEventRequest {
  /** events is the list of transaction event type. */
  events: string[];
  /**
   * pagination defines a pagination for the request.
   * Deprecated post v0.46.x: use page and limit instead.
   *
   * @deprecated
   */
  pagination: PageRequest | undefined;
  orderBy: OrderBy;
  /** page is the page number to query, starts at 1. If not provided, will default to first page. */
  page: number;
  /**
   * limit is the total number of results to be returned in the result page.
   * If left empty it will default to a value to be set by each app.
   */
  limit: number;
}

/**
 * GetTxsEventResponse is the response type for the Service.TxsByEvents
 * RPC method.
 */
export interface GetTxsEventResponse {
  /** txs is the list of queried transactions. */
  txs: Tx[];
  /** tx_responses is the list of queried TxResponses. */
  txResponses: TxResponse[];
  /**
   * pagination defines a pagination for the response.
   * Deprecated post v0.46.x: use total instead.
   *
   * @deprecated
   */
  pagination:
    | PageResponse
    | undefined;
  /** total is total number of results available */
  total: number;
}

/**
 * BroadcastTxRequest is the request type for the Service.BroadcastTxRequest
 * RPC method.
 */
export interface BroadcastTxRequest {
  /** tx_bytes is the raw transaction. */
  txBytes: Uint8Array;
  mode: BroadcastMode;
}

/**
 * BroadcastTxResponse is the response type for the
 * Service.BroadcastTx method.
 */
export interface BroadcastTxResponse {
  /** tx_response is the queried TxResponses. */
  txResponse: TxResponse | undefined;
}

/**
 * SimulateRequest is the request type for the Service.Simulate
 * RPC method.
 */
export interface SimulateRequest {
  /**
   * tx is the transaction to simulate.
   * Deprecated. Send raw tx bytes instead.
   *
   * @deprecated
   */
  tx:
    | Tx
    | undefined;
  /**
   * tx_bytes is the raw transaction.
   *
   * Since: cosmos-sdk 0.43
   */
  txBytes: Uint8Array;
}

/**
 * SimulateResponse is the response type for the
 * Service.SimulateRPC method.
 */
export interface SimulateResponse {
  /** gas_info is the information about gas used in the simulation. */
  gasInfo:
    | GasInfo
    | undefined;
  /** result is the result of the simulation. */
  result: Result | undefined;
}

/**
 * GetTxRequest is the request type for the Service.GetTx
 * RPC method.
 */
export interface GetTxRequest {
  /** hash is the tx hash to query, encoded as a hex string. */
  hash: string;
}

/** GetTxResponse is the response type for the Service.GetTx method. */
export interface GetTxResponse {
  /** tx is the queried transaction. */
  tx:
    | Tx
    | undefined;
  /** tx_response is the queried TxResponses. */
  txResponse: TxResponse | undefined;
}

/**
 * GetBlockWithTxsRequest is the request type for the Service.GetBlockWithTxs
 * RPC method.
 *
 * Since: cosmos-sdk 0.45.2
 */
export interface GetBlockWithTxsRequest {
  /** height is the height of the block to query. */
  height: number;
  /** pagination defines a pagination for the request. */
  pagination: PageRequest | undefined;
}

/**
 * GetBlockWithTxsResponse is the response type for the Service.GetBlockWithTxs method.
 *
 * Since: cosmos-sdk 0.45.2
 */
export interface GetBlockWithTxsResponse {
  /** txs are the transactions in the block. */
  txs: Tx[];
  blockId: BlockID | undefined;
  block:
    | Block
    | undefined;
  /** pagination defines a pagination for the response. */
  pagination: PageResponse | undefined;
}

/**
 * TxDecodeRequest is the request type for the Service.TxDecode
 * RPC method.
 *
 * Since: cosmos-sdk 0.47
 */
export interface TxDecodeRequest {
  /** tx_bytes is the raw transaction. */
  txBytes: Uint8Array;
}

/**
 * TxDecodeResponse is the response type for the
 * Service.TxDecode method.
 *
 * Since: cosmos-sdk 0.47
 */
export interface TxDecodeResponse {
  /** tx is the decoded transaction. */
  tx: Tx | undefined;
}

/**
 * TxEncodeRequest is the request type for the Service.TxEncode
 * RPC method.
 *
 * Since: cosmos-sdk 0.47
 */
export interface TxEncodeRequest {
  /** tx is the transaction to encode. */
  tx: Tx | undefined;
}

/**
 * TxEncodeResponse is the response type for the
 * Service.TxEncode method.
 *
 * Since: cosmos-sdk 0.47
 */
export interface TxEncodeResponse {
  /** tx_bytes is the encoded transaction bytes. */
  txBytes: Uint8Array;
}

/**
 * TxEncodeAminoRequest is the request type for the Service.TxEncodeAmino
 * RPC method.
 *
 * Since: cosmos-sdk 0.47
 */
export interface TxEncodeAminoRequest {
  aminoJson: string;
}

/**
 * TxEncodeAminoResponse is the response type for the Service.TxEncodeAmino
 * RPC method.
 *
 * Since: cosmos-sdk 0.47
 */
export interface TxEncodeAminoResponse {
  aminoBinary: Uint8Array;
}

/**
 * TxDecodeAminoRequest is the request type for the Service.TxDecodeAmino
 * RPC method.
 *
 * Since: cosmos-sdk 0.47
 */
export interface TxDecodeAminoRequest {
  aminoBinary: Uint8Array;
}

/**
 * TxDecodeAminoResponse is the response type for the Service.TxDecodeAmino
 * RPC method.
 *
 * Since: cosmos-sdk 0.47
 */
export interface TxDecodeAminoResponse {
  aminoJson: string;
}

function createBaseGetTxsEventRequest(): GetTxsEventRequest {
  return { events: [], pagination: undefined, orderBy: 0, page: 0, limit: 0 };
}

export const GetTxsEventRequest = {
  encode(message: GetTxsEventRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.events) {
      writer.uint32(10).string(v!);
    }
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    if (message.orderBy !== 0) {
      writer.uint32(24).int32(message.orderBy);
    }
    if (message.page !== 0) {
      writer.uint32(32).uint64(message.page);
    }
    if (message.limit !== 0) {
      writer.uint32(40).uint64(message.limit);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetTxsEventRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetTxsEventRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.events.push(reader.string());
          break;
        case 2:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        case 3:
          message.orderBy = reader.int32() as any;
          break;
        case 4:
          message.page = longToNumber(reader.uint64() as Long);
          break;
        case 5:
          message.limit = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GetTxsEventRequest {
    return {
      events: Array.isArray(object?.events) ? object.events.map((e: any) => String(e)) : [],
      pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined,
      orderBy: isSet(object.orderBy) ? orderByFromJSON(object.orderBy) : 0,
      page: isSet(object.page) ? Number(object.page) : 0,
      limit: isSet(object.limit) ? Number(object.limit) : 0,
    };
  },

  toJSON(message: GetTxsEventRequest): unknown {
    const obj: any = {};
    if (message.events) {
      obj.events = message.events.map((e) => e);
    } else {
      obj.events = [];
    }
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    message.orderBy !== undefined && (obj.orderBy = orderByToJSON(message.orderBy));
    message.page !== undefined && (obj.page = Math.round(message.page));
    message.limit !== undefined && (obj.limit = Math.round(message.limit));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GetTxsEventRequest>, I>>(object: I): GetTxsEventRequest {
    const message = createBaseGetTxsEventRequest();
    message.events = object.events?.map((e) => e) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    message.orderBy = object.orderBy ?? 0;
    message.page = object.page ?? 0;
    message.limit = object.limit ?? 0;
    return message;
  },
};

function createBaseGetTxsEventResponse(): GetTxsEventResponse {
  return { txs: [], txResponses: [], pagination: undefined, total: 0 };
}

export const GetTxsEventResponse = {
  encode(message: GetTxsEventResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.txs) {
      Tx.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.txResponses) {
      TxResponse.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(26).fork()).ldelim();
    }
    if (message.total !== 0) {
      writer.uint32(32).uint64(message.total);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetTxsEventResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetTxsEventResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.txs.push(Tx.decode(reader, reader.uint32()));
          break;
        case 2:
          message.txResponses.push(TxResponse.decode(reader, reader.uint32()));
          break;
        case 3:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        case 4:
          message.total = longToNumber(reader.uint64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GetTxsEventResponse {
    return {
      txs: Array.isArray(object?.txs) ? object.txs.map((e: any) => Tx.fromJSON(e)) : [],
      txResponses: Array.isArray(object?.txResponses) ? object.txResponses.map((e: any) => TxResponse.fromJSON(e)) : [],
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
      total: isSet(object.total) ? Number(object.total) : 0,
    };
  },

  toJSON(message: GetTxsEventResponse): unknown {
    const obj: any = {};
    if (message.txs) {
      obj.txs = message.txs.map((e) => e ? Tx.toJSON(e) : undefined);
    } else {
      obj.txs = [];
    }
    if (message.txResponses) {
      obj.txResponses = message.txResponses.map((e) => e ? TxResponse.toJSON(e) : undefined);
    } else {
      obj.txResponses = [];
    }
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    message.total !== undefined && (obj.total = Math.round(message.total));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GetTxsEventResponse>, I>>(object: I): GetTxsEventResponse {
    const message = createBaseGetTxsEventResponse();
    message.txs = object.txs?.map((e) => Tx.fromPartial(e)) || [];
    message.txResponses = object.txResponses?.map((e) => TxResponse.fromPartial(e)) || [];
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    message.total = object.total ?? 0;
    return message;
  },
};

function createBaseBroadcastTxRequest(): BroadcastTxRequest {
  return { txBytes: new Uint8Array(), mode: 0 };
}

export const BroadcastTxRequest = {
  encode(message: BroadcastTxRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.txBytes.length !== 0) {
      writer.uint32(10).bytes(message.txBytes);
    }
    if (message.mode !== 0) {
      writer.uint32(16).int32(message.mode);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BroadcastTxRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBroadcastTxRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.txBytes = reader.bytes();
          break;
        case 2:
          message.mode = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): BroadcastTxRequest {
    return {
      txBytes: isSet(object.txBytes) ? bytesFromBase64(object.txBytes) : new Uint8Array(),
      mode: isSet(object.mode) ? broadcastModeFromJSON(object.mode) : 0,
    };
  },

  toJSON(message: BroadcastTxRequest): unknown {
    const obj: any = {};
    message.txBytes !== undefined
      && (obj.txBytes = base64FromBytes(message.txBytes !== undefined ? message.txBytes : new Uint8Array()));
    message.mode !== undefined && (obj.mode = broadcastModeToJSON(message.mode));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<BroadcastTxRequest>, I>>(object: I): BroadcastTxRequest {
    const message = createBaseBroadcastTxRequest();
    message.txBytes = object.txBytes ?? new Uint8Array();
    message.mode = object.mode ?? 0;
    return message;
  },
};

function createBaseBroadcastTxResponse(): BroadcastTxResponse {
  return { txResponse: undefined };
}

export const BroadcastTxResponse = {
  encode(message: BroadcastTxResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.txResponse !== undefined) {
      TxResponse.encode(message.txResponse, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BroadcastTxResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBroadcastTxResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.txResponse = TxResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): BroadcastTxResponse {
    return { txResponse: isSet(object.txResponse) ? TxResponse.fromJSON(object.txResponse) : undefined };
  },

  toJSON(message: BroadcastTxResponse): unknown {
    const obj: any = {};
    message.txResponse !== undefined
      && (obj.txResponse = message.txResponse ? TxResponse.toJSON(message.txResponse) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<BroadcastTxResponse>, I>>(object: I): BroadcastTxResponse {
    const message = createBaseBroadcastTxResponse();
    message.txResponse = (object.txResponse !== undefined && object.txResponse !== null)
      ? TxResponse.fromPartial(object.txResponse)
      : undefined;
    return message;
  },
};

function createBaseSimulateRequest(): SimulateRequest {
  return { tx: undefined, txBytes: new Uint8Array() };
}

export const SimulateRequest = {
  encode(message: SimulateRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.tx !== undefined) {
      Tx.encode(message.tx, writer.uint32(10).fork()).ldelim();
    }
    if (message.txBytes.length !== 0) {
      writer.uint32(18).bytes(message.txBytes);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SimulateRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSimulateRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tx = Tx.decode(reader, reader.uint32());
          break;
        case 2:
          message.txBytes = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SimulateRequest {
    return {
      tx: isSet(object.tx) ? Tx.fromJSON(object.tx) : undefined,
      txBytes: isSet(object.txBytes) ? bytesFromBase64(object.txBytes) : new Uint8Array(),
    };
  },

  toJSON(message: SimulateRequest): unknown {
    const obj: any = {};
    message.tx !== undefined && (obj.tx = message.tx ? Tx.toJSON(message.tx) : undefined);
    message.txBytes !== undefined
      && (obj.txBytes = base64FromBytes(message.txBytes !== undefined ? message.txBytes : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<SimulateRequest>, I>>(object: I): SimulateRequest {
    const message = createBaseSimulateRequest();
    message.tx = (object.tx !== undefined && object.tx !== null) ? Tx.fromPartial(object.tx) : undefined;
    message.txBytes = object.txBytes ?? new Uint8Array();
    return message;
  },
};

function createBaseSimulateResponse(): SimulateResponse {
  return { gasInfo: undefined, result: undefined };
}

export const SimulateResponse = {
  encode(message: SimulateResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.gasInfo !== undefined) {
      GasInfo.encode(message.gasInfo, writer.uint32(10).fork()).ldelim();
    }
    if (message.result !== undefined) {
      Result.encode(message.result, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SimulateResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSimulateResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.gasInfo = GasInfo.decode(reader, reader.uint32());
          break;
        case 2:
          message.result = Result.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SimulateResponse {
    return {
      gasInfo: isSet(object.gasInfo) ? GasInfo.fromJSON(object.gasInfo) : undefined,
      result: isSet(object.result) ? Result.fromJSON(object.result) : undefined,
    };
  },

  toJSON(message: SimulateResponse): unknown {
    const obj: any = {};
    message.gasInfo !== undefined && (obj.gasInfo = message.gasInfo ? GasInfo.toJSON(message.gasInfo) : undefined);
    message.result !== undefined && (obj.result = message.result ? Result.toJSON(message.result) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<SimulateResponse>, I>>(object: I): SimulateResponse {
    const message = createBaseSimulateResponse();
    message.gasInfo = (object.gasInfo !== undefined && object.gasInfo !== null)
      ? GasInfo.fromPartial(object.gasInfo)
      : undefined;
    message.result = (object.result !== undefined && object.result !== null)
      ? Result.fromPartial(object.result)
      : undefined;
    return message;
  },
};

function createBaseGetTxRequest(): GetTxRequest {
  return { hash: "" };
}

export const GetTxRequest = {
  encode(message: GetTxRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.hash !== "") {
      writer.uint32(10).string(message.hash);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetTxRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetTxRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.hash = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GetTxRequest {
    return { hash: isSet(object.hash) ? String(object.hash) : "" };
  },

  toJSON(message: GetTxRequest): unknown {
    const obj: any = {};
    message.hash !== undefined && (obj.hash = message.hash);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GetTxRequest>, I>>(object: I): GetTxRequest {
    const message = createBaseGetTxRequest();
    message.hash = object.hash ?? "";
    return message;
  },
};

function createBaseGetTxResponse(): GetTxResponse {
  return { tx: undefined, txResponse: undefined };
}

export const GetTxResponse = {
  encode(message: GetTxResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.tx !== undefined) {
      Tx.encode(message.tx, writer.uint32(10).fork()).ldelim();
    }
    if (message.txResponse !== undefined) {
      TxResponse.encode(message.txResponse, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetTxResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetTxResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tx = Tx.decode(reader, reader.uint32());
          break;
        case 2:
          message.txResponse = TxResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GetTxResponse {
    return {
      tx: isSet(object.tx) ? Tx.fromJSON(object.tx) : undefined,
      txResponse: isSet(object.txResponse) ? TxResponse.fromJSON(object.txResponse) : undefined,
    };
  },

  toJSON(message: GetTxResponse): unknown {
    const obj: any = {};
    message.tx !== undefined && (obj.tx = message.tx ? Tx.toJSON(message.tx) : undefined);
    message.txResponse !== undefined
      && (obj.txResponse = message.txResponse ? TxResponse.toJSON(message.txResponse) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GetTxResponse>, I>>(object: I): GetTxResponse {
    const message = createBaseGetTxResponse();
    message.tx = (object.tx !== undefined && object.tx !== null) ? Tx.fromPartial(object.tx) : undefined;
    message.txResponse = (object.txResponse !== undefined && object.txResponse !== null)
      ? TxResponse.fromPartial(object.txResponse)
      : undefined;
    return message;
  },
};

function createBaseGetBlockWithTxsRequest(): GetBlockWithTxsRequest {
  return { height: 0, pagination: undefined };
}

export const GetBlockWithTxsRequest = {
  encode(message: GetBlockWithTxsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.height !== 0) {
      writer.uint32(8).int64(message.height);
    }
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetBlockWithTxsRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetBlockWithTxsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.height = longToNumber(reader.int64() as Long);
          break;
        case 2:
          message.pagination = PageRequest.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GetBlockWithTxsRequest {
    return {
      height: isSet(object.height) ? Number(object.height) : 0,
      pagination: isSet(object.pagination) ? PageRequest.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: GetBlockWithTxsRequest): unknown {
    const obj: any = {};
    message.height !== undefined && (obj.height = Math.round(message.height));
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GetBlockWithTxsRequest>, I>>(object: I): GetBlockWithTxsRequest {
    const message = createBaseGetBlockWithTxsRequest();
    message.height = object.height ?? 0;
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageRequest.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseGetBlockWithTxsResponse(): GetBlockWithTxsResponse {
  return { txs: [], blockId: undefined, block: undefined, pagination: undefined };
}

export const GetBlockWithTxsResponse = {
  encode(message: GetBlockWithTxsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.txs) {
      Tx.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.blockId !== undefined) {
      BlockID.encode(message.blockId, writer.uint32(18).fork()).ldelim();
    }
    if (message.block !== undefined) {
      Block.encode(message.block, writer.uint32(26).fork()).ldelim();
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetBlockWithTxsResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetBlockWithTxsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.txs.push(Tx.decode(reader, reader.uint32()));
          break;
        case 2:
          message.blockId = BlockID.decode(reader, reader.uint32());
          break;
        case 3:
          message.block = Block.decode(reader, reader.uint32());
          break;
        case 4:
          message.pagination = PageResponse.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GetBlockWithTxsResponse {
    return {
      txs: Array.isArray(object?.txs) ? object.txs.map((e: any) => Tx.fromJSON(e)) : [],
      blockId: isSet(object.blockId) ? BlockID.fromJSON(object.blockId) : undefined,
      block: isSet(object.block) ? Block.fromJSON(object.block) : undefined,
      pagination: isSet(object.pagination) ? PageResponse.fromJSON(object.pagination) : undefined,
    };
  },

  toJSON(message: GetBlockWithTxsResponse): unknown {
    const obj: any = {};
    if (message.txs) {
      obj.txs = message.txs.map((e) => e ? Tx.toJSON(e) : undefined);
    } else {
      obj.txs = [];
    }
    message.blockId !== undefined && (obj.blockId = message.blockId ? BlockID.toJSON(message.blockId) : undefined);
    message.block !== undefined && (obj.block = message.block ? Block.toJSON(message.block) : undefined);
    message.pagination !== undefined
      && (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GetBlockWithTxsResponse>, I>>(object: I): GetBlockWithTxsResponse {
    const message = createBaseGetBlockWithTxsResponse();
    message.txs = object.txs?.map((e) => Tx.fromPartial(e)) || [];
    message.blockId = (object.blockId !== undefined && object.blockId !== null)
      ? BlockID.fromPartial(object.blockId)
      : undefined;
    message.block = (object.block !== undefined && object.block !== null) ? Block.fromPartial(object.block) : undefined;
    message.pagination = (object.pagination !== undefined && object.pagination !== null)
      ? PageResponse.fromPartial(object.pagination)
      : undefined;
    return message;
  },
};

function createBaseTxDecodeRequest(): TxDecodeRequest {
  return { txBytes: new Uint8Array() };
}

export const TxDecodeRequest = {
  encode(message: TxDecodeRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.txBytes.length !== 0) {
      writer.uint32(10).bytes(message.txBytes);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TxDecodeRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTxDecodeRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.txBytes = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TxDecodeRequest {
    return { txBytes: isSet(object.txBytes) ? bytesFromBase64(object.txBytes) : new Uint8Array() };
  },

  toJSON(message: TxDecodeRequest): unknown {
    const obj: any = {};
    message.txBytes !== undefined
      && (obj.txBytes = base64FromBytes(message.txBytes !== undefined ? message.txBytes : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TxDecodeRequest>, I>>(object: I): TxDecodeRequest {
    const message = createBaseTxDecodeRequest();
    message.txBytes = object.txBytes ?? new Uint8Array();
    return message;
  },
};

function createBaseTxDecodeResponse(): TxDecodeResponse {
  return { tx: undefined };
}

export const TxDecodeResponse = {
  encode(message: TxDecodeResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.tx !== undefined) {
      Tx.encode(message.tx, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TxDecodeResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTxDecodeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tx = Tx.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TxDecodeResponse {
    return { tx: isSet(object.tx) ? Tx.fromJSON(object.tx) : undefined };
  },

  toJSON(message: TxDecodeResponse): unknown {
    const obj: any = {};
    message.tx !== undefined && (obj.tx = message.tx ? Tx.toJSON(message.tx) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TxDecodeResponse>, I>>(object: I): TxDecodeResponse {
    const message = createBaseTxDecodeResponse();
    message.tx = (object.tx !== undefined && object.tx !== null) ? Tx.fromPartial(object.tx) : undefined;
    return message;
  },
};

function createBaseTxEncodeRequest(): TxEncodeRequest {
  return { tx: undefined };
}

export const TxEncodeRequest = {
  encode(message: TxEncodeRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.tx !== undefined) {
      Tx.encode(message.tx, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TxEncodeRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTxEncodeRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tx = Tx.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TxEncodeRequest {
    return { tx: isSet(object.tx) ? Tx.fromJSON(object.tx) : undefined };
  },

  toJSON(message: TxEncodeRequest): unknown {
    const obj: any = {};
    message.tx !== undefined && (obj.tx = message.tx ? Tx.toJSON(message.tx) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TxEncodeRequest>, I>>(object: I): TxEncodeRequest {
    const message = createBaseTxEncodeRequest();
    message.tx = (object.tx !== undefined && object.tx !== null) ? Tx.fromPartial(object.tx) : undefined;
    return message;
  },
};

function createBaseTxEncodeResponse(): TxEncodeResponse {
  return { txBytes: new Uint8Array() };
}

export const TxEncodeResponse = {
  encode(message: TxEncodeResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.txBytes.length !== 0) {
      writer.uint32(10).bytes(message.txBytes);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TxEncodeResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTxEncodeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.txBytes = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TxEncodeResponse {
    return { txBytes: isSet(object.txBytes) ? bytesFromBase64(object.txBytes) : new Uint8Array() };
  },

  toJSON(message: TxEncodeResponse): unknown {
    const obj: any = {};
    message.txBytes !== undefined
      && (obj.txBytes = base64FromBytes(message.txBytes !== undefined ? message.txBytes : new Uint8Array()));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TxEncodeResponse>, I>>(object: I): TxEncodeResponse {
    const message = createBaseTxEncodeResponse();
    message.txBytes = object.txBytes ?? new Uint8Array();
    return message;
  },
};

function createBaseTxEncodeAminoRequest(): TxEncodeAminoRequest {
  return { aminoJson: "" };
}

export const TxEncodeAminoRequest = {
  encode(message: TxEncodeAminoRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.aminoJson !== "") {
      writer.uint32(10).string(message.aminoJson);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TxEncodeAminoRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTxEncodeAminoRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.aminoJson = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TxEncodeAminoRequest {
    return { aminoJson: isSet(object.aminoJson) ? String(object.aminoJson) : "" };
  },

  toJSON(message: TxEncodeAminoRequest): unknown {
    const obj: any = {};
    message.aminoJson !== undefined && (obj.aminoJson = message.aminoJson);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TxEncodeAminoRequest>, I>>(object: I): TxEncodeAminoRequest {
    const message = createBaseTxEncodeAminoRequest();
    message.aminoJson = object.aminoJson ?? "";
    return message;
  },
};

function createBaseTxEncodeAminoResponse(): TxEncodeAminoResponse {
  return { aminoBinary: new Uint8Array() };
}

export const TxEncodeAminoResponse = {
  encode(message: TxEncodeAminoResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.aminoBinary.length !== 0) {
      writer.uint32(10).bytes(message.aminoBinary);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TxEncodeAminoResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTxEncodeAminoResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.aminoBinary = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TxEncodeAminoResponse {
    return { aminoBinary: isSet(object.aminoBinary) ? bytesFromBase64(object.aminoBinary) : new Uint8Array() };
  },

  toJSON(message: TxEncodeAminoResponse): unknown {
    const obj: any = {};
    message.aminoBinary !== undefined
      && (obj.aminoBinary = base64FromBytes(
        message.aminoBinary !== undefined ? message.aminoBinary : new Uint8Array(),
      ));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TxEncodeAminoResponse>, I>>(object: I): TxEncodeAminoResponse {
    const message = createBaseTxEncodeAminoResponse();
    message.aminoBinary = object.aminoBinary ?? new Uint8Array();
    return message;
  },
};

function createBaseTxDecodeAminoRequest(): TxDecodeAminoRequest {
  return { aminoBinary: new Uint8Array() };
}

export const TxDecodeAminoRequest = {
  encode(message: TxDecodeAminoRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.aminoBinary.length !== 0) {
      writer.uint32(10).bytes(message.aminoBinary);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TxDecodeAminoRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTxDecodeAminoRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.aminoBinary = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TxDecodeAminoRequest {
    return { aminoBinary: isSet(object.aminoBinary) ? bytesFromBase64(object.aminoBinary) : new Uint8Array() };
  },

  toJSON(message: TxDecodeAminoRequest): unknown {
    const obj: any = {};
    message.aminoBinary !== undefined
      && (obj.aminoBinary = base64FromBytes(
        message.aminoBinary !== undefined ? message.aminoBinary : new Uint8Array(),
      ));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TxDecodeAminoRequest>, I>>(object: I): TxDecodeAminoRequest {
    const message = createBaseTxDecodeAminoRequest();
    message.aminoBinary = object.aminoBinary ?? new Uint8Array();
    return message;
  },
};

function createBaseTxDecodeAminoResponse(): TxDecodeAminoResponse {
  return { aminoJson: "" };
}

export const TxDecodeAminoResponse = {
  encode(message: TxDecodeAminoResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.aminoJson !== "") {
      writer.uint32(10).string(message.aminoJson);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TxDecodeAminoResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTxDecodeAminoResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.aminoJson = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TxDecodeAminoResponse {
    return { aminoJson: isSet(object.aminoJson) ? String(object.aminoJson) : "" };
  },

  toJSON(message: TxDecodeAminoResponse): unknown {
    const obj: any = {};
    message.aminoJson !== undefined && (obj.aminoJson = message.aminoJson);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TxDecodeAminoResponse>, I>>(object: I): TxDecodeAminoResponse {
    const message = createBaseTxDecodeAminoResponse();
    message.aminoJson = object.aminoJson ?? "";
    return message;
  },
};

/** Service defines a gRPC service for interacting with transactions. */
export interface Service {
  /** Simulate simulates executing a transaction for estimating gas usage. */
  Simulate(request: SimulateRequest): Promise<SimulateResponse>;
  /** GetTx fetches a tx by hash. */
  GetTx(request: GetTxRequest): Promise<GetTxResponse>;
  /** BroadcastTx broadcast transaction. */
  BroadcastTx(request: BroadcastTxRequest): Promise<BroadcastTxResponse>;
  /** GetTxsEvent fetches txs by event. */
  GetTxsEvent(request: GetTxsEventRequest): Promise<GetTxsEventResponse>;
  /**
   * GetBlockWithTxs fetches a block with decoded txs.
   *
   * Since: cosmos-sdk 0.45.2
   */
  GetBlockWithTxs(request: GetBlockWithTxsRequest): Promise<GetBlockWithTxsResponse>;
  /**
   * TxDecode decodes the transaction.
   *
   * Since: cosmos-sdk 0.47
   */
  TxDecode(request: TxDecodeRequest): Promise<TxDecodeResponse>;
  /**
   * TxEncode encodes the transaction.
   *
   * Since: cosmos-sdk 0.47
   */
  TxEncode(request: TxEncodeRequest): Promise<TxEncodeResponse>;
  /**
   * TxEncodeAmino encodes an Amino transaction from JSON to encoded bytes.
   *
   * Since: cosmos-sdk 0.47
   */
  TxEncodeAmino(request: TxEncodeAminoRequest): Promise<TxEncodeAminoResponse>;
  /**
   * TxDecodeAmino decodes an Amino transaction from encoded bytes to JSON.
   *
   * Since: cosmos-sdk 0.47
   */
  TxDecodeAmino(request: TxDecodeAminoRequest): Promise<TxDecodeAminoResponse>;
}

export class ServiceClientImpl implements Service {
  private readonly rpc: Rpc;
  constructor(rpc: Rpc) {
    this.rpc = rpc;
    this.Simulate = this.Simulate.bind(this);
    this.GetTx = this.GetTx.bind(this);
    this.BroadcastTx = this.BroadcastTx.bind(this);
    this.GetTxsEvent = this.GetTxsEvent.bind(this);
    this.GetBlockWithTxs = this.GetBlockWithTxs.bind(this);
    this.TxDecode = this.TxDecode.bind(this);
    this.TxEncode = this.TxEncode.bind(this);
    this.TxEncodeAmino = this.TxEncodeAmino.bind(this);
    this.TxDecodeAmino = this.TxDecodeAmino.bind(this);
  }
  Simulate(request: SimulateRequest): Promise<SimulateResponse> {
    const data = SimulateRequest.encode(request).finish();
    const promise = this.rpc.request("cosmos.tx.v1beta1.Service", "Simulate", data);
    return promise.then((data) => SimulateResponse.decode(new _m0.Reader(data)));
  }

  GetTx(request: GetTxRequest): Promise<GetTxResponse> {
    const data = GetTxRequest.encode(request).finish();
    const promise = this.rpc.request("cosmos.tx.v1beta1.Service", "GetTx", data);
    return promise.then((data) => GetTxResponse.decode(new _m0.Reader(data)));
  }

  BroadcastTx(request: BroadcastTxRequest): Promise<BroadcastTxResponse> {
    const data = BroadcastTxRequest.encode(request).finish();
    const promise = this.rpc.request("cosmos.tx.v1beta1.Service", "BroadcastTx", data);
    return promise.then((data) => BroadcastTxResponse.decode(new _m0.Reader(data)));
  }

  GetTxsEvent(request: GetTxsEventRequest): Promise<GetTxsEventResponse> {
    const data = GetTxsEventRequest.encode(request).finish();
    const promise = this.rpc.request("cosmos.tx.v1beta1.Service", "GetTxsEvent", data);
    return promise.then((data) => GetTxsEventResponse.decode(new _m0.Reader(data)));
  }

  GetBlockWithTxs(request: GetBlockWithTxsRequest): Promise<GetBlockWithTxsResponse> {
    const data = GetBlockWithTxsRequest.encode(request).finish();
    const promise = this.rpc.request("cosmos.tx.v1beta1.Service", "GetBlockWithTxs", data);
    return promise.then((data) => GetBlockWithTxsResponse.decode(new _m0.Reader(data)));
  }

  TxDecode(request: TxDecodeRequest): Promise<TxDecodeResponse> {
    const data = TxDecodeRequest.encode(request).finish();
    const promise = this.rpc.request("cosmos.tx.v1beta1.Service", "TxDecode", data);
    return promise.then((data) => TxDecodeResponse.decode(new _m0.Reader(data)));
  }

  TxEncode(request: TxEncodeRequest): Promise<TxEncodeResponse> {
    const data = TxEncodeRequest.encode(request).finish();
    const promise = this.rpc.request("cosmos.tx.v1beta1.Service", "TxEncode", data);
    return promise.then((data) => TxEncodeResponse.decode(new _m0.Reader(data)));
  }

  TxEncodeAmino(request: TxEncodeAminoRequest): Promise<TxEncodeAminoResponse> {
    const data = TxEncodeAminoRequest.encode(request).finish();
    const promise = this.rpc.request("cosmos.tx.v1beta1.Service", "TxEncodeAmino", data);
    return promise.then((data) => TxEncodeAminoResponse.decode(new _m0.Reader(data)));
  }

  TxDecodeAmino(request: TxDecodeAminoRequest): Promise<TxDecodeAminoResponse> {
    const data = TxDecodeAminoRequest.encode(request).finish();
    const promise = this.rpc.request("cosmos.tx.v1beta1.Service", "TxDecodeAmino", data);
    return promise.then((data) => TxDecodeAminoResponse.decode(new _m0.Reader(data)));
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
