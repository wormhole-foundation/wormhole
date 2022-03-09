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
export var protobufPackage = "certusone.wormholechain.tokenbridge";
var baseEventChainRegistered = { chainID: 0 };
export var EventChainRegistered = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.chainID !== 0) {
            writer.uint32(8).uint32(message.chainID);
        }
        if (message.emitterAddress.length !== 0) {
            writer.uint32(18).bytes(message.emitterAddress);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseEventChainRegistered);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.chainID = reader.uint32();
                    break;
                case 2:
                    message.emitterAddress = reader.bytes();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseEventChainRegistered);
        if (object.chainID !== undefined && object.chainID !== null) {
            message.chainID = Number(object.chainID);
        }
        else {
            message.chainID = 0;
        }
        if (object.emitterAddress !== undefined && object.emitterAddress !== null) {
            message.emitterAddress = bytesFromBase64(object.emitterAddress);
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.chainID !== undefined && (obj.chainID = message.chainID);
        message.emitterAddress !== undefined &&
            (obj.emitterAddress = base64FromBytes(message.emitterAddress !== undefined
                ? message.emitterAddress
                : new Uint8Array()));
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseEventChainRegistered);
        if (object.chainID !== undefined && object.chainID !== null) {
            message.chainID = object.chainID;
        }
        else {
            message.chainID = 0;
        }
        if (object.emitterAddress !== undefined && object.emitterAddress !== null) {
            message.emitterAddress = object.emitterAddress;
        }
        else {
            message.emitterAddress = new Uint8Array();
        }
        return message;
    },
};
var baseEventAssetRegistrationUpdate = {
    tokenChain: 0,
    name: "",
    symbol: "",
    decimals: 0,
};
export var EventAssetRegistrationUpdate = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.tokenChain !== 0) {
            writer.uint32(8).uint32(message.tokenChain);
        }
        if (message.tokenAddress.length !== 0) {
            writer.uint32(18).bytes(message.tokenAddress);
        }
        if (message.name !== "") {
            writer.uint32(26).string(message.name);
        }
        if (message.symbol !== "") {
            writer.uint32(34).string(message.symbol);
        }
        if (message.decimals !== 0) {
            writer.uint32(40).uint32(message.decimals);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseEventAssetRegistrationUpdate);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.tokenChain = reader.uint32();
                    break;
                case 2:
                    message.tokenAddress = reader.bytes();
                    break;
                case 3:
                    message.name = reader.string();
                    break;
                case 4:
                    message.symbol = reader.string();
                    break;
                case 5:
                    message.decimals = reader.uint32();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseEventAssetRegistrationUpdate);
        if (object.tokenChain !== undefined && object.tokenChain !== null) {
            message.tokenChain = Number(object.tokenChain);
        }
        else {
            message.tokenChain = 0;
        }
        if (object.tokenAddress !== undefined && object.tokenAddress !== null) {
            message.tokenAddress = bytesFromBase64(object.tokenAddress);
        }
        if (object.name !== undefined && object.name !== null) {
            message.name = String(object.name);
        }
        else {
            message.name = "";
        }
        if (object.symbol !== undefined && object.symbol !== null) {
            message.symbol = String(object.symbol);
        }
        else {
            message.symbol = "";
        }
        if (object.decimals !== undefined && object.decimals !== null) {
            message.decimals = Number(object.decimals);
        }
        else {
            message.decimals = 0;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.tokenChain !== undefined && (obj.tokenChain = message.tokenChain);
        message.tokenAddress !== undefined &&
            (obj.tokenAddress = base64FromBytes(message.tokenAddress !== undefined
                ? message.tokenAddress
                : new Uint8Array()));
        message.name !== undefined && (obj.name = message.name);
        message.symbol !== undefined && (obj.symbol = message.symbol);
        message.decimals !== undefined && (obj.decimals = message.decimals);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseEventAssetRegistrationUpdate);
        if (object.tokenChain !== undefined && object.tokenChain !== null) {
            message.tokenChain = object.tokenChain;
        }
        else {
            message.tokenChain = 0;
        }
        if (object.tokenAddress !== undefined && object.tokenAddress !== null) {
            message.tokenAddress = object.tokenAddress;
        }
        else {
            message.tokenAddress = new Uint8Array();
        }
        if (object.name !== undefined && object.name !== null) {
            message.name = object.name;
        }
        else {
            message.name = "";
        }
        if (object.symbol !== undefined && object.symbol !== null) {
            message.symbol = object.symbol;
        }
        else {
            message.symbol = "";
        }
        if (object.decimals !== undefined && object.decimals !== null) {
            message.decimals = object.decimals;
        }
        else {
            message.decimals = 0;
        }
        return message;
    },
};
var baseEventTransferReceived = {
    tokenChain: 0,
    to: "",
    feeRecipient: "",
    amount: "",
    fee: "",
    localDenom: "",
};
export var EventTransferReceived = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.tokenChain !== 0) {
            writer.uint32(8).uint32(message.tokenChain);
        }
        if (message.tokenAddress.length !== 0) {
            writer.uint32(18).bytes(message.tokenAddress);
        }
        if (message.to !== "") {
            writer.uint32(26).string(message.to);
        }
        if (message.feeRecipient !== "") {
            writer.uint32(34).string(message.feeRecipient);
        }
        if (message.amount !== "") {
            writer.uint32(42).string(message.amount);
        }
        if (message.fee !== "") {
            writer.uint32(50).string(message.fee);
        }
        if (message.localDenom !== "") {
            writer.uint32(58).string(message.localDenom);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseEventTransferReceived);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.tokenChain = reader.uint32();
                    break;
                case 2:
                    message.tokenAddress = reader.bytes();
                    break;
                case 3:
                    message.to = reader.string();
                    break;
                case 4:
                    message.feeRecipient = reader.string();
                    break;
                case 5:
                    message.amount = reader.string();
                    break;
                case 6:
                    message.fee = reader.string();
                    break;
                case 7:
                    message.localDenom = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseEventTransferReceived);
        if (object.tokenChain !== undefined && object.tokenChain !== null) {
            message.tokenChain = Number(object.tokenChain);
        }
        else {
            message.tokenChain = 0;
        }
        if (object.tokenAddress !== undefined && object.tokenAddress !== null) {
            message.tokenAddress = bytesFromBase64(object.tokenAddress);
        }
        if (object.to !== undefined && object.to !== null) {
            message.to = String(object.to);
        }
        else {
            message.to = "";
        }
        if (object.feeRecipient !== undefined && object.feeRecipient !== null) {
            message.feeRecipient = String(object.feeRecipient);
        }
        else {
            message.feeRecipient = "";
        }
        if (object.amount !== undefined && object.amount !== null) {
            message.amount = String(object.amount);
        }
        else {
            message.amount = "";
        }
        if (object.fee !== undefined && object.fee !== null) {
            message.fee = String(object.fee);
        }
        else {
            message.fee = "";
        }
        if (object.localDenom !== undefined && object.localDenom !== null) {
            message.localDenom = String(object.localDenom);
        }
        else {
            message.localDenom = "";
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.tokenChain !== undefined && (obj.tokenChain = message.tokenChain);
        message.tokenAddress !== undefined &&
            (obj.tokenAddress = base64FromBytes(message.tokenAddress !== undefined
                ? message.tokenAddress
                : new Uint8Array()));
        message.to !== undefined && (obj.to = message.to);
        message.feeRecipient !== undefined &&
            (obj.feeRecipient = message.feeRecipient);
        message.amount !== undefined && (obj.amount = message.amount);
        message.fee !== undefined && (obj.fee = message.fee);
        message.localDenom !== undefined && (obj.localDenom = message.localDenom);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseEventTransferReceived);
        if (object.tokenChain !== undefined && object.tokenChain !== null) {
            message.tokenChain = object.tokenChain;
        }
        else {
            message.tokenChain = 0;
        }
        if (object.tokenAddress !== undefined && object.tokenAddress !== null) {
            message.tokenAddress = object.tokenAddress;
        }
        else {
            message.tokenAddress = new Uint8Array();
        }
        if (object.to !== undefined && object.to !== null) {
            message.to = object.to;
        }
        else {
            message.to = "";
        }
        if (object.feeRecipient !== undefined && object.feeRecipient !== null) {
            message.feeRecipient = object.feeRecipient;
        }
        else {
            message.feeRecipient = "";
        }
        if (object.amount !== undefined && object.amount !== null) {
            message.amount = object.amount;
        }
        else {
            message.amount = "";
        }
        if (object.fee !== undefined && object.fee !== null) {
            message.fee = object.fee;
        }
        else {
            message.fee = "";
        }
        if (object.localDenom !== undefined && object.localDenom !== null) {
            message.localDenom = object.localDenom;
        }
        else {
            message.localDenom = "";
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
