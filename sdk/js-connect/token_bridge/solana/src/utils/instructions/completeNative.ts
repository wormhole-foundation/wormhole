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
  deriveCustodyKey,
  deriveCustodySignerKey,
} from '../accounts';
import { TokenBridge, toChainId } from '@wormhole-foundation/connect-sdk';

export function createCompleteTransferNativeInstruction(
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
  ).methods.completeNative();

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getCompleteTransferNativeAccounts(
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
  vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload'>,
  feeRecipient?: PublicKeyInitData,
): CompleteTransferNativeAccounts {
  const mint = new PublicKey(vaa.payload.token.address.toUint8Array());
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
    custody: deriveCustodyKey(tokenBridgeProgramId, mint),
    mint,
    custodySigner: deriveCustodySignerKey(tokenBridgeProgramId),
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
    tokenProgram: TOKEN_PROGRAM_ID,
    wormholeProgram: new PublicKey(wormholeProgramId),
  };
}
