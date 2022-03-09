//@ts-nocheck
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.tokenbridge";
export interface ChainRegistration {
    chainID: number;
    emitterAddress: Uint8Array;
}
export declare const ChainRegistration: {
    encode(message: ChainRegistration, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): ChainRegistration;
    fromJSON(object: any): ChainRegistration;
    toJSON(message: ChainRegistration): unknown;
    fromPartial(object: DeepPartial<ChainRegistration>): ChainRegistration;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
