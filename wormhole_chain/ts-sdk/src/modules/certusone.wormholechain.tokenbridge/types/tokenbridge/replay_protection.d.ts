//@ts-nocheck
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.tokenbridge";
export interface ReplayProtection {
    index: string;
}
export declare const ReplayProtection: {
    encode(message: ReplayProtection, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): ReplayProtection;
    fromJSON(object: any): ReplayProtection;
    toJSON(message: ReplayProtection): unknown;
    fromPartial(object: DeepPartial<ReplayProtection>): ReplayProtection;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
