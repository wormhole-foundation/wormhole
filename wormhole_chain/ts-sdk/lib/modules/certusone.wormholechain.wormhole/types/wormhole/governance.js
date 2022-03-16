"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.GovernanceWormholeMessageProposal = exports.GuardianSetUpdateProposal = exports.protobufPackage = void 0;
//@ts-nocheck
/* eslint-disable */
const guardian_set_1 = require("../wormhole/guardian_set");
const minimal_1 = require("protobufjs/minimal");
exports.protobufPackage = "certusone.wormholechain.wormhole";
const baseGuardianSetUpdateProposal = { title: "", description: "" };
exports.GuardianSetUpdateProposal = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.title !== "") {
            writer.uint32(10).string(message.title);
        }
        if (message.description !== "") {
            writer.uint32(18).string(message.description);
        }
        if (message.newGuardianSet !== undefined) {
            guardian_set_1.GuardianSet.encode(message.newGuardianSet, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseGuardianSetUpdateProposal,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.title = reader.string();
                    break;
                case 2:
                    message.description = reader.string();
                    break;
                case 3:
                    message.newGuardianSet = guardian_set_1.GuardianSet.decode(reader, reader.uint32());
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
            ...baseGuardianSetUpdateProposal,
        };
        if (object.title !== undefined && object.title !== null) {
            message.title = String(object.title);
        }
        else {
            message.title = "";
        }
        if (object.description !== undefined && object.description !== null) {
            message.description = String(object.description);
        }
        else {
            message.description = "";
        }
        if (object.newGuardianSet !== undefined && object.newGuardianSet !== null) {
            message.newGuardianSet = guardian_set_1.GuardianSet.fromJSON(object.newGuardianSet);
        }
        else {
            message.newGuardianSet = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.title !== undefined && (obj.title = message.title);
        message.description !== undefined &&
            (obj.description = message.description);
        message.newGuardianSet !== undefined &&
            (obj.newGuardianSet = message.newGuardianSet
                ? guardian_set_1.GuardianSet.toJSON(message.newGuardianSet)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseGuardianSetUpdateProposal,
        };
        if (object.title !== undefined && object.title !== null) {
            message.title = object.title;
        }
        else {
            message.title = "";
        }
        if (object.description !== undefined && object.description !== null) {
            message.description = object.description;
        }
        else {
            message.description = "";
        }
        if (object.newGuardianSet !== undefined && object.newGuardianSet !== null) {
            message.newGuardianSet = guardian_set_1.GuardianSet.fromPartial(object.newGuardianSet);
        }
        else {
            message.newGuardianSet = undefined;
        }
        return message;
    },
};
const baseGovernanceWormholeMessageProposal = {
    title: "",
    description: "",
    action: 0,
    targetChain: 0,
};
exports.GovernanceWormholeMessageProposal = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.title !== "") {
            writer.uint32(10).string(message.title);
        }
        if (message.description !== "") {
            writer.uint32(18).string(message.description);
        }
        if (message.action !== 0) {
            writer.uint32(24).uint32(message.action);
        }
        if (message.module.length !== 0) {
            writer.uint32(34).bytes(message.module);
        }
        if (message.targetChain !== 0) {
            writer.uint32(40).uint32(message.targetChain);
        }
        if (message.payload.length !== 0) {
            writer.uint32(50).bytes(message.payload);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseGovernanceWormholeMessageProposal,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.title = reader.string();
                    break;
                case 2:
                    message.description = reader.string();
                    break;
                case 3:
                    message.action = reader.uint32();
                    break;
                case 4:
                    message.module = reader.bytes();
                    break;
                case 5:
                    message.targetChain = reader.uint32();
                    break;
                case 6:
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
        const message = {
            ...baseGovernanceWormholeMessageProposal,
        };
        if (object.title !== undefined && object.title !== null) {
            message.title = String(object.title);
        }
        else {
            message.title = "";
        }
        if (object.description !== undefined && object.description !== null) {
            message.description = String(object.description);
        }
        else {
            message.description = "";
        }
        if (object.action !== undefined && object.action !== null) {
            message.action = Number(object.action);
        }
        else {
            message.action = 0;
        }
        if (object.module !== undefined && object.module !== null) {
            message.module = bytesFromBase64(object.module);
        }
        if (object.targetChain !== undefined && object.targetChain !== null) {
            message.targetChain = Number(object.targetChain);
        }
        else {
            message.targetChain = 0;
        }
        if (object.payload !== undefined && object.payload !== null) {
            message.payload = bytesFromBase64(object.payload);
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.title !== undefined && (obj.title = message.title);
        message.description !== undefined &&
            (obj.description = message.description);
        message.action !== undefined && (obj.action = message.action);
        message.module !== undefined &&
            (obj.module = base64FromBytes(message.module !== undefined ? message.module : new Uint8Array()));
        message.targetChain !== undefined &&
            (obj.targetChain = message.targetChain);
        message.payload !== undefined &&
            (obj.payload = base64FromBytes(message.payload !== undefined ? message.payload : new Uint8Array()));
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseGovernanceWormholeMessageProposal,
        };
        if (object.title !== undefined && object.title !== null) {
            message.title = object.title;
        }
        else {
            message.title = "";
        }
        if (object.description !== undefined && object.description !== null) {
            message.description = object.description;
        }
        else {
            message.description = "";
        }
        if (object.action !== undefined && object.action !== null) {
            message.action = object.action;
        }
        else {
            message.action = 0;
        }
        if (object.module !== undefined && object.module !== null) {
            message.module = object.module;
        }
        else {
            message.module = new Uint8Array();
        }
        if (object.targetChain !== undefined && object.targetChain !== null) {
            message.targetChain = object.targetChain;
        }
        else {
            message.targetChain = 0;
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
