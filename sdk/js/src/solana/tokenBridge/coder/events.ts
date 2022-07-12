import { EventCoder, Event, Idl } from "@project-serum/anchor";
import { IdlEvent } from "../../anchor";

export class TokenBridgeEventsCoder implements EventCoder {
  constructor(_idl: Idl) {}

  decode<E extends IdlEvent = IdlEvent, T = Record<string, string>>(
    _log: string
  ): Event<E, T> | null {
    throw new Error("Token Bridge program does not have events");
  }
}
