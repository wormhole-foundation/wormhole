"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.GuardianValidator = exports.protobufPackage = void 0;
//@ts-nocheck
/* eslint-disable */
const minimal_1 = require("protobufjs/minimal");
exports.protobufPackage = "certusone.wormholechain.wormhole";
const baseGuardianValidator = {};
exports.GuardianValidator = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.guardianKey.length !== 0) {
            writer.uint32(10).bytes(message.guardianKey);
        }
        if (message.validatorAddr.length !== 0) {
            writer.uint32(18).bytes(message.validatorAddr);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseGuardianValidator };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.guardianKey = reader.bytes();
                    break;
                case 2:
                    message.validatorAddr = reader.bytes();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseGuardianValidator };
        if (object.guardianKey !== undefined && object.guardianKey !== null) {
            message.guardianKey = bytesFromBase64(object.guardianKey);
        }
        if (object.validatorAddr !== undefined && object.validatorAddr !== null) {
            message.validatorAddr = bytesFromBase64(object.validatorAddr);
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.guardianKey !== undefined &&
            (obj.guardianKey = base64FromBytes(message.guardianKey !== undefined
                ? message.guardianKey
                : new Uint8Array()));
        message.validatorAddr !== undefined &&
            (obj.validatorAddr = base64FromBytes(message.validatorAddr !== undefined
                ? message.validatorAddr
                : new Uint8Array()));
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseGuardianValidator };
        if (object.guardianKey !== undefined && object.guardianKey !== null) {
            message.guardianKey = object.guardianKey;
        }
        else {
            message.guardianKey = new Uint8Array();
        }
        if (object.validatorAddr !== undefined && object.validatorAddr !== null) {
            message.validatorAddr = object.validatorAddr;
        }
        else {
            message.validatorAddr = new Uint8Array();
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
