import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress } from "../../utils";

export interface EmitterAccounts {
  emitter: PublicKey;
  sequence: PublicKey;
}

export function deriveWormholeEmitterKey(
  emitterProgramId: PublicKeyInitData
): PublicKey {
  return deriveAddress([Buffer.from("emitter")], emitterProgramId);
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
  emitterProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData
): EmitterAccounts {
  const emitter = deriveWormholeEmitterKey(emitterProgramId);
  return {
    emitter,
    sequence: deriveEmitterSequenceKey(emitter, wormholeProgramId),
  };
}
