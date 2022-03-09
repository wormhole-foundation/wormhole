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
import { Writer, Reader } from "protobufjs/minimal";
export var protobufPackage = "certusone.wormholechain.wormhole";
var baseReplayProtection = { index: "" };
export var ReplayProtection = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseReplayProtection);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.index = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseReplayProtection);
        if (object.index !== undefined && object.index !== null) {
            message.index = String(object.index);
        }
        else {
            message.index = "";
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseReplayProtection);
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = "";
        }
        return message;
    },
};
