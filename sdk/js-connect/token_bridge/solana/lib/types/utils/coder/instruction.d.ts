/// <reference types="node" />
import { Idl, InstructionCoder } from '@project-serum/anchor';
export declare class TokenBridgeInstructionCoder implements InstructionCoder {
    constructor(_: Idl);
    encode(ixName: string, ix: any): Buffer;
    encodeState(_ixName: string, _ix: any): Buffer;
}
/** Solitaire enum of existing the Token Bridge's instructions.
 *
 * https://github.com/certusone/wormhole/blob/main/solana/modules/token_bridge/program/src/lib.rs#L100
 */
export declare enum TokenBridgeInstruction {
    Initialize = 0,
    AttestToken = 1,
    CompleteNative = 2,
    CompleteWrapped = 3,
    TransferWrapped = 4,
    TransferNative = 5,
    RegisterChain = 6,
    CreateWrapped = 7,
    UpgradeContract = 8,
    CompleteNativeWithPayload = 9,
    CompleteWrappedWithPayload = 10,
    TransferWrappedWithPayload = 11,
    TransferNativeWithPayload = 12
}
