import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import {
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  SYSVAR_RENT_PUBKEY,
  TransactionInstruction,
} from "@solana/web3.js";
import {
  isBytes,
  ParsedNftTransferVaa,
  parseNftTransferVaa,
  SignedVaa,
} from "../../../vaa";
import { deriveTokenMetadataKey, TOKEN_METADATA_PROGRAM_ID } from "../../utils";
import { derivePostedVaaKey } from "../../wormhole";
import {
  deriveEndpointKey,
  deriveMintAuthorityKey,
  deriveNftBridgeConfigKey,
  deriveWrappedMetaKey,
  deriveWrappedMintKey,
} from "../accounts";
import { createReadOnlyNftBridgeProgramInterface } from "../program";

export function createCompleteWrappedMetaInstruction(
  nftBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedNftTransferVaa
): TransactionInstruction {
  const methods =
    createReadOnlyNftBridgeProgramInterface(
      nftBridgeProgramId
    ).methods.completeWrappedMeta();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getCompleteWrappedMetaAccounts(
      nftBridgeProgramId,
      wormholeProgramId,
      payer,
      vaa
    ) as any,
    signers: undefined,
    remainingAccounts: undefined,
    preInstructions: undefined,
    postInstructions: undefined,
  });
}

export interface CompleteWrappedMetaAccounts {
  payer: PublicKey;
  config: PublicKey;
  vaa: PublicKey;
  endpoint: PublicKey;
  mint: PublicKey;
  wrappedMeta: PublicKey;
  splMetadata: PublicKey;
  mintAuthority: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
  tokenProgram: PublicKey;
  splMetadataProgram: PublicKey;
  wormholeProgram: PublicKey;
}

export function getCompleteWrappedMetaAccounts(
  nftBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedNftTransferVaa
): CompleteWrappedMetaAccounts {
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
    endpoint: deriveEndpointKey(
      nftBridgeProgramId,
      parsed.emitterChain,
      parsed.emitterAddress
    ),
    mint,
    wrappedMeta: deriveWrappedMetaKey(nftBridgeProgramId, mint),
    splMetadata: deriveTokenMetadataKey(mint),
    mintAuthority: deriveMintAuthorityKey(nftBridgeProgramId),
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
    tokenProgram: TOKEN_PROGRAM_ID,
    splMetadataProgram: TOKEN_METADATA_PROGRAM_ID,
    wormholeProgram: new PublicKey(wormholeProgramId),
  };
}
