import { PublicKey } from "@solana/web3.js";

export * from "./Claim";
export * from "./Config";
export * from "./EmitterSequence";
export * from "./GuardianSet";
export * from "./PostedMessageV1";
export * from "./PostedMessageV1Unreliable";
export * from "./PostedVaaV1";
export * from "./SignatureSet";

export function upgradeAuthorityPda(programId: PublicKey): PublicKey {
  return PublicKey.findProgramAddressSync([Buffer.from("upgrade")], programId)[0];
}

export function feeCollectorPda(programId: PublicKey): PublicKey {
  return PublicKey.findProgramAddressSync([Buffer.from("fee_collector")], programId)[0];
}
