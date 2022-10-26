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
  deriveSenderAccountKey,
  deriveTokenBridgeConfigKey,
  deriveWrappedMetaKey,
  deriveWrappedMintKey,
} from "../accounts";

export function createTransferWrappedWithPayloadInstruction(
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
  targetAddress: Buffer | Uint8Array,
  targetChain: number,
  payload: Buffer | Uint8Array
): TransactionInstruction {
  const methods = createReadOnlyTokenBridgeProgramInterface(
    tokenBridgeProgramId
  ).methods.transferWrappedWithPayload(
    nonce,
    amount as any,
    Buffer.from(targetAddress) as any,
    targetChain,
    Buffer.from(payload) as any,
    null
  );

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getTransferWrappedWithPayloadAccounts(
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

export interface TransferWrappedWithPayloadAccounts {
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
  sender: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
  tokenProgram: PublicKey;
  wormholeProgram: PublicKey;
}

export function getTransferWrappedWithPayloadAccounts(
  tokenBridgeProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  message: PublicKeyInitData,
  from: PublicKeyInitData,
  fromOwner: PublicKeyInitData,
  tokenChain: number,
  tokenAddress: Buffer | Uint8Array,
  cpiProgramId?: PublicKeyInitData
): TransferWrappedWithPayloadAccounts {
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
    sender: new PublicKey(
      cpiProgramId === undefined ? payer : deriveSenderAccountKey(cpiProgramId)
    ),
    rent,
    systemProgram,
    wormholeProgram: new PublicKey(wormholeProgramId),
    tokenProgram: TOKEN_PROGRAM_ID,
  };
}
