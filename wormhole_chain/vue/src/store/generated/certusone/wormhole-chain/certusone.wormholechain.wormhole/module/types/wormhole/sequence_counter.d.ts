import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.wormhole";
export interface SequenceCounter {
    index: string;
    sequence: number;
}
export declare const SequenceCounter: {
    encode(message: SequenceCounter, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): SequenceCounter;
    fromJSON(object: any): SequenceCounter;
    toJSON(message: SequenceCounter): unknown;
    fromPartial(object: DeepPartial<SequenceCounter>): SequenceCounter;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
