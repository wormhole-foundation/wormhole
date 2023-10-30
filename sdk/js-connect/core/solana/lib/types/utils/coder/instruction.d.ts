/// <reference types="node" />
import { Idl, Instruction, InstructionCoder } from '@project-serum/anchor';
export declare class WormholeInstructionCoder implements InstructionCoder {
    private ixLayout;
    constructor(idl: Idl);
    private static parseIxLayout;
    encode(ixName: string, ix: any): Buffer;
    encodeState(_ixName: string, _ix: any): Buffer;
    decode(ix: Buffer | Uint8Array, _encoding?: 'hex' | 'base58'): Instruction | null;
}
/** Solitaire enum of existing the Core Bridge's instructions.
 *
 * https://github.com/certusone/wormhole/blob/main/solana/bridge/program/src/lib.rs#L92
 */
export declare enum WormholeInstruction {
    Initialize = 0,
    PostMessage = 1,
    PostVaa = 2,
    SetFees = 3,
    TransferFees = 4,
    UpgradeContract = 5,
    UpgradeGuardianSet = 6,
    VerifySignatures = 7,
    PostMessageUnreliable = 8
}
