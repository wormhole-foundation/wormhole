//@ts-nocheck
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.tokenbridge";
export interface Config {
}
export declare const Config: {
    encode(_: Config, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): Config;
    fromJSON(_: any): Config;
    toJSON(_: Config): unknown;
    fromPartial(_: DeepPartial<Config>): Config;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
