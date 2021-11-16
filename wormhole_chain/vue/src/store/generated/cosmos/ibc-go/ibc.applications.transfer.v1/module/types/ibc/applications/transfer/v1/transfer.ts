/* eslint-disable */
import * as Long from 'long'
import { util, configure, Writer, Reader } from 'protobufjs/minimal'

export const protobufPackage = 'ibc.applications.transfer.v1'

/**
 * FungibleTokenPacketData defines a struct for the packet payload
 * See FungibleTokenPacketData spec:
 * https://github.com/cosmos/ics/tree/master/spec/ics-020-fungible-token-transfer#data-structures
 */
export interface FungibleTokenPacketData {
  /** the token denomination to be transferred */
  denom: string
  /** the token amount to be transferred */
  amount: number
  /** the sender address */
  sender: string
  /** the recipient address on the destination chain */
  receiver: string
}

/**
 * DenomTrace contains the base denomination for ICS20 fungible tokens and the
 * source tracing information path.
 */
export interface DenomTrace {
  /**
   * path defines the chain of port/channel identifiers used for tracing the
   * source of the fungible token.
   */
  path: string
  /** base denomination of the relayed fungible token. */
  baseDenom: string
}

/**
 * Params defines the set of IBC transfer parameters.
 * NOTE: To prevent a single token from being transferred, set the
 * TransfersEnabled parameter to true and then set the bank module's SendEnabled
 * parameter for the denomination to false.
 */
export interface Params {
  /**
   * send_enabled enables or disables all cross-chain token transfers from this
   * chain.
   */
  sendEnabled: boolean
  /**
   * receive_enabled enables or disables all cross-chain token transfers to this
   * chain.
   */
  receiveEnabled: boolean
}

const baseFungibleTokenPacketData: object = { denom: '', amount: 0, sender: '', receiver: '' }

export const FungibleTokenPacketData = {
  encode(message: FungibleTokenPacketData, writer: Writer = Writer.create()): Writer {
    if (message.denom !== '') {
      writer.uint32(10).string(message.denom)
    }
    if (message.amount !== 0) {
      writer.uint32(16).uint64(message.amount)
    }
    if (message.sender !== '') {
      writer.uint32(26).string(message.sender)
    }
    if (message.receiver !== '') {
      writer.uint32(34).string(message.receiver)
    }
    return writer
  },

  decode(input: Reader | Uint8Array, length?: number): FungibleTokenPacketData {
    const reader = input instanceof Uint8Array ? new Reader(input) : input
    let end = length === undefined ? reader.len : reader.pos + length
    const message = { ...baseFungibleTokenPacketData } as FungibleTokenPacketData
    while (reader.pos < end) {
      const tag = reader.uint32()
      switch (tag >>> 3) {
        case 1:
          message.denom = reader.string()
          break
        case 2:
          message.amount = longToNumber(reader.uint64() as Long)
          break
        case 3:
          message.sender = reader.string()
          break
        case 4:
          message.receiver = reader.string()
          break
        default:
          reader.skipType(tag & 7)
          break
      }
    }
    return message
  },

  fromJSON(object: any): FungibleTokenPacketData {
    const message = { ...baseFungibleTokenPacketData } as FungibleTokenPacketData
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = String(object.denom)
    } else {
      message.denom = ''
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = Number(object.amount)
    } else {
      message.amount = 0
    }
    if (object.sender !== undefined && object.sender !== null) {
      message.sender = String(object.sender)
    } else {
      message.sender = ''
    }
    if (object.receiver !== undefined && object.receiver !== null) {
      message.receiver = String(object.receiver)
    } else {
      message.receiver = ''
    }
    return message
  },

  toJSON(message: FungibleTokenPacketData): unknown {
    const obj: any = {}
    message.denom !== undefined && (obj.denom = message.denom)
    message.amount !== undefined && (obj.amount = message.amount)
    message.sender !== undefined && (obj.sender = message.sender)
    message.receiver !== undefined && (obj.receiver = message.receiver)
    return obj
  },

  fromPartial(object: DeepPartial<FungibleTokenPacketData>): FungibleTokenPacketData {
    const message = { ...baseFungibleTokenPacketData } as FungibleTokenPacketData
    if (object.denom !== undefined && object.denom !== null) {
      message.denom = object.denom
    } else {
      message.denom = ''
    }
    if (object.amount !== undefined && object.amount !== null) {
      message.amount = object.amount
    } else {
      message.amount = 0
    }
    if (object.sender !== undefined && object.sender !== null) {
      message.sender = object.sender
    } else {
      message.sender = ''
    }
    if (object.receiver !== undefined && object.receiver !== null) {
      message.receiver = object.receiver
    } else {
      message.receiver = ''
    }
    return message
  }
}

