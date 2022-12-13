import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import {
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  SYSVAR_RENT_PUBKEY,
} from "@solana/web3.js";
import {
  isBytes,
  ParsedTokenTransferVaa,
  parseTokenTransferVaa,
  SignedVaa,
} from "../../vaa";
import {
  deriveClaimKey,
  derivePostedVaaKey,
  getWormholeDerivedAccounts,
} from "../wormhole";
import {
  deriveAuthoritySignerKey,
  deriveCustodyKey,
  deriveCustodySignerKey,
  deriveEndpointKey,
  deriveMintAuthorityKey,
  deriveRedeemerAccountKey,
  deriveSenderAccountKey,
  deriveTokenBridgeConfigKey,
  deriveWrappedMetaKey,
  deriveWrappedMintKey,
} from "./accounts";
import {
  getTransferNativeWithPayloadAccounts,
  getTransferWrappedWithPayloadAccounts,
} from "./instructions";

export interface TokenBridgeBaseDerivedAccounts {
  /**
   * seeds = ["config"], seeds::program = tokenBridgeProgram
   */
  tokenBridgeConfig: PublicKey;
}

export interface TokenBridgeBaseNativeDerivedAccounts
  extends TokenBridgeBaseDerivedAccounts {
  /**
   * seeds = ["custody_signer"], seeds::program = tokenBridgeProgram
   */
  tokenBridgeCustodySigner: PublicKey;
}

export interface TokenBridgeBaseSenderDerivedAccounts
  extends TokenBridgeBaseDerivedAccounts {
  /**
   * seeds = ["authority_signer"], seeds::program = tokenBridgeProgram
   */
  tokenBridgeAuthoritySigner: PublicKey;
  /**
   * seeds = ["sender"], seeds::program = cpiProgramId
   */
  tokenBridgeSender: PublicKey;
  /**
   * seeds = ["Bridge"], seeds::program = wormholeProgram
   */
  wormholeBridge: PublicKey;
  /**
   * seeds = ["emitter"], seeds::program = tokenBridgeProgram
   */
  tokenBridgeEmitter: PublicKey;
  /**
   * seeds = ["Sequence", tokenBridgeEmitter], seeds::program = wormholeProgram
   */
  tokenBridgeSequence: PublicKey;
  /**
   * seeds = ["fee_collector"], seeds::program = wormholeProgram
   */
  wormholeFeeCollector: PublicKey;
}

export interface TokenBridgeNativeSenderDerivedAccounts
  extends TokenBridgeBaseNativeDerivedAccounts,
    TokenBridgeBaseSenderDerivedAccounts {}

export interface TokenBridgeWrappedSenderDerivedAccounts
  extends TokenBridgeBaseSenderDerivedAccounts {}

export interface TokenBridgeBaseRedeemerDerivedAccounts
  extends TokenBridgeBaseDerivedAccounts {
  /**
   * seeds = ["redeemer"], seeds::program = cpiProgramId
   */
  tokenBridgeRedeemer: PublicKey;
}

export interface TokenBridgeNativeRedeemerDerivedAccounts
  extends TokenBridgeBaseNativeDerivedAccounts,
    TokenBridgeBaseRedeemerDerivedAccounts {}

export interface TokenBridgeWrappedRedeemerDerivedAccounts
  extends TokenBridgeBaseRedeemerDerivedAccounts {
  /**
   * seeds = ["mint_signer"], seeds::program = tokenBridgeProgram
   */
  tokenBridgeMintAuthority: PublicKey;
}

export interface TokenBridgeDerivedAccounts
  extends TokenBridgeNativeSenderDerivedAccounts,
    TokenBridgeWrappedSenderDerivedAccounts,
    TokenBridgeNativeRedeemerDerivedAccounts,
    TokenBridgeWrappedRedeemerDerivedAccounts {}

/**
 * Generate Token Bridge PDAs.
 *
 * @param cpiProgramId
 * @param tokenBridgeProgramId
 * @param wormholeProgramId
 * @returns
 */
