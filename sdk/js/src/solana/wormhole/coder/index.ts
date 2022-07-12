import { Coder, Idl } from "@project-serum/anchor";
import { WormholeAccountsCoder } from "./accounts";
import { WormholeEventsCoder } from "./events";
import { WormholeInstructionCoder } from "./instruction";
import { WormholeStateCoder } from "./state";
import { WormholeTypesCoder } from "./types";

export { WormholeInstruction } from "./instruction";

export class WormholeCoder implements Coder {
  readonly instruction: WormholeInstructionCoder;
  readonly accounts: WormholeAccountsCoder;
  readonly state: WormholeStateCoder;
  readonly events: WormholeEventsCoder;
  readonly types: WormholeTypesCoder;

  constructor(idl: Idl) {
    this.instruction = new WormholeInstructionCoder(idl);
    this.accounts = new WormholeAccountsCoder(idl);
    this.state = new WormholeStateCoder(idl);
    this.events = new WormholeEventsCoder(idl);
    this.types = new WormholeTypesCoder(idl);
  }
}
