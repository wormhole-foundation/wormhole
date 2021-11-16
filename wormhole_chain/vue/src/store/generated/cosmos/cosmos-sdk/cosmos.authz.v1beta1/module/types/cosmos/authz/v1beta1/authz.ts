/* eslint-disable */
import { Timestamp } from '../../../google/protobuf/timestamp'
import { Any } from '../../../google/protobuf/any'
import { Writer, Reader } from 'protobufjs/minimal'

export const protobufPackage = 'cosmos.authz.v1beta1'

/**
 * GenericAuthorization gives the grantee unrestricted permissions to execute
 * the provided method on behalf of the granter's account.
 */
export interface GenericAuthorization {
  /** Msg, identified by it's type URL, to grant unrestricted permissions to execute */
  msg: string
}

/**
 * Grant gives permissions to execute
 * the provide method with expiration time.
 */
export interface Grant {
  authorization: Any | undefined
  expiration: Date | undefined
}

const baseGenericAuthorization: object = { msg: '' }

export const GenericAuthorization = {
  encode(message: GenericAuthorization, writer: Writer = Writer.create()): Writer {
    if (message.msg !== '') {
      writer.uint32(10).string(message.msg)
    }
    return writer
  },

  decode(input: Reader | Uint8Array, length?: number): GenericAuthorization {
    const reader = input instanceof Uint8Array ? new Reader(input) : input
    let end = length === undefined ? reader.len : reader.pos + length
    const message = { ...baseGenericAuthorization } as GenericAuthorization
    while (reader.pos < end) {
      const tag = reader.uint32()
      switch (tag >>> 3) {
        case 1:
          message.msg = reader.string()
          break
        default:
          reader.skipType(tag & 7)
          break
      }
    }
    return message
  },

  fromJSON(object: any): GenericAuthorization {
    const message = { ...baseGenericAuthorization } as GenericAuthorization
    if (object.msg !== undefined && object.msg !== null) {
      message.msg = String(object.msg)
    } else {
      message.msg = ''
    }
    return message
  },

  toJSON(message: GenericAuthorization): unknown {
    const obj: any = {}
    message.msg !== undefined && (obj.msg = message.msg)
    return obj
  },

  fromPartial(object: DeepPartial<GenericAuthorization>): GenericAuthorization {
    const message = { ...baseGenericAuthorization } as GenericAuthorization
    if (object.msg !== undefined && object.msg !== null) {
      message.msg = object.msg
    } else {
      message.msg = ''
    }
    return message
  }
}

const baseGrant: object = {}

export const Grant = {
  encode(message: Grant, writer: Writer = Writer.create()): Writer {
    if (message.authorization !== undefined) {
      Any.encode(message.authorization, writer.uint32(10).fork()).ldelim()
    }
    if (message.expiration !== undefined) {
      Timestamp.encode(toTimestamp(message.expiration), writer.uint32(18).fork()).ldelim()
    }
    return writer
  },

  decode(input: Reader | Uint8Array, length?: number): Grant {
    const reader = input instanceof Uint8Array ? new Reader(input) : input
    let end = length === undefined ? reader.len : reader.pos + length
    const message = { ...baseGrant } as Grant
    while (reader.pos < end) {
      const tag = reader.uint32()
      switch (tag >>> 3) {
        case 1:
          message.authorization = Any.decode(reader, reader.uint32())
          break
        case 2:
          message.expiration = fromTimestamp(Timestamp.decode(reader, reader.uint32()))
          break
        default:
          reader.skipType(tag & 7)
          break
      }
    }
    return message
  },

  fromJSON(object: any): Grant {
    const message = { ...baseGrant } as Grant
    if (object.authorization !== undefined && object.authorization !== null) {
      message.authorization = Any.fromJSON(object.authorization)
    } else {
      message.authorization = undefined
    }
    if (object.expiration !== undefined && object.expiration !== null) {
      message.expiration = fromJsonTimestamp(object.expiration)
    } else {
      message.expiration = undefined
    }
    return message
  },

  toJSON(message: Grant): unknown {
    const obj: any = {}
    message.authorization !== undefined && (obj.authorization = message.authorization ? Any.toJSON(message.authorization) : undefined)
    message.expiration !== undefined && (obj.expiration = message.expiration !== undefined ? message.expiration.toISOString() : null)
    return obj
  },

  fromPartial(object: DeepPartial<Grant>): Grant {
    const message = { ...baseGrant } as Grant
    if (object.authorization !== undefined && object.authorization !== null) {
      message.authorization = Any.fromPartial(object.authorization)
    } else {
      message.authorization = undefined
    }
    if (object.expiration !== undefined && object.expiration !== null) {
      message.expiration = object.expiration
    } else {
      message.expiration = undefined
    }
    return message
  }
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

function toTimestamp(date: Date): Timestamp {
  const seconds = date.getTime() / 1_000
  const nanos = (date.getTime() % 1_000) * 1_000_000
  return { seconds, nanos }
}

function fromTimestamp(t: Timestamp): Date {
  let millis = t.seconds * 1_000
  millis += t.nanos / 1_000_000
  return new Date(millis)
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof Date) {
    return o
  } else if (typeof o === 'string') {
    return new Date(o)
  } else {
    return fromTimestamp(Timestamp.fromJSON(o))
  }
}
