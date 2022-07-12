import { PublicKey, PublicKeyInitData } from "@solana/web3.js";
import {
  deriveEmitterSequenceKey,
  deriveFeeCollectorKey,
  deriveWormholeEmitterKey,
  deriveWormholeBridgeDataKey,
  getEmitterKeys,
} from "./accounts";
import { getPostMessageAccounts } from "./instructions";

export interface WormholeDerivedAccounts {
  /**
   * seeds = ["Bridge"], seeds::program = wormholeProgram
   */
  wormholeBridge: PublicKey;
  /**
   * seeds = ["emitter"], seeds::program = cpiProgramId
   */
  wormholeEmitter: PublicKey;
  /**
   * seeds = ["Sequence", wormholeEmitter], seeds::program = wormholeProgram
   */
  wormholeSequence: PublicKey;
  /**
   * seeds = ["fee_collector"], seeds::program = wormholeProgram
   */
  wormholeFeeCollector: PublicKey;
}

/**
 * Generate Wormhole PDAs.
 *
 * @param cpiProgramId
 * @param wormholeProgramId
 * @returns
 */
export function getWormholeDerivedAccounts(
  cpiProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData
): WormholeDerivedAccounts {
  const { emitter: wormholeEmitter, sequence: wormholeSequence } =
    getEmitterKeys(cpiProgramId, wormholeProgramId);
  return {
    wormholeBridge: deriveWormholeBridgeDataKey(wormholeProgramId),
    wormholeEmitter,
    wormholeSequence,
    wormholeFeeCollector: deriveFeeCollectorKey(wormholeProgramId),
  };
}

export interface PostMessageCpiAccounts extends WormholeDerivedAccounts {
  payer: PublicKey;
  wormholeMessage: PublicKey;
  clock: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
}

/**
 * Generate accounts needed to perform `post_message` instruction
 * as cross-program invocation.
 *
 * @param cpiProgramId
 * @param wormholeProgramId
 * @param payer
 * @param message
 * @returns
 */
export function getPostMessageCpiAccounts(
  cpiProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  message: PublicKeyInitData
): PostMessageCpiAccounts {
  const accounts = getPostMessageAccounts(
    wormholeProgramId,
    payer,
    cpiProgramId,
    message
  );
  return {
    payer: accounts.payer,
    wormholeBridge: accounts.bridge,
    wormholeMessage: accounts.message,
    wormholeEmitter: accounts.emitter,
    wormholeSequence: accounts.sequence,
    wormholeFeeCollector: accounts.feeCollector,
    clock: accounts.clock,
    rent: accounts.rent,
    systemProgram: accounts.systemProgram,
  };
}
