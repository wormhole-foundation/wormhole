import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress } from "../../utils/account";

export interface EmitterAccounts {
  emitter: PublicKey;
  sequence: PublicKey;
}

export function emitterKey(emitterDeriverId: PublicKeyInitData): PublicKey {
  return deriveAddress([Buffer.from("emitter")], emitterDeriverId);
}

export function emitterSequenceKey(emitter: PublicKeyInitData, wormholeProgramId: PublicKeyInitData): PublicKey {
  return deriveAddress([Buffer.from("Sequence"), new PublicKey(emitter).toBytes()], wormholeProgramId);
}

export function getEmitterKeys(
  emitterDeriverId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData
): EmitterAccounts {
  const emitter = emitterKey(emitterDeriverId);
  return {
    emitter,
    sequence: emitterSequenceKey(emitter, wormholeProgramId),
  };
}
