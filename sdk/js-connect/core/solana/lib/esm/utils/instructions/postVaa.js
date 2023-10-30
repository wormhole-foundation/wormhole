import { PublicKey, SYSVAR_CLOCK_PUBKEY, SYSVAR_RENT_PUBKEY, SystemProgram, } from '@solana/web3.js';
import { createReadOnlyWormholeProgramInterface } from '../program';
import { deriveWormholeBridgeDataKey, deriveGuardianSetKey, derivePostedVaaKey, } from '../accounts';
import { serializePayload, toChainId, } from '@wormhole-foundation/connect-sdk';
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
export function createPostVaaInstruction(connection, wormholeProgramId, payer, vaa, signatureSet) {
    const methods = createReadOnlyWormholeProgramInterface(wormholeProgramId, connection).methods.postVaa(1, // TODO: hardcoded VAA version
    vaa.guardianSet, vaa.timestamp, vaa.nonce, toChainId(vaa.emitterChain), [...vaa.emitterAddress.toUint8Array()], BigInt(vaa.sequence.toString()), vaa.consistencyLevel, 
    // Note: This _must_ be a Buffer, a Uint8Array does not work
    Buffer.from(serializePayload(vaa.payloadLiteral, vaa.payload)));
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getPostVaaAccounts(wormholeProgramId, payer, signatureSet, vaa),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getPostVaaAccounts(wormholeProgramId, payer, signatureSet, vaa) {
    return {
        guardianSet: deriveGuardianSetKey(wormholeProgramId, vaa.guardianSet),
        bridge: deriveWormholeBridgeDataKey(wormholeProgramId),
        signatureSet: new PublicKey(signatureSet),
        vaa: derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        payer: new PublicKey(payer),
        clock: SYSVAR_CLOCK_PUBKEY,
        rent: SYSVAR_RENT_PUBKEY,
        systemProgram: SystemProgram.programId,
    };
}
