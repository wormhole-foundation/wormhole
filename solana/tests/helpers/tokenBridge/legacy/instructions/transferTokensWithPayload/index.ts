import { BN } from "@coral-xyz/anchor";
import { PublicKey } from "@solana/web3.js";

export * from "./native";
export * from "./wrapped";

export type LegacyTransferTokensWithPayloadArgs = {
  nonce: number;
  amount: BN;
  redeemer: number[];
  redeemerChain: number;
  payload: Buffer;
  cpiProgramId: PublicKey | null;
};
