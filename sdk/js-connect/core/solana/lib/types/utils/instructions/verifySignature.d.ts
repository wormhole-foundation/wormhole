import { Commitment, Connection, PublicKey, PublicKeyInitData, TransactionInstruction } from '@solana/web3.js';
import { VAA } from '@wormhole-foundation/connect-sdk';
/**
 * This is used in {@link createPostSignedVaaTransactions}'s initial transactions.
 *
 * Signatures are batched in groups of 7 due to instruction
 * data limits. These signatures are passed through to the Secp256k1
 * program to verify that the guardian public keys can be recovered.
 * This instruction is paired with `verify_signatures` to validate the
 * pubkey recovery.
 *
 * There are at most three pairs of instructions created.
 *
 * https://github.com/certusone/wormhole/blob/main/solana/bridge/program/src/api/verify_signature.rs
 *
 *
 * @param {Connection} connection - Solana web3 connection
 * @param {PublicKeyInitData} wormholeProgramId - wormhole program address
 * @param {PublicKeyInitData} payer - transaction signer address
 * @param {SignedVaa | ParsedVaa} vaa - either signed VAA bytes or parsed VAA (use {@link parseVaa} on signed VAA)
 * @param {PublicKeyInitData} signatureSet - address to account of verified signatures
 * @param {web3.ConfirmOptions} [options] - Solana confirmation options
 */
export declare function createVerifySignaturesInstructions(connection: Connection, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: VAA<any>, signatureSet: PublicKeyInitData, commitment?: Commitment): Promise<TransactionInstruction[]>;
export interface VerifySignatureAccounts {
    payer: PublicKey;
    guardianSet: PublicKey;
    signatureSet: PublicKey;
    instructions: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
}
export declare function getVerifySignatureAccounts(wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, signatureSet: PublicKeyInitData, vaa: VAA): VerifySignatureAccounts;
