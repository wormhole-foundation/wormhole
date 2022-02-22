/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "certusone.wormholechain.tokenbridge";
const baseCoinMetaRollbackProtection = {
    index: "",
    lastUpdateSequence: 0,
};
export const CoinMetaRollbackProtection = {
    encode(message, writer = Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        if (message.lastUpdateSequence !== 0) {
            writer.uint32(16).uint64(message.lastUpdateSequence);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseCoinMetaRollbackProtection,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.index = reader.string();
                    break;
                case 2:
                    message.lastUpdateSequence = longToNumber(reader.uint64());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseCoinMetaRollbackProtection,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = String(object.index);
        }
        else {
            message.index = "";
        }
        if (object.lastUpdateSequence !== undefined &&
            object.lastUpdateSequence !== null) {
            message.lastUpdateSequence = Number(object.lastUpdateSequence);
        }
        else {
            message.lastUpdateSequence = 0;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        message.lastUpdateSequence !== undefined &&
            (obj.lastUpdateSequence = message.lastUpdateSequence);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseCoinMetaRollbackProtection,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = "";
        }
        if (object.lastUpdateSequence !== undefined &&
            object.lastUpdateSequence !== null) {
            message.lastUpdateSequence = object.lastUpdateSequence;
        }
        else {
            message.lastUpdateSequence = 0;
        }
        return message;
    },
};
var globalThis = (() => {
    if (typeof globalThis !== "undefined")
        return globalThis;
    if (typeof self !== "undefined")
        return self;
    if (typeof window !== "undefined")
        return window;
    if (typeof global !== "undefined")
        return global;
    throw "Unable to locate global object";
})();
function longToNumber(long) {
    if (long.gt(Number.MAX_SAFE_INTEGER)) {
        throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
    }
    return long.toNumber();
}
if (util.Long !== Long) {
    util.Long = Long;
    configure();
}
