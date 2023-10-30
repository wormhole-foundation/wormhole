"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getPostVaaAccounts = exports.createPostVaaInstruction = void 0;
const web3_js_1 = require("@solana/web3.js");
const program_1 = require("../program");
const accounts_1 = require("../accounts");
const connect_sdk_1 = require("@wormhole-foundation/connect-sdk");
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
function createPostVaaInstruction(connection, wormholeProgramId, payer, vaa, signatureSet) {
    const methods = (0, program_1.createReadOnlyWormholeProgramInterface)(wormholeProgramId, connection).methods.postVaa(1, // TODO: hardcoded VAA version
    vaa.guardianSet, vaa.timestamp, vaa.nonce, (0, connect_sdk_1.toChainId)(vaa.emitterChain), [...vaa.emitterAddress.toUint8Array()], BigInt(vaa.sequence.toString()), vaa.consistencyLevel, 
    // Note: This _must_ be a Buffer, a Uint8Array does not work
    Buffer.from((0, connect_sdk_1.serializePayload)(vaa.payloadLiteral, vaa.payload)));
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getPostVaaAccounts(wormholeProgramId, payer, signatureSet, vaa),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
exports.createPostVaaInstruction = createPostVaaInstruction;
function getPostVaaAccounts(wormholeProgramId, payer, signatureSet, vaa) {
    return {
        guardianSet: (0, accounts_1.deriveGuardianSetKey)(wormholeProgramId, vaa.guardianSet),
        bridge: (0, accounts_1.deriveWormholeBridgeDataKey)(wormholeProgramId),
        signatureSet: new web3_js_1.PublicKey(signatureSet),
        vaa: (0, accounts_1.derivePostedVaaKey)(wormholeProgramId, Buffer.from(vaa.hash)),
        payer: new web3_js_1.PublicKey(payer),
        clock: web3_js_1.SYSVAR_CLOCK_PUBKEY,
        rent: web3_js_1.SYSVAR_RENT_PUBKEY,
        systemProgram: web3_js_1.SystemProgram.programId,
    };
}
exports.getPostVaaAccounts = getPostVaaAccounts;
