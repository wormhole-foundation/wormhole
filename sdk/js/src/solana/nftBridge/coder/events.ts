import { EventCoder, Event, Idl } from "@project-serum/anchor";
import { IdlEvent } from "../../anchor";

export class NftBridgeEventsCoder implements EventCoder {
  constructor(_idl: Idl) {}

  decode<E extends IdlEvent = IdlEvent, T = Record<string, string>>(
    _log: string
  ): Event<E, T> | null {
    throw new Error("NFT Bridge program does not have events");
  }
}
