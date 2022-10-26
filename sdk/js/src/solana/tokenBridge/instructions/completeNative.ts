import {
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  SYSVAR_RENT_PUBKEY,
  TransactionInstruction,
} from "@solana/web3.js";
import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import { createReadOnlyTokenBridgeProgramInterface } from "../program";
import { deriveClaimKey, derivePostedVaaKey } from "../../wormhole";
import {
  deriveEndpointKey,
  deriveTokenBridgeConfigKey,
  deriveCustodyKey,
  deriveCustodySignerKey,
} from "../accounts";
import {
  isBytes,
  ParsedTokenTransferVaa,
  parseTokenTransferVaa,
  SignedVaa,
} from "../../../vaa";

export function createCompleteTransferNativeInstruction(
  tokenBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedTokenTransferVaa,
  feeRecipient?: PublicKeyInitData
): TransactionInstruction {
  const methods =
    createReadOnlyTokenBridgeProgramInterface(
      tokenBridgeProgramId
    ).methods.completeNative();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getCompleteTransferNativeAccounts(
      tokenBridgeProgramId,
      wormholeProgramId,
      payer,
      vaa,
      feeRecipient
    ) as any,
    signers: undefined,
    remainingAccounts: undefined,
    preInstructions: undefined,
    postInstructions: undefined,
  });
}

export interface CompleteTransferNativeAccounts {
  payer: PublicKey;
  config: PublicKey;
  vaa: PublicKey;
  claim: PublicKey;
  endpoint: PublicKey;
  to: PublicKey;
  toFees: PublicKey;
  custody: PublicKey;
  mint: PublicKey;
  custodySigner: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
  tokenProgram: PublicKey;
  wormholeProgram: PublicKey;
}

export function getCompleteTransferNativeAccounts(
  tokenBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedTokenTransferVaa,
  feeRecipient?: PublicKeyInitData
): CompleteTransferNativeAccounts {
  const parsed = isBytes(vaa) ? parseTokenTransferVaa(vaa) : vaa;
  const mint = new PublicKey(parsed.tokenAddress);
  return {
    payer: new PublicKey(payer),
    config: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
    vaa: derivePostedVaaKey(wormholeProgramId, parsed.hash),
    claim: deriveClaimKey(
      tokenBridgeProgramId,
      parsed.emitterAddress,
      parsed.emitterChain,
      parsed.sequence
    ),
    endpoint: deriveEndpointKey(
      tokenBridgeProgramId,
      parsed.emitterChain,
      parsed.emitterAddress
    ),
    to: new PublicKey(parsed.to),
    toFees: new PublicKey(
      feeRecipient === undefined ? parsed.to : feeRecipient
    ),
    custody: deriveCustodyKey(tokenBridgeProgramId, mint),
    mint,
    custodySigner: deriveCustodySignerKey(tokenBridgeProgramId),
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
    tokenProgram: TOKEN_PROGRAM_ID,
    wormholeProgram: new PublicKey(wormholeProgramId),
  };
}
