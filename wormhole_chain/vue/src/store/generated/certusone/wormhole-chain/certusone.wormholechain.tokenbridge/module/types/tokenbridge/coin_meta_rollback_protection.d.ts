import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.tokenbridge";
export interface CoinMetaRollbackProtection {
    index: string;
}
export declare const CoinMetaRollbackProtection: {
    encode(message: CoinMetaRollbackProtection, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): CoinMetaRollbackProtection;
    fromJSON(object: any): CoinMetaRollbackProtection;
    toJSON(message: CoinMetaRollbackProtection): unknown;
    fromPartial(object: DeepPartial<CoinMetaRollbackProtection>): CoinMetaRollbackProtection;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