const baseDenomTrace: object = { path: '', baseDenom: '' }

export const DenomTrace = {
  encode(message: DenomTrace, writer: Writer = Writer.create()): Writer {
    if (message.path !== '') {
      writer.uint32(10).string(message.path)
    }
    if (message.baseDenom !== '') {
      writer.uint32(18).string(message.baseDenom)
    }
    return writer
  },

  decode(input: Reader | Uint8Array, length?: number): DenomTrace {
    const reader = input instanceof Uint8Array ? new Reader(input) : input
    let end = length === undefined ? reader.len : reader.pos + length
    const message = { ...baseDenomTrace } as DenomTrace
    while (reader.pos < end) {
      const tag = reader.uint32()
      switch (tag >>> 3) {
        case 1:
          message.path = reader.string()
          break
        case 2:
          message.baseDenom = reader.string()
          break
        default:
          reader.skipType(tag & 7)
          break
      }
    }
    return message
  },

  fromJSON(object: any): DenomTrace {
    const message = { ...baseDenomTrace } as DenomTrace
    if (object.path !== undefined && object.path !== null) {
      message.path = String(object.path)
    } else {
      message.path = ''
    }
    if (object.baseDenom !== undefined && object.baseDenom !== null) {
      message.baseDenom = String(object.baseDenom)
    } else {
      message.baseDenom = ''
    }
    return message
  },

  toJSON(message: DenomTrace): unknown {
    const obj: any = {}
    message.path !== undefined && (obj.path = message.path)
    message.baseDenom !== undefined && (obj.baseDenom = message.baseDenom)
    return obj
  },

  fromPartial(object: DeepPartial<DenomTrace>): DenomTrace {
    const message = { ...baseDenomTrace } as DenomTrace
    if (object.path !== undefined && object.path !== null) {
      message.path = object.path
    } else {
      message.path = ''
    }
    if (object.baseDenom !== undefined && object.baseDenom !== null) {
      message.baseDenom = object.baseDenom
    } else {
      message.baseDenom = ''
    }
    return message
  }
}

const baseParams: object = { sendEnabled: false, receiveEnabled: false }

export const Params = {
  encode(message: Params, writer: Writer = Writer.create()): Writer {
    if (message.sendEnabled === true) {
      writer.uint32(8).bool(message.sendEnabled)
    }
    if (message.receiveEnabled === true) {
      writer.uint32(16).bool(message.receiveEnabled)
    }
    return writer
  },

  decode(input: Reader | Uint8Array, length?: number): Params {
    const reader = input instanceof Uint8Array ? new Reader(input) : input
    let end = length === undefined ? reader.len : reader.pos + length
    const message = { ...baseParams } as Params
    while (reader.pos < end) {
      const tag = reader.uint32()
      switch (tag >>> 3) {
        case 1:
          message.sendEnabled = reader.bool()
          break
        case 2:
          message.receiveEnabled = reader.bool()
          break
        default:
          reader.skipType(tag & 7)
          break
      }
    }
    return message
  },

  fromJSON(object: any): Params {
    const message = { ...baseParams } as Params
    if (object.sendEnabled !== undefined && object.sendEnabled !== null) {
      message.sendEnabled = Boolean(object.sendEnabled)
    } else {
      message.sendEnabled = false
    }
    if (object.receiveEnabled !== undefined && object.receiveEnabled !== null) {
      message.receiveEnabled = Boolean(object.receiveEnabled)
    } else {
      message.receiveEnabled = false
    }
    return message
  },

  toJSON(message: Params): unknown {
    const obj: any = {}
    message.sendEnabled !== undefined && (obj.sendEnabled = message.sendEnabled)
    message.receiveEnabled !== undefined && (obj.receiveEnabled = message.receiveEnabled)
    return obj
  },

  fromPartial(object: DeepPartial<Params>): Params {
    const message = { ...baseParams } as Params
    if (object.sendEnabled !== undefined && object.sendEnabled !== null) {
      message.sendEnabled = object.sendEnabled
    } else {
      message.sendEnabled = false
    }
    if (object.receiveEnabled !== undefined && object.receiveEnabled !== null) {
      message.receiveEnabled = object.receiveEnabled
    } else {
      message.receiveEnabled = false
    }
    return message
  }
}

declare var self: any | undefined
declare var window: any | undefined
var globalThis: any = (() => {
  if (typeof globalThis !== 'undefined') return globalThis
  if (typeof self !== 'undefined') return self
  if (typeof window !== 'undefined') return window
  if (typeof global !== 'undefined') return global
  throw 'Unable to locate global object'
})()

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

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error('Value is larger than Number.MAX_SAFE_INTEGER')
  }
  return long.toNumber()
}

if (util.Long !== Long) {
  util.Long = Long as any
  configure()
}
