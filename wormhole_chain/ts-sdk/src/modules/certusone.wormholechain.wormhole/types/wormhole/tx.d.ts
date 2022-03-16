//@ts-nocheck
import { Reader, Writer } from "protobufjs/minimal";
import { GuardianKey } from "../wormhole/guardian_key";
export declare const protobufPackage = "certusone.wormholechain.wormhole";
export interface MsgExecuteGovernanceVAA {
    vaa: Uint8Array;
    signer: string;
}
export interface MsgExecuteGovernanceVAAResponse {
}
export interface MsgRegisterAccountAsGuardian {
    signer: string;
    guardianPubkey: GuardianKey | undefined;
    addressBech32: string;
    signature: Uint8Array;
}
export interface MsgRegisterAccountAsGuardianResponse {
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
export declare const MsgRegisterAccountAsGuardian: {
    encode(message: MsgRegisterAccountAsGuardian, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgRegisterAccountAsGuardian;
    fromJSON(object: any): MsgRegisterAccountAsGuardian;
    toJSON(message: MsgRegisterAccountAsGuardian): unknown;
    fromPartial(object: DeepPartial<MsgRegisterAccountAsGuardian>): MsgRegisterAccountAsGuardian;
};
export declare const MsgRegisterAccountAsGuardianResponse: {
    encode(_: MsgRegisterAccountAsGuardianResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): MsgRegisterAccountAsGuardianResponse;
    fromJSON(_: any): MsgRegisterAccountAsGuardianResponse;
    toJSON(_: MsgRegisterAccountAsGuardianResponse): unknown;
    fromPartial(_: DeepPartial<MsgRegisterAccountAsGuardianResponse>): MsgRegisterAccountAsGuardianResponse;
};
/** Msg defines the Msg service. */
export interface Msg {
    ExecuteGovernanceVAA(request: MsgExecuteGovernanceVAA): Promise<MsgExecuteGovernanceVAAResponse>;
    /** this line is used by starport scaffolding # proto/tx/rpc */
    RegisterAccountAsGuardian(request: MsgRegisterAccountAsGuardian): Promise<MsgRegisterAccountAsGuardianResponse>;
}
export declare class MsgClientImpl implements Msg {
    private readonly rpc;
    constructor(rpc: Rpc);
    ExecuteGovernanceVAA(request: MsgExecuteGovernanceVAA): Promise<MsgExecuteGovernanceVAAResponse>;
    RegisterAccountAsGuardian(request: MsgRegisterAccountAsGuardian): Promise<MsgRegisterAccountAsGuardianResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
