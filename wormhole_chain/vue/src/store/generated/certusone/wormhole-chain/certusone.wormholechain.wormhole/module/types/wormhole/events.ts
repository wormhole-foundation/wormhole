/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";

export const protobufPackage = "certusone.wormholechain.wormhole";

export interface EventGuardianSetUpdate {
  oldIndex: number;
  newIndex: number;
}

const baseEventGuardianSetUpdate: object = { oldIndex: 0, newIndex: 0 };

export const EventGuardianSetUpdate = {
  encode(
    message: EventGuardianSetUpdate,
    writer: Writer = Writer.create()
  ): Writer {
    if (message.oldIndex !== 0) {
      writer.uint32(8).uint32(message.oldIndex);
    }
    if (message.newIndex !== 0) {
      writer.uint32(16).uint32(message.newIndex);
    }
    return writer;
  },

  decode(input: Reader | Uint8Array, length?: number): EventGuardianSetUpdate {
    const reader = input instanceof Uint8Array ? new Reader(input) : input;
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = { ...baseEventGuardianSetUpdate } as EventGuardianSetUpdate;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.oldIndex = reader.uint32();
          break;
        case 2:
          message.newIndex = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): EventGuardianSetUpdate {
    const message = { ...baseEventGuardianSetUpdate } as EventGuardianSetUpdate;
    if (object.oldIndex !== undefined && object.oldIndex !== null) {
      message.oldIndex = Number(object.oldIndex);
    } else {
      message.oldIndex = 0;
    }
    if (object.newIndex !== undefined && object.newIndex !== null) {
      message.newIndex = Number(object.newIndex);
    } else {
      message.newIndex = 0;
    }
    return message;
  },

  toJSON(message: EventGuardianSetUpdate): unknown {
    const obj: any = {};
    message.oldIndex !== undefined && (obj.oldIndex = message.oldIndex);
    message.newIndex !== undefined && (obj.newIndex = message.newIndex);
    return obj;
  },

  fromPartial(
    object: DeepPartial<EventGuardianSetUpdate>
  ): EventGuardianSetUpdate {
    const message = { ...baseEventGuardianSetUpdate } as EventGuardianSetUpdate;
    if (object.oldIndex !== undefined && object.oldIndex !== null) {
      message.oldIndex = object.oldIndex;
    } else {
      message.oldIndex = 0;
    }
    if (object.newIndex !== undefined && object.newIndex !== null) {
      message.newIndex = object.newIndex;
    } else {
      message.newIndex = 0;
    }
    return message;
  },
};

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
