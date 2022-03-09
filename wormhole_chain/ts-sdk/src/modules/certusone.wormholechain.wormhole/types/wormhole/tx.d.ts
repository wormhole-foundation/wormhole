//@ts-nocheck
import { Reader, Writer } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.wormhole";
export interface MsgExecuteGovernanceVAA {
    vaa: Uint8Array;
    signer: string;
}
export interface MsgExecuteGovernanceVAAResponse {
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
/** Msg defines the Msg service. */
export interface Msg {
    /** this line is used by starport scaffolding # proto/tx/rpc */
    ExecuteGovernanceVAA(request: MsgExecuteGovernanceVAA): Promise<MsgExecuteGovernanceVAAResponse>;
}
export declare class MsgClientImpl implements Msg {
    private readonly rpc;
    constructor(rpc: Rpc);
    ExecuteGovernanceVAA(request: MsgExecuteGovernanceVAA): Promise<MsgExecuteGovernanceVAAResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
