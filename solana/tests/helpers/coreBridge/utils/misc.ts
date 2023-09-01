import { Commitment, GetAccountInfoConfig, PublicKey, SystemProgram } from "@solana/web3.js";
import { Config, feeCollectorPda } from "../legacy/state";
import { CoreBridgeProgram } from "..";

export async function transferMessageFeeIx(
  program: CoreBridgeProgram,
  payer: PublicKey,
  commitmentOrConfig?: Commitment | GetAccountInfoConfig
) {
  const addr = Config.address(program.programId);
  return Config.fromAccountAddress(program.provider.connection, addr, commitmentOrConfig).then(
    (bridgeProgramData) =>
      SystemProgram.transfer({
        fromPubkey: payer,
        toPubkey: feeCollectorPda(program.programId),
        lamports: BigInt(bridgeProgramData.feeLamports.toString()),
      })
  );
}
