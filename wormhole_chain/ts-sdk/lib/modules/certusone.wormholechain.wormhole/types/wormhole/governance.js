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
import { GuardianSet } from "../wormhole/guardian_set";
import { Writer, Reader } from "protobufjs/minimal";
export var protobufPackage = "certusone.wormholechain.wormhole";
var baseGuardianSetUpdateProposal = { title: "", description: "" };
export var GuardianSetUpdateProposal = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.title !== "") {
            writer.uint32(10).string(message.title);
        }
        if (message.description !== "") {
            writer.uint32(18).string(message.description);
        }
        if (message.newGuardianSet !== undefined) {
            GuardianSet.encode(message.newGuardianSet, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseGuardianSetUpdateProposal);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.title = reader.string();
                    break;
                case 2:
                    message.description = reader.string();
                    break;
                case 3:
                    message.newGuardianSet = GuardianSet.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseGuardianSetUpdateProposal);
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
            message.newGuardianSet = GuardianSet.fromJSON(object.newGuardianSet);
        }
        else {
            message.newGuardianSet = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.title !== undefined && (obj.title = message.title);
        message.description !== undefined &&
            (obj.description = message.description);
        message.newGuardianSet !== undefined &&
            (obj.newGuardianSet = message.newGuardianSet
                ? GuardianSet.toJSON(message.newGuardianSet)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseGuardianSetUpdateProposal);
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
            message.newGuardianSet = GuardianSet.fromPartial(object.newGuardianSet);
        }
        else {
            message.newGuardianSet = undefined;
        }
        return message;
    },
};
var baseGovernanceWormholeMessageProposal = {
    title: "",
    description: "",
    action: 0,
    targetChain: 0,
};
export var GovernanceWormholeMessageProposal = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
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
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseGovernanceWormholeMessageProposal);
        while (reader.pos < end) {
            var tag = reader.uint32();
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
    fromJSON: function (object) {
        var message = __assign({}, baseGovernanceWormholeMessageProposal);
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
    toJSON: function (message) {
        var obj = {};
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
    fromPartial: function (object) {
        var message = __assign({}, baseGovernanceWormholeMessageProposal);
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
