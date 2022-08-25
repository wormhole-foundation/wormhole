/* eslint-disable */
import { Reader, Writer } from 'protobufjs/minimal'
import { PageRequest, PageResponse } from '../../../cosmos/base/query/v1beta1/pagination'
import { Grant } from '../../../cosmos/authz/v1beta1/authz'

export const protobufPackage = 'cosmos.authz.v1beta1'

/** QueryGrantsRequest is the request type for the Query/Grants RPC method. */
export interface QueryGrantsRequest {
  granter: string
  grantee: string
  /** Optional, msg_type_url, when set, will query only grants matching given msg type. */
  msgTypeUrl: string
  /** pagination defines an pagination for the request. */
  pagination: PageRequest | undefined
}

/** QueryGrantsResponse is the response type for the Query/Authorizations RPC method. */
export interface QueryGrantsResponse {
  /** authorizations is a list of grants granted for grantee by granter. */
  grants: Grant[]
  /** pagination defines an pagination for the response. */
  pagination: PageResponse | undefined
}

const baseQueryGrantsRequest: object = { granter: '', grantee: '', msgTypeUrl: '' }

export const QueryGrantsRequest = {
  encode(message: QueryGrantsRequest, writer: Writer = Writer.create()): Writer {
    if (message.granter !== '') {
      writer.uint32(10).string(message.granter)
    }
    if (message.grantee !== '') {
      writer.uint32(18).string(message.grantee)
    }
    if (message.msgTypeUrl !== '') {
      writer.uint32(26).string(message.msgTypeUrl)
    }
    if (message.pagination !== undefined) {
      PageRequest.encode(message.pagination, writer.uint32(34).fork()).ldelim()
    }
    return writer
  },

  decode(input: Reader | Uint8Array, length?: number): QueryGrantsRequest {
    const reader = input instanceof Uint8Array ? new Reader(input) : input
    let end = length === undefined ? reader.len : reader.pos + length
    const message = { ...baseQueryGrantsRequest } as QueryGrantsRequest
    while (reader.pos < end) {
      const tag = reader.uint32()
      switch (tag >>> 3) {
        case 1:
          message.granter = reader.string()
          break
        case 2:
          message.grantee = reader.string()
          break
        case 3:
          message.msgTypeUrl = reader.string()
          break
        case 4:
          message.pagination = PageRequest.decode(reader, reader.uint32())
          break
        default:
          reader.skipType(tag & 7)
          break
      }
    }
    return message
  },

  fromJSON(object: any): QueryGrantsRequest {
    const message = { ...baseQueryGrantsRequest } as QueryGrantsRequest
    if (object.granter !== undefined && object.granter !== null) {
      message.granter = String(object.granter)
    } else {
      message.granter = ''
    }
    if (object.grantee !== undefined && object.grantee !== null) {
      message.grantee = String(object.grantee)
    } else {
      message.grantee = ''
    }
    if (object.msgTypeUrl !== undefined && object.msgTypeUrl !== null) {
      message.msgTypeUrl = String(object.msgTypeUrl)
    } else {
      message.msgTypeUrl = ''
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromJSON(object.pagination)
    } else {
      message.pagination = undefined
    }
    return message
  },

  toJSON(message: QueryGrantsRequest): unknown {
    const obj: any = {}
    message.granter !== undefined && (obj.granter = message.granter)
    message.grantee !== undefined && (obj.grantee = message.grantee)
    message.msgTypeUrl !== undefined && (obj.msgTypeUrl = message.msgTypeUrl)
    message.pagination !== undefined && (obj.pagination = message.pagination ? PageRequest.toJSON(message.pagination) : undefined)
    return obj
  },

  fromPartial(object: DeepPartial<QueryGrantsRequest>): QueryGrantsRequest {
    const message = { ...baseQueryGrantsRequest } as QueryGrantsRequest
    if (object.granter !== undefined && object.granter !== null) {
      message.granter = object.granter
    } else {
      message.granter = ''
    }
    if (object.grantee !== undefined && object.grantee !== null) {
      message.grantee = object.grantee
    } else {
      message.grantee = ''
    }
    if (object.msgTypeUrl !== undefined && object.msgTypeUrl !== null) {
      message.msgTypeUrl = object.msgTypeUrl
    } else {
      message.msgTypeUrl = ''
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageRequest.fromPartial(object.pagination)
    } else {
      message.pagination = undefined
    }
    return message
  }
}

