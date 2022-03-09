var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
//@ts-nocheck
/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
export var protobufPackage = "certusone.wormholechain.tokenbridge";
var baseCoinMetaRollbackProtection = {
    index: "",
    lastUpdateSequence: 0,
};
export var CoinMetaRollbackProtection = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        if (message.lastUpdateSequence !== 0) {
            writer.uint32(16).uint64(message.lastUpdateSequence);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseCoinMetaRollbackProtection);
        while (reader.pos < end) {
            var tag = reader.uint32();
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
    fromJSON: function (object) {
        var message = __assign({}, baseCoinMetaRollbackProtection);
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
    toJSON: function (message) {
        var obj = {};
        message.index !== undefined && (obj.index = message.index);
        message.lastUpdateSequence !== undefined &&
            (obj.lastUpdateSequence = message.lastUpdateSequence);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseCoinMetaRollbackProtection);
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
var globalThis = (function () {
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
