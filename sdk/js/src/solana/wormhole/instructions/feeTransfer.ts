import {
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { deriveFeeCollectorKey, getWormholeBridgeData } from "../accounts";

export async function createBridgeFeeTransferInstruction(
  connection: Connection,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  commitment?: Commitment
): Promise<TransactionInstruction> {
  const fee = await getWormholeBridgeData(
    connection,
    wormholeProgramId,
    commitment
  ).then((data) => data.config.fee);
  return SystemProgram.transfer({
    fromPubkey: new PublicKey(payer),
    toPubkey: deriveFeeCollectorKey(wormholeProgramId),
    lamports: fee,
  });
}
