import {
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  SYSVAR_RENT_PUBKEY,
  TransactionInstruction,
} from "@solana/web3.js";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { createReadOnlyNftBridgeProgramInterface } from "../program";
import { deriveClaimKey, derivePostedVaaKey } from "../../wormhole";
import {
  deriveEndpointKey,
  deriveNftBridgeConfigKey,
  deriveWrappedMintKey,
  deriveWrappedMetaKey,
  deriveMintAuthorityKey,
} from "../accounts";
import {
  isBytes,
  ParsedNftTransferVaa,
  parseNftTransferVaa,
  SignedVaa,
} from "../../../vaa";
import { SplTokenMetadataProgram } from "../../utils";

export function createCompleteTransferWrappedInstruction(
  nftBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedNftTransferVaa,
  toAuthority?: PublicKeyInitData
): TransactionInstruction {
  const methods =
    createReadOnlyNftBridgeProgramInterface(
      nftBridgeProgramId
    ).methods.completeWrapped();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getCompleteTransferWrappedAccounts(
      nftBridgeProgramId,
      wormholeProgramId,
      payer,
      vaa,
      toAuthority
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
  toAuthority: PublicKey;
  mint: PublicKey;
  wrappedMeta: PublicKey;
  mintAuthority: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
  tokenProgram: PublicKey;
  splMetadataProgram: PublicKey;
  associatedTokenProgram: PublicKey;
  wormholeProgram: PublicKey;
}

export function getCompleteTransferWrappedAccounts(
  nftBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedNftTransferVaa,
  toAuthority?: PublicKeyInitData
): CompleteTransferWrappedAccounts {
  const parsed = isBytes(vaa) ? parseNftTransferVaa(vaa) : vaa;
  const mint = deriveWrappedMintKey(
    nftBridgeProgramId,
    parsed.tokenChain,
    parsed.tokenAddress,
    parsed.tokenId
  );
  return {
    payer: new PublicKey(payer),
    config: deriveNftBridgeConfigKey(nftBridgeProgramId),
    vaa: derivePostedVaaKey(wormholeProgramId, parsed.hash),
    claim: deriveClaimKey(
      nftBridgeProgramId,
      parsed.emitterAddress,
      parsed.emitterChain,
      parsed.sequence
    ),
    endpoint: deriveEndpointKey(
      nftBridgeProgramId,
      parsed.emitterChain,
      parsed.emitterAddress
    ),
    to: new PublicKey(parsed.to),
    toAuthority: new PublicKey(toAuthority === undefined ? payer : toAuthority),
    mint,
    wrappedMeta: deriveWrappedMetaKey(nftBridgeProgramId, mint),
    mintAuthority: deriveMintAuthorityKey(nftBridgeProgramId),
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
    tokenProgram: TOKEN_PROGRAM_ID,
    splMetadataProgram: SplTokenMetadataProgram.programId,
    associatedTokenProgram: ASSOCIATED_TOKEN_PROGRAM_ID,
    wormholeProgram: new PublicKey(wormholeProgramId),
  };
}