export function getTokenBridgeDerivedAccounts(
  cpiProgramId: PublicKeyInitData,
  tokenBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData
): TokenBridgeDerivedAccounts {
  const {
    wormholeEmitter: tokenBridgeEmitter,
    wormholeBridge,
    wormholeFeeCollector,
    wormholeSequence: tokenBridgeSequence,
  } = getWormholeDerivedAccounts(tokenBridgeProgramId, wormholeProgramId);
  return {
    tokenBridgeConfig: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
    tokenBridgeAuthoritySigner: deriveAuthoritySignerKey(tokenBridgeProgramId),
    tokenBridgeCustodySigner: deriveCustodySignerKey(tokenBridgeProgramId),
    tokenBridgeMintAuthority: deriveMintAuthorityKey(tokenBridgeProgramId),
    tokenBridgeSender: deriveSenderAccountKey(cpiProgramId),
    tokenBridgeRedeemer: deriveRedeemerAccountKey(cpiProgramId),
    wormholeBridge,
    tokenBridgeEmitter,
    wormholeFeeCollector,
    tokenBridgeSequence,
  };
}

export interface TransferNativeWithPayloadCpiAccounts
  extends TokenBridgeNativeSenderDerivedAccounts {
  payer: PublicKey;
  /**
   * seeds = [mint], seeds::program = tokenBridgeProgram
   */
  tokenBridgeCustody: PublicKey;
  /**
   * Token account where tokens reside
   */
  fromTokenAccount: PublicKey;
  mint: PublicKey;
  wormholeMessage: PublicKey;
  clock: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
  tokenProgram: PublicKey;
  wormholeProgram: PublicKey;
}

/**
 * Generate accounts needed to perform `transfer_wrapped_with_payload` instruction
 * as cross-program invocation.
 *
 * @param cpiProgramId
 * @param tokenBridgeProgramId
 * @param wormholeProgramId
 * @param payer
 * @param message
 * @param fromTokenAccount
 * @param mint
 * @returns
 */
export function getTransferNativeWithPayloadCpiAccounts(
  cpiProgramId: PublicKeyInitData,
  tokenBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  message: PublicKeyInitData,
  fromTokenAccount: PublicKeyInitData,
  mint: PublicKeyInitData
): TransferNativeWithPayloadCpiAccounts {
  const accounts = getTransferNativeWithPayloadAccounts(
    tokenBridgeProgramId,
    wormholeProgramId,
    payer,
    message,
    fromTokenAccount,
    mint,
    cpiProgramId
  );
  return {
    payer: accounts.payer,
    tokenBridgeConfig: accounts.config,
    fromTokenAccount: accounts.from,
    mint: accounts.mint,
    tokenBridgeCustody: accounts.custody,
    tokenBridgeAuthoritySigner: accounts.authoritySigner,
    tokenBridgeCustodySigner: accounts.custodySigner,
    wormholeBridge: accounts.wormholeBridge,
    wormholeMessage: accounts.wormholeMessage,
    tokenBridgeEmitter: accounts.wormholeEmitter,
    tokenBridgeSequence: accounts.wormholeSequence,
    wormholeFeeCollector: accounts.wormholeFeeCollector,
    clock: accounts.clock,
    tokenBridgeSender: accounts.sender,
    rent: accounts.rent,
    systemProgram: accounts.systemProgram,
    tokenProgram: accounts.tokenProgram,
    wormholeProgram: accounts.wormholeProgram,
  };
}

export interface TransferWrappedWithPayloadCpiAccounts
  extends TokenBridgeWrappedSenderDerivedAccounts {
  payer: PublicKey;
  /**
   * Token account where tokens reside
   */
  fromTokenAccount: PublicKey;
  /**
   * Token account owner (usually cpiProgramId)
   */
  fromTokenAccountOwner: PublicKey;
  /**
   * seeds = ["wrapped", token_chain, token_address], seeds::program = tokenBridgeProgram
   */
  tokenBridgeWrappedMint: PublicKey;
  /**
   * seeds = ["meta", mint], seeds::program = tokenBridgeProgram
   */
  tokenBridgeWrappedMeta: PublicKey;
  wormholeMessage: PublicKey;
  clock: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
  tokenProgram: PublicKey;
  wormholeProgram: PublicKey;
}

/**
 * Generate accounts needed to perform `transfer_wrapped_with_payload` instruction
 * as cross-program invocation.
 *
 * @param cpiProgramId
 * @param tokenBridgeProgramId
 * @param wormholeProgramId
 * @param payer
 * @param message
 * @param fromTokenAccount
 * @param tokenChain
 * @param tokenAddress
 * @param [fromTokenAccountOwner]
 * @returns
 */
