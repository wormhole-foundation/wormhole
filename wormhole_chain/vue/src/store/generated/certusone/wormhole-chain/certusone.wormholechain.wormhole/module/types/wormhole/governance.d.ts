import { GuardianSet } from "../wormhole/guardian_set";
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.wormhole";
/** GuardianSetUpdateProposal defines a guardian set update governance proposal */
export interface GuardianSetUpdateProposal {
    title: string;
    description: string;
    newGuardianSet: GuardianSet | undefined;
}
/**
 * GovernanceWormholeMessageProposal defines a governance proposal to emit a generic message in the governance message
 * format.
 */
export interface GovernanceWormholeMessageProposal {
    title: string;
    description: string;
    module: Uint8Array;
    targetChain: number;
    payload: Uint8Array;
}
export declare const GuardianSetUpdateProposal: {
    encode(message: GuardianSetUpdateProposal, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): GuardianSetUpdateProposal;
    fromJSON(object: any): GuardianSetUpdateProposal;
    toJSON(message: GuardianSetUpdateProposal): unknown;
    fromPartial(object: DeepPartial<GuardianSetUpdateProposal>): GuardianSetUpdateProposal;
};
export declare const GovernanceWormholeMessageProposal: {
    encode(message: GovernanceWormholeMessageProposal, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): GovernanceWormholeMessageProposal;
    fromJSON(object: any): GovernanceWormholeMessageProposal;
    toJSON(message: GovernanceWormholeMessageProposal): unknown;
    fromPartial(object: DeepPartial<GovernanceWormholeMessageProposal>): GovernanceWormholeMessageProposal;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
