import {
  PublicKey,
  PublicKeyInitData,
  TransactionInstruction,
} from "@solana/web3.js";
import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import { createReadOnlyTokenBridgeProgramInterface } from "../program";
import { getPostMessageCpiAccounts } from "../../wormhole";
import {
  deriveAuthoritySignerKey,
  deriveTokenBridgeConfigKey,
  deriveWrappedMetaKey,
  deriveWrappedMintKey,
} from "../accounts";

export function createTransferWrappedInstruction(
  tokenBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  message: PublicKeyInitData,
  from: PublicKeyInitData,
  fromOwner: PublicKeyInitData,
  tokenChain: number,
  tokenAddress: Buffer | Uint8Array,
  nonce: number,
  amount: bigint,
  fee: bigint,
  targetAddress: Buffer | Uint8Array,
  targetChain: number
): TransactionInstruction {
  const methods = createReadOnlyTokenBridgeProgramInterface(
    tokenBridgeProgramId
  ).methods.transferWrapped(
    nonce,
    amount as any,
    fee as any,
    Buffer.from(targetAddress) as any,
    targetChain
  );

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getTransferWrappedAccounts(
      tokenBridgeProgramId,
      wormholeProgramId,
      payer,
      message,
      from,
      fromOwner,
      tokenChain,
      tokenAddress
    ) as any,
    signers: undefined,
    remainingAccounts: undefined,
    preInstructions: undefined,
    postInstructions: undefined,
  });
}

export interface TransferWrappedAccounts {
  payer: PublicKey;
  config: PublicKey;
  from: PublicKey;
  fromOwner: PublicKey;
  mint: PublicKey;
  wrappedMeta: PublicKey;
  authoritySigner: PublicKey;
  wormholeBridge: PublicKey;
  wormholeMessage: PublicKey;
  wormholeEmitter: PublicKey;
  wormholeSequence: PublicKey;
  wormholeFeeCollector: PublicKey;
  clock: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
  wormholeProgram: PublicKey;
  tokenProgram: PublicKey;
}

export function getTransferWrappedAccounts(
  tokenBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  message: PublicKeyInitData,
  from: PublicKeyInitData,
  fromOwner: PublicKeyInitData,
  tokenChain: number,
  tokenAddress: Buffer | Uint8Array
): TransferWrappedAccounts {
  const mint = deriveWrappedMintKey(
    tokenBridgeProgramId,
    tokenChain,
    tokenAddress
  );
  const {
    wormholeBridge,
    wormholeMessage,
    wormholeEmitter,
    wormholeSequence,
    wormholeFeeCollector,
    clock,
    rent,
    systemProgram,
  } = getPostMessageCpiAccounts(
    tokenBridgeProgramId,
    wormholeProgramId,
    payer,
    message
  );
  return {
    payer: new PublicKey(payer),
    config: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
    from: new PublicKey(from),
    fromOwner: new PublicKey(fromOwner),
    mint: mint,
    wrappedMeta: deriveWrappedMetaKey(tokenBridgeProgramId, mint),
    authoritySigner: deriveAuthoritySignerKey(tokenBridgeProgramId),
    wormholeBridge,
    wormholeMessage: wormholeMessage,
    wormholeEmitter,
    wormholeSequence,
    wormholeFeeCollector,
    clock,
    rent,
    systemProgram,
    wormholeProgram: new PublicKey(wormholeProgramId),
    tokenProgram: TOKEN_PROGRAM_ID,
  };
}