const baseQueryGrantsResponse: object = {}

export const QueryGrantsResponse = {
  encode(message: QueryGrantsResponse, writer: Writer = Writer.create()): Writer {
    for (const v of message.grants) {
      Grant.encode(v!, writer.uint32(10).fork()).ldelim()
    }
    if (message.pagination !== undefined) {
      PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim()
    }
    return writer
  },

  decode(input: Reader | Uint8Array, length?: number): QueryGrantsResponse {
    const reader = input instanceof Uint8Array ? new Reader(input) : input
    let end = length === undefined ? reader.len : reader.pos + length
    const message = { ...baseQueryGrantsResponse } as QueryGrantsResponse
    message.grants = []
    while (reader.pos < end) {
      const tag = reader.uint32()
      switch (tag >>> 3) {
        case 1:
          message.grants.push(Grant.decode(reader, reader.uint32()))
          break
        case 2:
          message.pagination = PageResponse.decode(reader, reader.uint32())
          break
        default:
          reader.skipType(tag & 7)
          break
      }
    }
    return message
  },

  fromJSON(object: any): QueryGrantsResponse {
    const message = { ...baseQueryGrantsResponse } as QueryGrantsResponse
    message.grants = []
    if (object.grants !== undefined && object.grants !== null) {
      for (const e of object.grants) {
        message.grants.push(Grant.fromJSON(e))
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromJSON(object.pagination)
    } else {
      message.pagination = undefined
    }
    return message
  },

  toJSON(message: QueryGrantsResponse): unknown {
    const obj: any = {}
    if (message.grants) {
      obj.grants = message.grants.map((e) => (e ? Grant.toJSON(e) : undefined))
    } else {
      obj.grants = []
    }
    message.pagination !== undefined && (obj.pagination = message.pagination ? PageResponse.toJSON(message.pagination) : undefined)
    return obj
  },

  fromPartial(object: DeepPartial<QueryGrantsResponse>): QueryGrantsResponse {
    const message = { ...baseQueryGrantsResponse } as QueryGrantsResponse
    message.grants = []
    if (object.grants !== undefined && object.grants !== null) {
      for (const e of object.grants) {
        message.grants.push(Grant.fromPartial(e))
      }
    }
    if (object.pagination !== undefined && object.pagination !== null) {
      message.pagination = PageResponse.fromPartial(object.pagination)
    } else {
      message.pagination = undefined
    }
    return message
  }
}

/** Query defines the gRPC querier service. */
export interface Query {
  /** Returns list of `Authorization`, granted to the grantee by the granter. */
  Grants(request: QueryGrantsRequest): Promise<QueryGrantsResponse>
}

export class QueryClientImpl implements Query {
  private readonly rpc: Rpc
  constructor(rpc: Rpc) {
    this.rpc = rpc
  }
  Grants(request: QueryGrantsRequest): Promise<QueryGrantsResponse> {
    const data = QueryGrantsRequest.encode(request).finish()
    const promise = this.rpc.request('cosmos.authz.v1beta1.Query', 'Grants', data)
    return promise.then((data) => QueryGrantsResponse.decode(new Reader(data)))
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>
}

type Builtin = Date | Function | Uint8Array | string | number | undefined
export type DeepPartial<T> = T extends Builtin
  ? T
  : T extends Array<infer U>
  ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U>
  ? ReadonlyArray<DeepPartial<U>>
  : T extends {}
  ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>
