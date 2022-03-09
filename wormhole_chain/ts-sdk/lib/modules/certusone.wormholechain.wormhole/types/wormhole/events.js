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
export var protobufPackage = "certusone.wormholechain.wormhole";
var baseEventGuardianSetUpdate = { oldIndex: 0, newIndex: 0 };
export var EventGuardianSetUpdate = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.oldIndex !== 0) {
            writer.uint32(8).uint32(message.oldIndex);
        }
        if (message.newIndex !== 0) {
            writer.uint32(16).uint32(message.newIndex);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseEventGuardianSetUpdate);
        while (reader.pos < end) {
            var tag = reader.uint32();
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
    fromJSON: function (object) {
        var message = __assign({}, baseEventGuardianSetUpdate);
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
    toJSON: function (message) {
        var obj = {};
        message.oldIndex !== undefined && (obj.oldIndex = message.oldIndex);
        message.newIndex !== undefined && (obj.newIndex = message.newIndex);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseEventGuardianSetUpdate);
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
var baseEventPostedMessage = { sequence: 0, nonce: 0 };
export var EventPostedMessage = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.emitter.length !== 0) {
            writer.uint32(10).bytes(message.emitter);
        }
        if (message.sequence !== 0) {
            writer.uint32(16).uint64(message.sequence);
        }
        if (message.nonce !== 0) {
            writer.uint32(24).uint32(message.nonce);
        }
        if (message.payload.length !== 0) {
            writer.uint32(34).bytes(message.payload);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseEventPostedMessage);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.emitter = reader.bytes();
                    break;
                case 2:
                    message.sequence = longToNumber(reader.uint64());
                    break;
                case 3:
                    message.nonce = reader.uint32();
                    break;
                case 4:
                    message.payload = reader.bytes();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseEventPostedMessage);
        if (object.emitter !== undefined && object.emitter !== null) {
            message.emitter = bytesFromBase64(object.emitter);
        }
        if (object.sequence !== undefined && object.sequence !== null) {
            message.sequence = Number(object.sequence);
        }
        else {
            message.sequence = 0;
        }
        if (object.nonce !== undefined && object.nonce !== null) {
            message.nonce = Number(object.nonce);
        }
        else {
            message.nonce = 0;
        }
        if (object.payload !== undefined && object.payload !== null) {
            message.payload = bytesFromBase64(object.payload);
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.emitter !== undefined &&
            (obj.emitter = base64FromBytes(message.emitter !== undefined ? message.emitter : new Uint8Array()));
        message.sequence !== undefined && (obj.sequence = message.sequence);
        message.nonce !== undefined && (obj.nonce = message.nonce);
        message.payload !== undefined &&
            (obj.payload = base64FromBytes(message.payload !== undefined ? message.payload : new Uint8Array()));
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseEventPostedMessage);
        if (object.emitter !== undefined && object.emitter !== null) {
            message.emitter = object.emitter;
        }
        else {
            message.emitter = new Uint8Array();
        }
        if (object.sequence !== undefined && object.sequence !== null) {
            message.sequence = object.sequence;
        }
        else {
            message.sequence = 0;
        }
        if (object.nonce !== undefined && object.nonce !== null) {
            message.nonce = object.nonce;
        }
        else {
            message.nonce = 0;
        }
        if (object.payload !== undefined && object.payload !== null) {
            message.payload = object.payload;
        }
        else {
            message.payload = new Uint8Array();
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
var atob = globalThis.atob ||
    (function (b64) { return globalThis.Buffer.from(b64, "base64").toString("binary"); });
function bytesFromBase64(b64) {
    var bin = atob(b64);
    var arr = new Uint8Array(bin.length);
    for (var i = 0; i < bin.length; ++i) {
        arr[i] = bin.charCodeAt(i);
    }
    return arr;
}
var btoa = globalThis.btoa ||
    (function (bin) { return globalThis.Buffer.from(bin, "binary").toString("base64"); });
function base64FromBytes(arr) {
    var bin = [];
    for (var i = 0; i < arr.byteLength; ++i) {
        bin.push(String.fromCharCode(arr[i]));
    }
    return btoa(bin.join(""));
}
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
