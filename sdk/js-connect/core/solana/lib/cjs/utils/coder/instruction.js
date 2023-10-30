"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
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
exports.WormholeInstruction = exports.WormholeInstructionCoder = void 0;
const connect_sdk_1 = require("@wormhole-foundation/connect-sdk");
const lodash_1 = require("lodash");
const borsh = __importStar(require("@coral-xyz/borsh"));
const idl_1 = require("./idl");
// Inspired by  coral-xyz/anchor
//
// https://github.com/coral-xyz/anchor/blob/master/ts/packages/anchor/src/coder/borsh/instruction.ts
class WormholeInstructionCoder {
    constructor(idl) {
        this.ixLayout = WormholeInstructionCoder.parseIxLayout(idl);
    }
    static parseIxLayout(idl) {
        const stateMethods = idl.state ? idl.state.methods : [];
        const ixLayouts = stateMethods
            .map((m) => {
            let fieldLayouts = m.args.map((arg) => {
                var _a, _b;
                return idl_1.IdlCoder.fieldLayout(arg, Array.from([...((_a = idl.accounts) !== null && _a !== void 0 ? _a : []), ...((_b = idl.types) !== null && _b !== void 0 ? _b : [])]));
            });
            const name = (0, lodash_1.camelCase)(m.name);
            return [name, borsh.struct(fieldLayouts, name)];
        })
            .concat(idl.instructions.map((ix) => {
            let fieldLayouts = ix.args.map((arg) => {
                var _a, _b;
                return idl_1.IdlCoder.fieldLayout(arg, Array.from([...((_a = idl.accounts) !== null && _a !== void 0 ? _a : []), ...((_b = idl.types) !== null && _b !== void 0 ? _b : [])]));
            });
            const name = (0, lodash_1.camelCase)(ix.name);
            return [name, borsh.struct(fieldLayouts, name)];
        }));
        return new Map(ixLayouts);
    }
    encode(ixName, ix) {
        const buffer = Buffer.alloc(1000); // TODO: use a tighter buffer.
        const methodName = (0, lodash_1.camelCase)(ixName);
        const layout = this.ixLayout.get(methodName);
        if (!layout) {
            throw new Error(`Unknown method: ${methodName}`);
        }
        const len = layout.encode(ix, buffer);
        const data = buffer.slice(0, len);
        return encodeWormholeInstructionData(WormholeInstruction[(0, lodash_1.upperFirst)(methodName)], data);
    }
    encodeState(_ixName, _ix) {
        throw new Error('Wormhole program does not have state');
    }
    decode(ix, _encoding = 'hex') {
        var _a;
        if (typeof ix === 'string') {
            ix =
                _encoding === 'hex' ? Buffer.from(ix, 'hex') : connect_sdk_1.encoding.b58.decode(ix);
        }
        let discriminator = Buffer.from(ix.slice(0, 1)).readInt8();
        let data = Buffer.from(ix.slice(1));
        let name = (0, lodash_1.camelCase)(WormholeInstruction[discriminator]);
        let layout = this.ixLayout.get(name);
        if (!layout) {
            return null;
        }
        return { data: (_a = this.ixLayout.get(name)) === null || _a === void 0 ? void 0 : _a.decode(data), name };
    }
}
exports.WormholeInstructionCoder = WormholeInstructionCoder;
/** Solitaire enum of existing the Core Bridge's instructions.
 *
 * https://github.com/certusone/wormhole/blob/main/solana/bridge/program/src/lib.rs#L92
 */
var WormholeInstruction;
(function (WormholeInstruction) {
    WormholeInstruction[WormholeInstruction["Initialize"] = 0] = "Initialize";
    WormholeInstruction[WormholeInstruction["PostMessage"] = 1] = "PostMessage";
    WormholeInstruction[WormholeInstruction["PostVaa"] = 2] = "PostVaa";
    WormholeInstruction[WormholeInstruction["SetFees"] = 3] = "SetFees";
    WormholeInstruction[WormholeInstruction["TransferFees"] = 4] = "TransferFees";
    WormholeInstruction[WormholeInstruction["UpgradeContract"] = 5] = "UpgradeContract";
    WormholeInstruction[WormholeInstruction["UpgradeGuardianSet"] = 6] = "UpgradeGuardianSet";
    WormholeInstruction[WormholeInstruction["VerifySignatures"] = 7] = "VerifySignatures";
    WormholeInstruction[WormholeInstruction["PostMessageUnreliable"] = 8] = "PostMessageUnreliable";
})(WormholeInstruction || (exports.WormholeInstruction = WormholeInstruction = {}));
function encodeWormholeInstructionData(discriminator, data) {
    const instructionData = Buffer.alloc(1 + (data === undefined ? 0 : data.length));
    instructionData.writeUInt8(discriminator, 0);
    if (data !== undefined) {
        instructionData.write(data.toString('hex'), 1, 'hex');
    }
    return instructionData;
}
