//@ts-nocheck
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.wormhole";
export interface GuardianValidator {
    guardianKey: Uint8Array;
    validatorAddr: Uint8Array;
}
export declare const GuardianValidator: {
    encode(message: GuardianValidator, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): GuardianValidator;
    fromJSON(object: any): GuardianValidator;
    toJSON(message: GuardianValidator): unknown;
    fromPartial(object: DeepPartial<GuardianValidator>): GuardianValidator;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
