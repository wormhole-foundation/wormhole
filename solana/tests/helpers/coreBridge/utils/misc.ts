import { Commitment, GetAccountInfoConfig, PublicKey, SystemProgram } from "@solana/web3.js";
import { BridgeProgramData, FeeCollector } from "../legacy/state";
import { CoreBridgeProgram } from "..";

export async function transferMessageFeeIx(
  program: CoreBridgeProgram,
  payer: PublicKey,
  commitmentOrConfig?: Commitment | GetAccountInfoConfig
) {
  const addr = BridgeProgramData.address(program.programId);
  return BridgeProgramData.fromAccountAddress(
    program.provider.connection,
    addr,
    commitmentOrConfig
  ).then((bridgeProgramData) =>
    SystemProgram.transfer({
      fromPubkey: payer,
      toPubkey: FeeCollector.address(program.programId),
      lamports: BigInt(bridgeProgramData.config.feeLamports.toString()),
    })
  );
}
