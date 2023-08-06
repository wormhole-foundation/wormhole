import {
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
  SystemProgram,
} from "@solana/web3.js";
import { CORE_BRIDGE_PROGRAM_ID, ProgramId } from "../consts";
import { BridgeProgramData, FeeCollector } from "../state";

export function getProgramPubkey(programId?: ProgramId): PublicKey {
  return programId === undefined
    ? CORE_BRIDGE_PROGRAM_ID
    : new PublicKey(programId);
}

export function upgradeAuthority(programId: ProgramId): PublicKey {
  return PublicKey.findProgramAddressSync(
    [Buffer.from("upgrade")],
    getProgramPubkey(programId)
  )[0];
}

export async function transferMessageFeeIx(
  connection: Connection,
  programId: ProgramId,
  payer: PublicKey,
  commitmentOrConfig?: Commitment | GetAccountInfoConfig
) {
  const addr = BridgeProgramData.address(programId);
  return BridgeProgramData.fromAccountAddress(
    connection,
    addr,
    commitmentOrConfig
  ).then((bridgeProgramData) =>
    SystemProgram.transfer({
      fromPubkey: payer,
      toPubkey: FeeCollector.address(programId),
      lamports: BigInt(bridgeProgramData.config.feeLamports.toString()),
    })
  );
}
