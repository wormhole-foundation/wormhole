import { encoding } from '@wormhole-foundation/connect-sdk';
import { camelCase, upperFirst } from 'lodash';
import * as borsh from '@coral-xyz/borsh';
import { IdlCoder } from './idl';
// Inspired by  coral-xyz/anchor
//
// https://github.com/coral-xyz/anchor/blob/master/ts/packages/anchor/src/coder/borsh/instruction.ts
export class WormholeInstructionCoder {
    constructor(idl) {
        this.ixLayout = WormholeInstructionCoder.parseIxLayout(idl);
    }
    static parseIxLayout(idl) {
        const stateMethods = idl.state ? idl.state.methods : [];
        const ixLayouts = stateMethods
            .map((m) => {
            let fieldLayouts = m.args.map((arg) => {
                var _a, _b;
                return IdlCoder.fieldLayout(arg, Array.from([...((_a = idl.accounts) !== null && _a !== void 0 ? _a : []), ...((_b = idl.types) !== null && _b !== void 0 ? _b : [])]));
            });
            const name = camelCase(m.name);
            return [name, borsh.struct(fieldLayouts, name)];
        })
            .concat(idl.instructions.map((ix) => {
            let fieldLayouts = ix.args.map((arg) => {
                var _a, _b;
                return IdlCoder.fieldLayout(arg, Array.from([...((_a = idl.accounts) !== null && _a !== void 0 ? _a : []), ...((_b = idl.types) !== null && _b !== void 0 ? _b : [])]));
            });
            const name = camelCase(ix.name);
            return [name, borsh.struct(fieldLayouts, name)];
        }));
        return new Map(ixLayouts);
    }
    encode(ixName, ix) {
        const buffer = Buffer.alloc(1000); // TODO: use a tighter buffer.
        const methodName = camelCase(ixName);
        const layout = this.ixLayout.get(methodName);
        if (!layout) {
            throw new Error(`Unknown method: ${methodName}`);
        }
        const len = layout.encode(ix, buffer);
        const data = buffer.slice(0, len);
        return encodeWormholeInstructionData(WormholeInstruction[upperFirst(methodName)], data);
    }
    encodeState(_ixName, _ix) {
        throw new Error('Wormhole program does not have state');
    }
    decode(ix, _encoding = 'hex') {
        var _a;
        if (typeof ix === 'string') {
            ix =
                _encoding === 'hex' ? Buffer.from(ix, 'hex') : encoding.b58.decode(ix);
        }
        let discriminator = Buffer.from(ix.slice(0, 1)).readInt8();
        let data = Buffer.from(ix.slice(1));
        let name = camelCase(WormholeInstruction[discriminator]);
        let layout = this.ixLayout.get(name);
        if (!layout) {
            return null;
        }
        return { data: (_a = this.ixLayout.get(name)) === null || _a === void 0 ? void 0 : _a.decode(data), name };
    }
}
/** Solitaire enum of existing the Core Bridge's instructions.
 *
 * https://github.com/certusone/wormhole/blob/main/solana/bridge/program/src/lib.rs#L92
 */
export var WormholeInstruction;
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
})(WormholeInstruction || (WormholeInstruction = {}));
function encodeWormholeInstructionData(discriminator, data) {
    const instructionData = Buffer.alloc(1 + (data === undefined ? 0 : data.length));
    instructionData.writeUInt8(discriminator, 0);
    if (data !== undefined) {
        instructionData.write(data.toString('hex'), 1, 'hex');
    }
    return instructionData;
}
