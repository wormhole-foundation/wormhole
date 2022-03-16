//@ts-nocheck
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.wormhole";
export interface GuardianKey {
    key: Uint8Array;
}
export declare const GuardianKey: {
    encode(message: GuardianKey, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): GuardianKey;
    fromJSON(object: any): GuardianKey;
    toJSON(message: GuardianKey): unknown;
    fromPartial(object: DeepPartial<GuardianKey>): GuardianKey;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
