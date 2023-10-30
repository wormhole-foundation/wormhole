import { Coder, Idl } from '@project-serum/anchor';
import { TokenBridgeAccountsCoder } from './accounts';
import { TokenBridgeEventsCoder } from './events';
import { TokenBridgeInstructionCoder } from './instruction';
import { TokenBridgeStateCoder } from './state';
import { TokenBridgeTypesCoder } from './types';
export { TokenBridgeInstruction } from './instruction';
export declare class TokenBridgeCoder implements Coder {
    readonly instruction: TokenBridgeInstructionCoder;
    readonly accounts: TokenBridgeAccountsCoder;
    readonly state: TokenBridgeStateCoder;
    readonly events: TokenBridgeEventsCoder;
    readonly types: TokenBridgeTypesCoder;
    constructor(idl: Idl);
}
