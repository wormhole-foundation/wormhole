import { WormholeAccountsCoder } from './accounts';
import { WormholeEventsCoder } from './events';
import { WormholeInstructionCoder } from './instruction';
import { WormholeStateCoder } from './state';
import { WormholeTypesCoder } from './types';
export { WormholeInstruction } from './instruction';
export class WormholeCoder {
    constructor(idl) {
        this.instruction = new WormholeInstructionCoder(idl);
        this.accounts = new WormholeAccountsCoder(idl);
        this.state = new WormholeStateCoder(idl);
        this.events = new WormholeEventsCoder(idl);
        this.types = new WormholeTypesCoder(idl);
    }
}
