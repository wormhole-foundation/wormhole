import { EventCoder, Event, Idl } from '@project-serum/anchor';
import { anchor } from '@wormhole-foundation/connect-sdk-solana';

export class TokenBridgeEventsCoder implements EventCoder {
  constructor(_idl: Idl) {}

  decode<
    E extends anchor.IdlEvent = anchor.IdlEvent,
    T = Record<string, string>,
  >(_log: string): Event<E, T> | null {
    throw new Error('Token Bridge program does not have events');
  }
}
