import {
  Connection,
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  SYSVAR_RENT_PUBKEY,
  TransactionInstruction,
} from '@solana/web3.js';
import { TOKEN_PROGRAM_ID } from '@solana/spl-token';
import { createReadOnlyTokenBridgeProgramInterface } from '../program';
import { utils } from '@wormhole-foundation/wormhole-connect-sdk-core-solana';
import {
  deriveEndpointKey,
  deriveTokenBridgeConfigKey,
  deriveWrappedMintKey,
  deriveWrappedMetaKey,
  deriveMintAuthorityKey,
} from '../accounts';
import { TokenBridge, toChainId } from '@wormhole-foundation/connect-sdk';

export function createCompleteTransferWrappedInstruction(
  connection: Connection,
  tokenBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload'>,
  feeRecipient?: PublicKeyInitData,
): TransactionInstruction {
  const methods = createReadOnlyTokenBridgeProgramInterface(
    tokenBridgeProgramId,
    connection,
  ).methods.completeWrapped();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getCompleteTransferWrappedAccounts(
      tokenBridgeProgramId,
      wormholeProgramId,
      payer,
      vaa,
      feeRecipient,
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
  vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload'>,
  feeRecipient?: PublicKeyInitData,
): CompleteTransferWrappedAccounts {
  const mint = deriveWrappedMintKey(
    tokenBridgeProgramId,
    toChainId(vaa.payload.token.chain),
    vaa.payload.token.address.toUint8Array(),
  );
  return {
    payer: new PublicKey(payer),
    config: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
    vaa: utils.derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
    claim: utils.deriveClaimKey(
      tokenBridgeProgramId,
      vaa.emitterAddress.toUint8Array(),
      toChainId(vaa.emitterChain),
      vaa.sequence,
    ),
    endpoint: deriveEndpointKey(
      tokenBridgeProgramId,
      toChainId(vaa.emitterChain),
      vaa.emitterAddress.toUint8Array(),
    ),
    to: new PublicKey(vaa.payload.to.address.toUint8Array()),
    toFees: new PublicKey(
      feeRecipient === undefined
        ? vaa.payload.to.address.toUint8Array()
        : feeRecipient,
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
