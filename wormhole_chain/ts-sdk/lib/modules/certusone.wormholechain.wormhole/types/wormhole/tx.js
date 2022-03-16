"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.MsgClientImpl = exports.MsgRegisterAccountAsGuardianResponse = exports.MsgRegisterAccountAsGuardian = exports.MsgExecuteGovernanceVAAResponse = exports.MsgExecuteGovernanceVAA = exports.protobufPackage = void 0;
//@ts-nocheck
/* eslint-disable */
const minimal_1 = require("protobufjs/minimal");
const guardian_key_1 = require("../wormhole/guardian_key");
exports.protobufPackage = "certusone.wormholechain.wormhole";
const baseMsgExecuteGovernanceVAA = { signer: "" };
exports.MsgExecuteGovernanceVAA = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.vaa.length !== 0) {
            writer.uint32(10).bytes(message.vaa);
        }
        if (message.signer !== "") {
            writer.uint32(18).string(message.signer);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseMsgExecuteGovernanceVAA,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
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
    fromJSON(object) {
        const message = {
            ...baseMsgExecuteGovernanceVAA,
        };
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
    toJSON(message) {
        const obj = {};
        message.vaa !== undefined &&
            (obj.vaa = base64FromBytes(message.vaa !== undefined ? message.vaa : new Uint8Array()));
        message.signer !== undefined && (obj.signer = message.signer);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseMsgExecuteGovernanceVAA,
        };
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
const baseMsgExecuteGovernanceVAAResponse = {};
exports.MsgExecuteGovernanceVAAResponse = {
    encode(_, writer = minimal_1.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseMsgExecuteGovernanceVAAResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(_) {
        const message = {
            ...baseMsgExecuteGovernanceVAAResponse,
        };
        return message;
    },
    toJSON(_) {
        const obj = {};
        return obj;
    },
    fromPartial(_) {
        const message = {
            ...baseMsgExecuteGovernanceVAAResponse,
        };
        return message;
    },
};
const baseMsgRegisterAccountAsGuardian = {
    signer: "",
    addressBech32: "",
};
exports.MsgRegisterAccountAsGuardian = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.signer !== "") {
            writer.uint32(10).string(message.signer);
        }
        if (message.guardianPubkey !== undefined) {
            guardian_key_1.GuardianKey.encode(message.guardianPubkey, writer.uint32(18).fork()).ldelim();
        }
        if (message.addressBech32 !== "") {
            writer.uint32(26).string(message.addressBech32);
        }
        if (message.signature.length !== 0) {
            writer.uint32(34).bytes(message.signature);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseMsgRegisterAccountAsGuardian,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.signer = reader.string();
                    break;
                case 2:
                    message.guardianPubkey = guardian_key_1.GuardianKey.decode(reader, reader.uint32());
                    break;
                case 3:
                    message.addressBech32 = reader.string();
                    break;
                case 4:
                    message.signature = reader.bytes();
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
            ...baseMsgRegisterAccountAsGuardian,
        };
        if (object.signer !== undefined && object.signer !== null) {
            message.signer = String(object.signer);
        }
        else {
            message.signer = "";
        }
        if (object.guardianPubkey !== undefined && object.guardianPubkey !== null) {
            message.guardianPubkey = guardian_key_1.GuardianKey.fromJSON(object.guardianPubkey);
        }
        else {
            message.guardianPubkey = undefined;
        }
        if (object.addressBech32 !== undefined && object.addressBech32 !== null) {
            message.addressBech32 = String(object.addressBech32);
        }
        else {
            message.addressBech32 = "";
        }
        if (object.signature !== undefined && object.signature !== null) {
            message.signature = bytesFromBase64(object.signature);
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.signer !== undefined && (obj.signer = message.signer);
        message.guardianPubkey !== undefined &&
            (obj.guardianPubkey = message.guardianPubkey
                ? guardian_key_1.GuardianKey.toJSON(message.guardianPubkey)
                : undefined);
        message.addressBech32 !== undefined &&
            (obj.addressBech32 = message.addressBech32);
        message.signature !== undefined &&
            (obj.signature = base64FromBytes(message.signature !== undefined ? message.signature : new Uint8Array()));
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseMsgRegisterAccountAsGuardian,
        };
        if (object.signer !== undefined && object.signer !== null) {
            message.signer = object.signer;
        }
        else {
            message.signer = "";
        }
        if (object.guardianPubkey !== undefined && object.guardianPubkey !== null) {
            message.guardianPubkey = guardian_key_1.GuardianKey.fromPartial(object.guardianPubkey);
        }
        else {
            message.guardianPubkey = undefined;
        }
        if (object.addressBech32 !== undefined && object.addressBech32 !== null) {
            message.addressBech32 = object.addressBech32;
        }
        else {
            message.addressBech32 = "";
        }
        if (object.signature !== undefined && object.signature !== null) {
            message.signature = object.signature;
        }
        else {
            message.signature = new Uint8Array();
        }
        return message;
    },
};
const baseMsgRegisterAccountAsGuardianResponse = {};
exports.MsgRegisterAccountAsGuardianResponse = {
    encode(_, writer = minimal_1.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseMsgRegisterAccountAsGuardianResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(_) {
        const message = {
            ...baseMsgRegisterAccountAsGuardianResponse,
        };
        return message;
    },
    toJSON(_) {
        const obj = {};
        return obj;
    },
    fromPartial(_) {
        const message = {
            ...baseMsgRegisterAccountAsGuardianResponse,
        };
        return message;
    },
};
class MsgClientImpl {
    rpc;
    constructor(rpc) {
        this.rpc = rpc;
    }
    ExecuteGovernanceVAA(request) {
        const data = exports.MsgExecuteGovernanceVAA.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Msg", "ExecuteGovernanceVAA", data);
        return promise.then((data) => exports.MsgExecuteGovernanceVAAResponse.decode(new minimal_1.Reader(data)));
    }
    RegisterAccountAsGuardian(request) {
        const data = exports.MsgRegisterAccountAsGuardian.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Msg", "RegisterAccountAsGuardian", data);
        return promise.then((data) => exports.MsgRegisterAccountAsGuardianResponse.decode(new minimal_1.Reader(data)));
    }
}
exports.MsgClientImpl = MsgClientImpl;
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
