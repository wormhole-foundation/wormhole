import { Coder, Idl } from '@project-serum/anchor';
import { WormholeAccountsCoder } from './accounts';
import { WormholeEventsCoder } from './events';
import { WormholeInstructionCoder } from './instruction';
import { WormholeStateCoder } from './state';
import { WormholeTypesCoder } from './types';
export { WormholeInstruction } from './instruction';
export declare class WormholeCoder implements Coder {
    readonly instruction: WormholeInstructionCoder;
    readonly accounts: WormholeAccountsCoder;
    readonly state: WormholeStateCoder;
    readonly events: WormholeEventsCoder;
    readonly types: WormholeTypesCoder;
    constructor(idl: Idl);
}
