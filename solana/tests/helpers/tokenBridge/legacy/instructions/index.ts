import { BN } from "@coral-xyz/anchor";
import { createApproveInstruction } from "@solana/spl-token";
import { PublicKey, TransactionInstruction } from "@solana/web3.js";
import { TokenBridgeProgram } from "../..";
import { transferAuthorityPda } from "../state";

export * from "./attestToken";
export * from "./createOrUpdateWrapped";
export * from "./governance";
export * from "./initialize";
export * from "./transferTokens";
export * from "./transferTokensWithPayload";
export * from "./completeTransfer";
export * from "./completeTransferWithPayload";

export function approveTransferAuthorityIx(
  program: TokenBridgeProgram,
  token: PublicKey,
  owner: PublicKey,
  amount: BN
): TransactionInstruction {
  return createApproveInstruction(
    token,
    transferAuthorityPda(program.programId),
    owner,
    BigInt(amount.toString())
  );
}
