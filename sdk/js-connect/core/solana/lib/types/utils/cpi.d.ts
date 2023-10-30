import { PublicKey, PublicKeyInitData } from '@solana/web3.js';
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
export declare function getWormholeDerivedAccounts(cpiProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData): WormholeDerivedAccounts;
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
export declare function getPostMessageCpiAccounts(cpiProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, message: PublicKeyInitData): PostMessageCpiAccounts;
