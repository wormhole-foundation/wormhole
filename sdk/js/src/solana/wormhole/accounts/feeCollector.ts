import {
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { deriveAddress } from "../../utils/account";
import { getBridgeInfo } from "./bridgeInfo";

export function feeCollectorKey(wormholeProgramId: PublicKeyInitData): PublicKey {
  return deriveAddress([Buffer.from("fee_collector")], wormholeProgramId);
}
