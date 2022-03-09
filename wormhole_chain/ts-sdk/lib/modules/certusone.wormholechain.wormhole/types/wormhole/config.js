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
var baseConfig = {
    guardianSetExpiration: 0,
    governanceChain: 0,
    chainId: 0,
};
export var Config = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.guardianSetExpiration !== 0) {
            writer.uint32(8).uint64(message.guardianSetExpiration);
        }
        if (message.governanceEmitter.length !== 0) {
            writer.uint32(18).bytes(message.governanceEmitter);
        }
        if (message.governanceChain !== 0) {
            writer.uint32(24).uint32(message.governanceChain);
        }
        if (message.chainId !== 0) {
            writer.uint32(32).uint32(message.chainId);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseConfig);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.guardianSetExpiration = longToNumber(reader.uint64());
                    break;
                case 2:
                    message.governanceEmitter = reader.bytes();
                    break;
                case 3:
                    message.governanceChain = reader.uint32();
                    break;
                case 4:
                    message.chainId = reader.uint32();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseConfig);
        if (object.guardianSetExpiration !== undefined &&
            object.guardianSetExpiration !== null) {
            message.guardianSetExpiration = Number(object.guardianSetExpiration);
        }
        else {
            message.guardianSetExpiration = 0;
        }
        if (object.governanceEmitter !== undefined &&
            object.governanceEmitter !== null) {
            message.governanceEmitter = bytesFromBase64(object.governanceEmitter);
        }
        if (object.governanceChain !== undefined &&
            object.governanceChain !== null) {
            message.governanceChain = Number(object.governanceChain);
        }
        else {
            message.governanceChain = 0;
        }
        if (object.chainId !== undefined && object.chainId !== null) {
            message.chainId = Number(object.chainId);
        }
        else {
            message.chainId = 0;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.guardianSetExpiration !== undefined &&
            (obj.guardianSetExpiration = message.guardianSetExpiration);
        message.governanceEmitter !== undefined &&
            (obj.governanceEmitter = base64FromBytes(message.governanceEmitter !== undefined
                ? message.governanceEmitter
                : new Uint8Array()));
        message.governanceChain !== undefined &&
            (obj.governanceChain = message.governanceChain);
        message.chainId !== undefined && (obj.chainId = message.chainId);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseConfig);
        if (object.guardianSetExpiration !== undefined &&
            object.guardianSetExpiration !== null) {
            message.guardianSetExpiration = object.guardianSetExpiration;
        }
        else {
            message.guardianSetExpiration = 0;
        }
        if (object.governanceEmitter !== undefined &&
            object.governanceEmitter !== null) {
            message.governanceEmitter = object.governanceEmitter;
        }
        else {
            message.governanceEmitter = new Uint8Array();
        }
        if (object.governanceChain !== undefined &&
            object.governanceChain !== null) {
            message.governanceChain = object.governanceChain;
        }
        else {
            message.governanceChain = 0;
        }
        if (object.chainId !== undefined && object.chainId !== null) {
            message.chainId = object.chainId;
        }
        else {
            message.chainId = 0;
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
