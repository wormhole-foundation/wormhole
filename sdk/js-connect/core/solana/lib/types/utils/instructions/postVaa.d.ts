import { PublicKey, PublicKeyInitData, TransactionInstruction, Connection } from '@solana/web3.js';
import { VAA } from '@wormhole-foundation/connect-sdk';
/**
 * Make {@link TransactionInstruction} for `post_vaa` instruction.
 *
 * This is used in {@link createPostSignedVaaTransactions}'s last transaction.
 * `signatureSet` is a {@link @solana/web3.Keypair} generated outside of this method, which was used
 * to write signatures and the message hash to.
 *
 * https://github.com/certusone/wormhole/blob/main/solana/bridge/program/src/api/post_vaa.rs
 *
 * @param {PublicKeyInitData} wormholeProgramId - wormhole program address
 * @param {PublicKeyInitData} payer - transaction signer address
 * @param {SignedVaa | ParsedVaa} vaa - either signed VAA bytes or parsed VAA (use {@link parseVaa} on signed VAA)
 * @param {PublicKeyInitData} signatureSet - key for signature set account
 */
export declare function createPostVaaInstruction(connection: Connection, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: VAA, signatureSet: PublicKeyInitData): TransactionInstruction;
export interface PostVaaAccounts {
    guardianSet: PublicKey;
    bridge: PublicKey;
    signatureSet: PublicKey;
    vaa: PublicKey;
    payer: PublicKey;
    clock: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
}
export declare function getPostVaaAccounts(wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, signatureSet: PublicKeyInitData, vaa: VAA): PostVaaAccounts;
