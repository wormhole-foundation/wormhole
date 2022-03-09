//@ts-nocheck
import { Reader, Writer } from "protobufjs/minimal";
import { Coin } from "../cosmos/base/v1beta1/coin";
export declare const protobufPackage = "certusone.wormholechain.tokenbridge";
export interface MsgExecuteGovernanceVAA {
    creator: string;
    vaa: Uint8Array;
}
export interface MsgExecuteGovernanceVAAResponse {
}
export interface MsgExecuteVAA {
    creator: string;
    vaa: Uint8Array;
}
export interface MsgExecuteVAAResponse {
}
export interface MsgAttestToken {
    creator: string;
    denom: string;
}
export interface MsgAttestTokenResponse {
}
export interface MsgTransfer {
    creator: string;
    amount: Coin | undefined;
    toChain: number;
    toAddress: Uint8Array;
    fee: string;
}
export interface MsgTransferResponse {
}
export declare const MsgExecuteGovernanceVAA: {
    encode(message: MsgExecuteGovernanceVAA, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgExecuteGovernanceVAA;
    fromJSON(object: any): MsgExecuteGovernanceVAA;
    toJSON(message: MsgExecuteGovernanceVAA): unknown;
    fromPartial(object: DeepPartial<MsgExecuteGovernanceVAA>): MsgExecuteGovernanceVAA;
};
export declare const MsgExecuteGovernanceVAAResponse: {
    encode(_: MsgExecuteGovernanceVAAResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgExecuteGovernanceVAAResponse;
    fromJSON(_: any): MsgExecuteGovernanceVAAResponse;
    toJSON(_: MsgExecuteGovernanceVAAResponse): unknown;
    fromPartial(_: DeepPartial<MsgExecuteGovernanceVAAResponse>): MsgExecuteGovernanceVAAResponse;
};
export declare const MsgExecuteVAA: {
    encode(message: MsgExecuteVAA, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgExecuteVAA;
    fromJSON(object: any): MsgExecuteVAA;
    toJSON(message: MsgExecuteVAA): unknown;
    fromPartial(object: DeepPartial<MsgExecuteVAA>): MsgExecuteVAA;
};
export declare const MsgExecuteVAAResponse: {
    encode(_: MsgExecuteVAAResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgExecuteVAAResponse;
    fromJSON(_: any): MsgExecuteVAAResponse;
    toJSON(_: MsgExecuteVAAResponse): unknown;
    fromPartial(_: DeepPartial<MsgExecuteVAAResponse>): MsgExecuteVAAResponse;
};
export declare const MsgAttestToken: {
    encode(message: MsgAttestToken, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgAttestToken;
    fromJSON(object: any): MsgAttestToken;
    toJSON(message: MsgAttestToken): unknown;
    fromPartial(object: DeepPartial<MsgAttestToken>): MsgAttestToken;
};
export declare const MsgAttestTokenResponse: {
    encode(_: MsgAttestTokenResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgAttestTokenResponse;
    fromJSON(_: any): MsgAttestTokenResponse;
    toJSON(_: MsgAttestTokenResponse): unknown;
    fromPartial(_: DeepPartial<MsgAttestTokenResponse>): MsgAttestTokenResponse;
};
export declare const MsgTransfer: {
    encode(message: MsgTransfer, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgTransfer;
    fromJSON(object: any): MsgTransfer;
    toJSON(message: MsgTransfer): unknown;
    fromPartial(object: DeepPartial<MsgTransfer>): MsgTransfer;
};
export declare const MsgTransferResponse: {
    encode(_: MsgTransferResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgTransferResponse;
    fromJSON(_: any): MsgTransferResponse;
    toJSON(_: MsgTransferResponse): unknown;
    fromPartial(_: DeepPartial<MsgTransferResponse>): MsgTransferResponse;
};
/** Msg defines the Msg service. */
export interface Msg {
    ExecuteGovernanceVAA(request: MsgExecuteGovernanceVAA): Promise<MsgExecuteGovernanceVAAResponse>;
    ExecuteVAA(request: MsgExecuteVAA): Promise<MsgExecuteVAAResponse>;
    AttestToken(request: MsgAttestToken): Promise<MsgAttestTokenResponse>;
    /** this line is used by starport scaffolding # proto/tx/rpc */
    Transfer(request: MsgTransfer): Promise<MsgTransferResponse>;
}
export declare class MsgClientImpl implements Msg {
    private readonly rpc;
    constructor(rpc: Rpc);
    ExecuteGovernanceVAA(request: MsgExecuteGovernanceVAA): Promise<MsgExecuteGovernanceVAAResponse>;
    ExecuteVAA(request: MsgExecuteVAA): Promise<MsgExecuteVAAResponse>;
    AttestToken(request: MsgAttestToken): Promise<MsgAttestTokenResponse>;
    Transfer(request: MsgTransfer): Promise<MsgTransferResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