export function getTransferWrappedWithPayloadCpiAccounts(
  cpiProgramId: PublicKeyInitData,
  tokenBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  message: PublicKeyInitData,
  fromTokenAccount: PublicKeyInitData,
  tokenChain: number,
  tokenAddress: Buffer | Uint8Array,
  fromTokenAccountOwner?: PublicKeyInitData
): TransferWrappedWithPayloadCpiAccounts {
  const accounts = getTransferWrappedWithPayloadAccounts(
    tokenBridgeProgramId,
    wormholeProgramId,
    payer,
    message,
    fromTokenAccount,
    fromTokenAccountOwner === undefined ? cpiProgramId : fromTokenAccountOwner,
    tokenChain,
    tokenAddress,
    cpiProgramId
  );
  return {
    payer: accounts.payer,
    tokenBridgeConfig: accounts.config,
    fromTokenAccount: accounts.from,
    fromTokenAccountOwner: accounts.fromOwner,
    tokenBridgeWrappedMint: accounts.mint,
    tokenBridgeWrappedMeta: accounts.wrappedMeta,
    tokenBridgeAuthoritySigner: accounts.authoritySigner,
    wormholeBridge: accounts.wormholeBridge,
    wormholeMessage: accounts.wormholeMessage,
    tokenBridgeEmitter: accounts.wormholeEmitter,
    tokenBridgeSequence: accounts.wormholeSequence,
    wormholeFeeCollector: accounts.wormholeFeeCollector,
    clock: accounts.clock,
    tokenBridgeSender: accounts.sender,
    rent: accounts.rent,
    systemProgram: accounts.systemProgram,
    tokenProgram: accounts.tokenProgram,
    wormholeProgram: accounts.wormholeProgram,
  };
}

export interface CompleteTransferNativeWithPayloadCpiAccounts
  extends TokenBridgeNativeRedeemerDerivedAccounts {
  payer: PublicKey;
  /**
   * seeds = ["PostedVAA", vaa_hash], seeds::program = wormholeProgram
   */
  vaa: PublicKey;
  /**
   * seeds = [emitter_address, emitter_chain, sequence], seeds::program = tokenBridgeProgram
   */
  tokenBridgeClaim: PublicKey;
  /**
   * seeds = [emitter_chain, emitter_address], seeds::program = tokenBridgeProgram
   */
  tokenBridgeForeignEndpoint: PublicKey;
  /**
   * Token account to receive tokens
   */
  toTokenAccount: PublicKey;
  toFeesTokenAccount: PublicKey; // this shouldn't exist?
  /**
   * seeds = [mint], seeds::program = tokenBridgeProgram
   */
  tokenBridgeCustody: PublicKey;
  mint: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
  tokenProgram: PublicKey;
  wormholeProgram: PublicKey;
}

/**
 * Generate accounts needed to perform `complete_native_with_payload` instruction
 * as cross-program invocation.
 *
 * Note: `toFeesTokenAccount` is the same as `toTokenAccount`. For your program,
 * you only need to pass your `toTokenAccount` into the complete transfer
 * instruction for the `toFeesTokenAccount`.
 *
 * @param tokenBridgeProgramId
 * @param wormholeProgramId
 * @param payer
 * @param vaa
 * @param toTokenAccount
 * @returns
 */
