import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import {
  PublicKey,
  PublicKeyInitData,
  TransactionInstruction,
} from "@solana/web3.js";
import { TOKEN_METADATA_PROGRAM_ID, deriveTokenMetadataKey } from "../../utils";
import { getPostMessageAccounts } from "../../wormhole";
import {
  deriveAuthoritySignerKey,
  deriveCustodyKey,
  deriveCustodySignerKey,
  deriveNftBridgeConfigKey,
} from "../accounts";
import { createReadOnlyNftBridgeProgramInterface } from "../program";

export function createTransferNativeInstruction(
  nftBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  message: PublicKeyInitData,
  from: PublicKeyInitData,
  mint: PublicKeyInitData,
  nonce: number,
  targetAddress: Buffer | Uint8Array,
  targetChain: number
): TransactionInstruction {
  const methods = createReadOnlyNftBridgeProgramInterface(
    nftBridgeProgramId
  ).methods.transferNative(
    nonce,
    Buffer.from(targetAddress) as any,
    targetChain
  );

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getTransferNativeAccounts(
      nftBridgeProgramId,
      wormholeProgramId,
      payer,
      message,
      from,
      mint
    ) as any,
    signers: undefined,
    remainingAccounts: undefined,
    preInstructions: undefined,
    postInstructions: undefined,
  });
}

export interface TransferNativeAccounts {
  payer: PublicKey;
  config: PublicKey;
  from: PublicKey;
  mint: PublicKey;
  splMetadata: PublicKey;
  custody: PublicKey;
  authoritySigner: PublicKey;
  custodySigner: PublicKey;
  wormholeBridge: PublicKey;
  wormholeMessage: PublicKey;
  wormholeEmitter: PublicKey;
  wormholeSequence: PublicKey;
  wormholeFeeCollector: PublicKey;
  clock: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
  tokenProgram: PublicKey;
  splMetadataProgram: PublicKey;
  wormholeProgram: PublicKey;
}

export function getTransferNativeAccounts(
  nftBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  message: PublicKeyInitData,
  from: PublicKeyInitData,
  mint: PublicKeyInitData
): TransferNativeAccounts {
  const {
    bridge: wormholeBridge,
    message: wormholeMessage,
    emitter: wormholeEmitter,
    sequence: wormholeSequence,
    feeCollector: wormholeFeeCollector,
    clock,
    rent,
    systemProgram,
  } = getPostMessageAccounts(
    wormholeProgramId,
    payer,
    nftBridgeProgramId,
    message
  );
  return {
    payer: new PublicKey(payer),
    config: deriveNftBridgeConfigKey(nftBridgeProgramId),
    from: new PublicKey(from),
    mint: new PublicKey(mint),
    splMetadata: deriveTokenMetadataKey(mint),
    custody: deriveCustodyKey(nftBridgeProgramId, mint),
    authoritySigner: deriveAuthoritySignerKey(nftBridgeProgramId),
    custodySigner: deriveCustodySignerKey(nftBridgeProgramId),
    wormholeBridge,
    wormholeMessage,
    wormholeEmitter,
    wormholeSequence,
    wormholeFeeCollector,
    clock,
    rent,
    systemProgram,
    tokenProgram: TOKEN_PROGRAM_ID,
    splMetadataProgram: TOKEN_METADATA_PROGRAM_ID,
    wormholeProgram: new PublicKey(wormholeProgramId),
  };
}
