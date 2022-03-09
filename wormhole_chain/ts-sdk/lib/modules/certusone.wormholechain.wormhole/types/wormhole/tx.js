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
import { Reader, Writer } from "protobufjs/minimal";
export var protobufPackage = "certusone.wormholechain.wormhole";
var baseMsgExecuteGovernanceVAA = { signer: "" };
export var MsgExecuteGovernanceVAA = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.vaa.length !== 0) {
            writer.uint32(10).bytes(message.vaa);
        }
        if (message.signer !== "") {
            writer.uint32(18).string(message.signer);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseMsgExecuteGovernanceVAA);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.vaa = reader.bytes();
                    break;
                case 2:
                    message.signer = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseMsgExecuteGovernanceVAA);
        if (object.vaa !== undefined && object.vaa !== null) {
            message.vaa = bytesFromBase64(object.vaa);
        }
        if (object.signer !== undefined && object.signer !== null) {
            message.signer = String(object.signer);
        }
        else {
            message.signer = "";
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.vaa !== undefined &&
            (obj.vaa = base64FromBytes(message.vaa !== undefined ? message.vaa : new Uint8Array()));
        message.signer !== undefined && (obj.signer = message.signer);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseMsgExecuteGovernanceVAA);
        if (object.vaa !== undefined && object.vaa !== null) {
            message.vaa = object.vaa;
        }
        else {
            message.vaa = new Uint8Array();
        }
        if (object.signer !== undefined && object.signer !== null) {
            message.signer = object.signer;
        }
        else {
            message.signer = "";
        }
        return message;
    },
};
var baseMsgExecuteGovernanceVAAResponse = {};
export var MsgExecuteGovernanceVAAResponse = {
    encode: function (_, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseMsgExecuteGovernanceVAAResponse);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (_) {
        var message = __assign({}, baseMsgExecuteGovernanceVAAResponse);
        return message;
    },
    toJSON: function (_) {
        var obj = {};
        return obj;
    },
    fromPartial: function (_) {
        var message = __assign({}, baseMsgExecuteGovernanceVAAResponse);
        return message;
    },
};
var MsgClientImpl = /** @class */ (function () {
    function MsgClientImpl(rpc) {
        this.rpc = rpc;
    }
    MsgClientImpl.prototype.ExecuteGovernanceVAA = function (request) {
        var data = MsgExecuteGovernanceVAA.encode(request).finish();
        var promise = this.rpc.request("certusone.wormholechain.wormhole.Msg", "ExecuteGovernanceVAA", data);
        return promise.then(function (data) {
            return MsgExecuteGovernanceVAAResponse.decode(new Reader(data));
        });
    };
    return MsgClientImpl;
}());
export { MsgClientImpl };
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
