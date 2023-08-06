import { PublicKey } from "@solana/web3.js";
import { ProgramId } from "../consts";
import { getProgramPubkey } from "../utils/misc";

export class FeeCollector {
  static address(programId: ProgramId): PublicKey {
    return PublicKey.findProgramAddressSync(
      [Buffer.from("fee_collector")],
      getProgramPubkey(programId)
    )[0];
  }
}
