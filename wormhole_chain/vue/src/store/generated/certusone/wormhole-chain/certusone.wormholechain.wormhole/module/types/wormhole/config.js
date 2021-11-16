/* eslint-disable */
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "certusone.wormholechain.wormhole";
const baseConfig = {
    guardianSetExpiration: 0,
    governanceChain: 0,
    chainId: 0,
};
export const Config = {
    encode(message, writer = Writer.create()) {
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
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseConfig };
        while (reader.pos < end) {
            const tag = reader.uint32();
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
    fromJSON(object) {
        const message = { ...baseConfig };
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
    toJSON(message) {
        const obj = {};
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
    fromPartial(object) {
        const message = { ...baseConfig };
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
if (util.Long !== Long) {
    util.Long = Long;
    configure();
}
