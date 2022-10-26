import {
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  SYSVAR_RENT_PUBKEY,
  TransactionInstruction,
} from "@solana/web3.js";
import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import {
  createReadOnlyNftBridgeProgramInterface,
  tokenIdToMint,
} from "../program";
import { deriveClaimKey, derivePostedVaaKey } from "../../wormhole";
import {
  deriveEndpointKey,
  deriveNftBridgeConfigKey,
  deriveCustodyKey,
  deriveCustodySignerKey,
} from "../accounts";
import {
  isBytes,
  ParsedNftTransferVaa,
  parseNftTransferVaa,
  SignedVaa,
} from "../../../vaa";

export function createCompleteTransferNativeInstruction(
  nftBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedNftTransferVaa,
  toAuthority?: PublicKeyInitData
): TransactionInstruction {
  const methods =
    createReadOnlyNftBridgeProgramInterface(
      nftBridgeProgramId
    ).methods.completeNative();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getCompleteTransferNativeAccounts(
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

export interface CompleteTransferNativeAccounts {
  payer: PublicKey;
  config: PublicKey;
  vaa: PublicKey;
  claim: PublicKey;
  endpoint: PublicKey;
  to: PublicKey;
  toAuthority: PublicKey;
  custody: PublicKey;
  mint: PublicKey;
  custodySigner: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
  tokenProgram: PublicKey;
  wormholeProgram: PublicKey;
}

export function getCompleteTransferNativeAccounts(
  nftBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedNftTransferVaa,
  toAuthority?: PublicKeyInitData
): CompleteTransferNativeAccounts {
  const parsed = isBytes(vaa) ? parseNftTransferVaa(vaa) : vaa;
  // the mint key is encoded in the tokenId when it was transferred out
  const mint = tokenIdToMint(parsed.tokenId);
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
    custody: deriveCustodyKey(nftBridgeProgramId, mint),
    mint,
    custodySigner: deriveCustodySignerKey(nftBridgeProgramId),
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
    tokenProgram: TOKEN_PROGRAM_ID,
    wormholeProgram: new PublicKey(wormholeProgramId),
  };
}
