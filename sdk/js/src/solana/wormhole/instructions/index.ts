import {
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { feeCollectorKey, getBridgeInfo } from "../accounts";

export * from "./governance";
export * from "./postMessage";
export * from "./postVaa";
export * from "./verifySignature";

export async function createBridgeFeeTransferInstruction(
  connection: Connection,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  commitment?: Commitment
): Promise<TransactionInstruction> {
  const fee = await getBridgeInfo(connection, wormholeProgramId, commitment).then((data) => data.config.fee);
  return SystemProgram.transfer({
    fromPubkey: new PublicKey(payer),
    toPubkey: feeCollectorKey(wormholeProgramId),
    lamports: fee,
  });
}
