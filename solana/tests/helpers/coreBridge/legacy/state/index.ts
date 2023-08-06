import { PublicKey } from "@solana/web3.js";

export * from "./BridgeProgramData";
export * from "./Claim";
export * from "./EmitterSequence";
export * from "./FeeCollector";
export * from "./GuardianSet";
export * from "./PostedMessageV1";
export * from "./PostedMessageV1Unreliable";
export * from "./PostedVaaV1";
export * from "./SignatureSet";

export function upgradeAuthorityPda(programId: PublicKey): PublicKey {
  return PublicKey.findProgramAddressSync(
    [Buffer.from("upgrade")],
    programId
  )[0];
}
