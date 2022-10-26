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
  deriveWrappedMintKey,
  deriveWrappedMetaKey,
  deriveMintAuthorityKey,
} from "../accounts";
import {
  isBytes,
  ParsedTokenTransferVaa,
  parseTokenTransferVaa,
  SignedVaa,
} from "../../../vaa";

export function createCompleteTransferWrappedInstruction(
  tokenBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedTokenTransferVaa,
  feeRecipient?: PublicKeyInitData
): TransactionInstruction {
  const methods =
    createReadOnlyTokenBridgeProgramInterface(
      tokenBridgeProgramId
    ).methods.completeWrapped();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getCompleteTransferWrappedAccounts(
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

export interface CompleteTransferWrappedAccounts {
  payer: PublicKey;
  config: PublicKey;
  vaa: PublicKey;
  claim: PublicKey;
  endpoint: PublicKey;
  to: PublicKey;
  toFees: PublicKey;
  mint: PublicKey;
  wrappedMeta: PublicKey;
  mintAuthority: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
  tokenProgram: PublicKey;
  wormholeProgram: PublicKey;
}

export function getCompleteTransferWrappedAccounts(
  tokenBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedTokenTransferVaa,
  feeRecipient?: PublicKeyInitData
): CompleteTransferWrappedAccounts {
  const parsed = isBytes(vaa) ? parseTokenTransferVaa(vaa) : vaa;
  const mint = deriveWrappedMintKey(
    tokenBridgeProgramId,
    parsed.tokenChain,
    parsed.tokenAddress
  );
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
    mint,
    wrappedMeta: deriveWrappedMetaKey(tokenBridgeProgramId, mint),
    mintAuthority: deriveMintAuthorityKey(tokenBridgeProgramId),
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
    tokenProgram: TOKEN_PROGRAM_ID,
    wormholeProgram: new PublicKey(wormholeProgramId),
  };
}
