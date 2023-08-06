import { PublicKey } from "@solana/web3.js";

export class FeeCollector {
  static address(programId: PublicKey): PublicKey {
    return PublicKey.findProgramAddressSync(
      [Buffer.from("fee_collector")],
      programId
    )[0];
  }
}
