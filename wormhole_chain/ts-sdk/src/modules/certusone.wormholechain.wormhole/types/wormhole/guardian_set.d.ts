//@ts-nocheck
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.wormhole";
export interface GuardianSet {
    index: number;
    keys: Uint8Array[];
    expirationTime: number;
}
export declare const GuardianSet: {
    encode(message: GuardianSet, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): GuardianSet;
    fromJSON(object: any): GuardianSet;
    toJSON(message: GuardianSet): unknown;
    fromPartial(object: DeepPartial<GuardianSet>): GuardianSet;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
