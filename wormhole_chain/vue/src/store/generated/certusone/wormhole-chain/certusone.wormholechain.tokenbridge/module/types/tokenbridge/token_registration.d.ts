import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.tokenbridge";
export interface TokenRegistration {
    index: string;
}
export declare const TokenRegistration: {
    encode(message: TokenRegistration, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): TokenRegistration;
    fromJSON(object: any): TokenRegistration;
    toJSON(message: TokenRegistration): unknown;
    fromPartial(object: DeepPartial<TokenRegistration>): TokenRegistration;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
