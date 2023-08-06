import { PublicKey } from "@solana/web3.js";
import { ProgramId, TOKEN_BRIDGE_PROGRAM_ID } from "../consts";

export function getProgramPubkey(programId?: ProgramId): PublicKey {
  return programId === undefined
    ? TOKEN_BRIDGE_PROGRAM_ID
    : new PublicKey(programId);
}

export function upgradeAuthority(programId: ProgramId): PublicKey {
  return PublicKey.findProgramAddressSync(
    [Buffer.from("upgrade")],
    getProgramPubkey(programId)
  )[0];
}
