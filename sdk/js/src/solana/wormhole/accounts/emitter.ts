import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress } from "../../utils";

export interface EmitterAccounts {
  emitter: PublicKey;
  sequence: PublicKey;
}

export function deriveWormholeEmitterKey(
  emitterDeriverId: PublicKeyInitData
): PublicKey {
  return deriveAddress([Buffer.from("emitter")], emitterDeriverId);
}

export function deriveEmitterSequenceKey(
  emitter: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData
): PublicKey {
  return deriveAddress(
    [Buffer.from("Sequence"), new PublicKey(emitter).toBytes()],
    wormholeProgramId
  );
}

export function getEmitterKeys(
  emitterDeriverId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData
): EmitterAccounts {
  const emitter = deriveWormholeEmitterKey(emitterDeriverId);
  return {
    emitter,
    sequence: deriveEmitterSequenceKey(emitter, wormholeProgramId),
  };
}
