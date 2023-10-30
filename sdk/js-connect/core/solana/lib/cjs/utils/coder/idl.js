"use strict";
// Borrowed from coral-xyz/anchor
//
// https://github.com/coral-xyz/anchor/blob/master/ts/packages/anchor/src/coder/borsh/idl.ts
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
exports.IdlCoder = void 0;
const borsh = __importStar(require("@coral-xyz/borsh"));
const lodash_1 = require("lodash");
class IdlCoder {
    static fieldLayout(field, types) {
        const fieldName = field.name !== undefined ? (0, lodash_1.camelCase)(field.name) : undefined;
        switch (field.type) {
            case 'bool': {
                return borsh.bool(fieldName);
            }
            case 'u8': {
                return borsh.u8(fieldName);
            }
            case 'i8': {
                return borsh.i8(fieldName);
            }
            case 'u16': {
                return borsh.u16(fieldName);
            }
            case 'i16': {
                return borsh.i16(fieldName);
            }
            case 'u32': {
                return borsh.u32(fieldName);
            }
            case 'i32': {
                return borsh.i32(fieldName);
            }
            case 'f32': {
                return borsh.f32(fieldName);
            }
            case 'u64': {
                return borsh.u64(fieldName);
            }
            case 'i64': {
                return borsh.i64(fieldName);
            }
            case 'f64': {
                return borsh.f64(fieldName);
            }
            case 'u128': {
                return borsh.u128(fieldName);
            }
            case 'i128': {
                return borsh.i128(fieldName);
            }
            case 'bytes': {
                return borsh.vecU8(fieldName);
            }
            case 'string': {
                return borsh.str(fieldName);
            }
            case 'publicKey': {
                return borsh.publicKey(fieldName);
            }
            default: {
                if ('vec' in field.type) {
                    return borsh.vec(IdlCoder.fieldLayout({
                        name: undefined,
                        type: field.type.vec,
                    }, types), fieldName);
                }
                else if ('option' in field.type) {
                    return borsh.option(IdlCoder.fieldLayout({
                        name: undefined,
                        type: field.type.option,
                    }, types), fieldName);
                }
                else if ('array' in field.type) {
                    let arrayTy = field.type.array[0];
                    let arrayLen = field.type.array[1];
                    let innerLayout = IdlCoder.fieldLayout({
                        name: undefined,
                        type: arrayTy,
                    }, types);
                    return borsh.array(innerLayout, arrayLen, fieldName);
                }
                else {
                    throw new Error(`Not yet implemented: ${field}`);
                }
            }
        }
    }
}
exports.IdlCoder = IdlCoder;
