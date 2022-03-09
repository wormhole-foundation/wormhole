//@ts-nocheck
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.tokenbridge";
export interface EventChainRegistered {
    chainID: number;
    emitterAddress: Uint8Array;
}
export interface EventAssetRegistrationUpdate {
    tokenChain: number;
    tokenAddress: Uint8Array;
    name: string;
    symbol: string;
    decimals: number;
}
export interface EventTransferReceived {
    tokenChain: number;
    tokenAddress: Uint8Array;
    to: string;
    feeRecipient: string;
    amount: string;
    fee: string;
    localDenom: string;
}
export declare const EventChainRegistered: {
    encode(message: EventChainRegistered, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): EventChainRegistered;
    fromJSON(object: any): EventChainRegistered;
    toJSON(message: EventChainRegistered): unknown;
    fromPartial(object: DeepPartial<EventChainRegistered>): EventChainRegistered;
};
export declare const EventAssetRegistrationUpdate: {
    encode(message: EventAssetRegistrationUpdate, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): EventAssetRegistrationUpdate;
    fromJSON(object: any): EventAssetRegistrationUpdate;
    toJSON(message: EventAssetRegistrationUpdate): unknown;
    fromPartial(object: DeepPartial<EventAssetRegistrationUpdate>): EventAssetRegistrationUpdate;
};
export declare const EventTransferReceived: {
    encode(message: EventTransferReceived, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): EventTransferReceived;
    fromJSON(object: any): EventTransferReceived;
    toJSON(message: EventTransferReceived): unknown;
    fromPartial(object: DeepPartial<EventTransferReceived>): EventTransferReceived;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
