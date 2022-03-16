"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.GuardianKey = exports.protobufPackage = void 0;
//@ts-nocheck
/* eslint-disable */
const minimal_1 = require("protobufjs/minimal");
exports.protobufPackage = "certusone.wormholechain.wormhole";
const baseGuardianKey = {};
exports.GuardianKey = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.key.length !== 0) {
            writer.uint32(10).bytes(message.key);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseGuardianKey };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.key = reader.bytes();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseGuardianKey };
        if (object.key !== undefined && object.key !== null) {
            message.key = bytesFromBase64(object.key);
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.key !== undefined &&
            (obj.key = base64FromBytes(message.key !== undefined ? message.key : new Uint8Array()));
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseGuardianKey };
        if (object.key !== undefined && object.key !== null) {
            message.key = object.key;
        }
        else {
            message.key = new Uint8Array();
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