export function getCompleteTransferNativeWithPayloadCpiAccounts(
  tokenBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedTokenTransferVaa,
  toTokenAccount: PublicKeyInitData
): CompleteTransferNativeWithPayloadCpiAccounts {
  const parsed = isBytes(vaa) ? parseTokenTransferVaa(vaa) : vaa;
  const mint = new PublicKey(parsed.tokenAddress);
  const cpiProgramId = new PublicKey(parsed.to);

  return {
    payer: new PublicKey(payer),
    tokenBridgeConfig: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
    vaa: derivePostedVaaKey(wormholeProgramId, parsed.hash),
    tokenBridgeClaim: deriveClaimKey(
      tokenBridgeProgramId,
      parsed.emitterAddress,
      parsed.emitterChain,
      parsed.sequence
    ),
    tokenBridgeForeignEndpoint: deriveEndpointKey(
      tokenBridgeProgramId,
      parsed.emitterChain,
      parsed.emitterAddress
    ),
    toTokenAccount: new PublicKey(toTokenAccount),
    tokenBridgeRedeemer: deriveRedeemerAccountKey(cpiProgramId),
    toFeesTokenAccount: new PublicKey(toTokenAccount),
    tokenBridgeCustody: deriveCustodyKey(tokenBridgeProgramId, mint),
    mint,
    tokenBridgeCustodySigner: deriveCustodySignerKey(tokenBridgeProgramId),
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
    tokenProgram: TOKEN_PROGRAM_ID,
    wormholeProgram: new PublicKey(wormholeProgramId),
  };
}

export interface CompleteTransferWrappedWithPayloadCpiAccounts
  extends TokenBridgeWrappedRedeemerDerivedAccounts {
  payer: PublicKey;
  /**
   * seeds = ["PostedVAA", vaa_hash], seeds::program = wormholeProgram
   */
  vaa: PublicKey;
  /**
   * seeds = [emitter_address, emitter_chain, sequence], seeds::program = tokenBridgeProgram
   */
  tokenBridgeClaim: PublicKey;
  /**
   * seeds = [emitter_chain, emitter_address], seeds::program = tokenBridgeProgram
   */
  tokenBridgeForeignEndpoint: PublicKey;
  /**
   * Token account to receive tokens
   */
  toTokenAccount: PublicKey;
  toFeesTokenAccount: PublicKey; // this shouldn't exist?
  /**
   * seeds = ["wrapped", token_chain, token_address], seeds::program = tokenBridgeProgram
   */
  tokenBridgeWrappedMint: PublicKey;
  /**
   * seeds = ["meta", mint], seeds::program = tokenBridgeProgram
   */
  tokenBridgeWrappedMeta: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
  tokenProgram: PublicKey;
  wormholeProgram: PublicKey;
}

/**
 * Generate accounts needed to perform `complete_wrapped_with_payload` instruction
 * as cross-program invocation.
 *
 * Note: `toFeesTokenAccount` is the same as `toTokenAccount`. For your program,
 * you only need to pass your `toTokenAccount` into the complete transfer
 * instruction for the `toFeesTokenAccount`.
 *
 * @param cpiProgramId
 * @param tokenBridgeProgramId
 * @param wormholeProgramId
 * @param payer
 * @param vaa
 * @returns
 */
export function getCompleteTransferWrappedWithPayloadCpiAccounts(
  tokenBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: SignedVaa | ParsedTokenTransferVaa,
  toTokenAccount: PublicKeyInitData
): CompleteTransferWrappedWithPayloadCpiAccounts {
  const parsed = isBytes(vaa) ? parseTokenTransferVaa(vaa) : vaa;
  const mint = deriveWrappedMintKey(
    tokenBridgeProgramId,
    parsed.tokenChain,
    parsed.tokenAddress
  );
  const cpiProgramId = new PublicKey(parsed.to);
  return {
    payer: new PublicKey(payer),
    tokenBridgeConfig: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
    vaa: derivePostedVaaKey(wormholeProgramId, parsed.hash),
    tokenBridgeClaim: deriveClaimKey(
      tokenBridgeProgramId,
      parsed.emitterAddress,
      parsed.emitterChain,
      parsed.sequence
    ),
    tokenBridgeForeignEndpoint: deriveEndpointKey(
      tokenBridgeProgramId,
      parsed.emitterChain,
      parsed.emitterAddress
    ),
    toTokenAccount: new PublicKey(toTokenAccount),
    tokenBridgeRedeemer: deriveRedeemerAccountKey(cpiProgramId),
    toFeesTokenAccount: new PublicKey(toTokenAccount),
    tokenBridgeWrappedMint: mint,
    tokenBridgeWrappedMeta: deriveWrappedMetaKey(tokenBridgeProgramId, mint),
    tokenBridgeMintAuthority: deriveMintAuthorityKey(tokenBridgeProgramId),
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
    tokenProgram: TOKEN_PROGRAM_ID,
    wormholeProgram: new PublicKey(wormholeProgramId),
  };
}
