/* eslint-disable */
import { Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "certusone.wormholechain.wormhole";
const baseEventGuardianSetUpdate = { oldIndex: 0, newIndex: 0 };
export const EventGuardianSetUpdate = {
    encode(message, writer = Writer.create()) {
        if (message.oldIndex !== 0) {
            writer.uint32(8).uint32(message.oldIndex);
        }
        if (message.newIndex !== 0) {
            writer.uint32(16).uint32(message.newIndex);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseEventGuardianSetUpdate };
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
    fromJSON(object) {
        const message = { ...baseEventGuardianSetUpdate };
        if (object.oldIndex !== undefined && object.oldIndex !== null) {
            message.oldIndex = Number(object.oldIndex);
        }
        else {
            message.oldIndex = 0;
        }
        if (object.newIndex !== undefined && object.newIndex !== null) {
            message.newIndex = Number(object.newIndex);
        }
        else {
            message.newIndex = 0;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.oldIndex !== undefined && (obj.oldIndex = message.oldIndex);
        message.newIndex !== undefined && (obj.newIndex = message.newIndex);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseEventGuardianSetUpdate };
        if (object.oldIndex !== undefined && object.oldIndex !== null) {
            message.oldIndex = object.oldIndex;
        }
        else {
            message.oldIndex = 0;
        }
        if (object.newIndex !== undefined && object.newIndex !== null) {
            message.newIndex = object.newIndex;
        }
        else {
            message.newIndex = 0;
        }
        return message;
    },
};
