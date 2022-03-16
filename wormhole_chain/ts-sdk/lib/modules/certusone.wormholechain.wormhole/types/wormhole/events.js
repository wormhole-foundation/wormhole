"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    Object.defineProperty(o, k2, { enumerable: true, get: function() { return m[k]; } });
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.EventPostedMessage = exports.EventGuardianSetUpdate = exports.protobufPackage = void 0;
//@ts-nocheck
/* eslint-disable */
const Long = __importStar(require("long"));
const minimal_1 = require("protobufjs/minimal");
exports.protobufPackage = "certusone.wormholechain.wormhole";
const baseEventGuardianSetUpdate = { oldIndex: 0, newIndex: 0 };
exports.EventGuardianSetUpdate = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.oldIndex !== 0) {
            writer.uint32(8).uint32(message.oldIndex);
        }
        if (message.newIndex !== 0) {
            writer.uint32(16).uint32(message.newIndex);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
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
const baseEventPostedMessage = { sequence: 0, nonce: 0 };
exports.EventPostedMessage = {
    encode(message, writer = minimal_1.Writer.create()) {
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
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseEventPostedMessage };
        while (reader.pos < end) {
            const tag = reader.uint32();
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
    fromJSON(object) {
        const message = { ...baseEventPostedMessage };
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
    toJSON(message) {
        const obj = {};
        message.emitter !== undefined &&
            (obj.emitter = base64FromBytes(message.emitter !== undefined ? message.emitter : new Uint8Array()));
        message.sequence !== undefined && (obj.sequence = message.sequence);
        message.nonce !== undefined && (obj.nonce = message.nonce);
        message.payload !== undefined &&
            (obj.payload = base64FromBytes(message.payload !== undefined ? message.payload : new Uint8Array()));
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseEventPostedMessage };
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
const atob = globalThis.atob ||
    ((b64) => globalThis.Buffer.from(b64, "base64").toString("binary"));
function bytesFromBase64(b64) {
    const bin = atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; ++i) {
        arr[i] = bin.charCodeAt(i);
    }
    return arr;
}
const btoa = globalThis.btoa ||
    ((bin) => globalThis.Buffer.from(bin, "binary").toString("base64"));
function base64FromBytes(arr) {
    const bin = [];
    for (let i = 0; i < arr.byteLength; ++i) {
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
if (minimal_1.util.Long !== Long) {
    minimal_1.util.Long = Long;
    (0, minimal_1.configure)();
}
