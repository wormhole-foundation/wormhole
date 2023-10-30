import { TokenBridgeAccountsCoder } from './accounts';
import { TokenBridgeEventsCoder } from './events';
import { TokenBridgeInstructionCoder } from './instruction';
import { TokenBridgeStateCoder } from './state';
import { TokenBridgeTypesCoder } from './types';
export { TokenBridgeInstruction } from './instruction';
export class TokenBridgeCoder {
    constructor(idl) {
        this.instruction = new TokenBridgeInstructionCoder(idl);
        this.accounts = new TokenBridgeAccountsCoder(idl);
        this.state = new TokenBridgeStateCoder(idl);
        this.events = new TokenBridgeEventsCoder(idl);
        this.types = new TokenBridgeTypesCoder(idl);
    }
}
