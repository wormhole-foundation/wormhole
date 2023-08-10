import { BN } from "@coral-xyz/anchor";

export * from "./native";

export type LegacyTransferTokensArgs = {
  nonce: number;
  amount: BN;
  relayerFee: BN;
  recipient: number[];
  recipientChain: number;
};
