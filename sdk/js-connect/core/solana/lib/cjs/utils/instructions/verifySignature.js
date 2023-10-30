"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getVerifySignatureAccounts = exports.createVerifySignaturesInstructions = void 0;
const web3_js_1 = require("@solana/web3.js");
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
const accounts_1 = require("../accounts");
const program_1 = require("../program");
const layout_items_1 = require("@wormhole-foundation/sdk-definitions/src/layout-items");
const MAX_LEN_GUARDIAN_KEYS = 19;
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
function createVerifySignaturesInstructions(connection, wormholeProgramId, payer, vaa, signatureSet, commitment) {
    return __awaiter(this, void 0, void 0, function* () {
        const guardianSetIndex = vaa.guardianSet;
        const info = yield (0, accounts_1.getWormholeBridgeData)(connection, wormholeProgramId);
        if (guardianSetIndex != info.guardianSetIndex)
            throw new Error('guardianSetIndex != config.guardianSetIndex');
        const guardianSetData = yield (0, accounts_1.getGuardianSet)(connection, wormholeProgramId, guardianSetIndex, commitment);
        const guardianSignatures = vaa.signatures;
        const guardianKeys = guardianSetData.keys;
        const batchSize = 7;
        const instructions = [];
        for (let i = 0; i < Math.ceil(guardianSignatures.length / batchSize); ++i) {
            const start = i * batchSize;
            const end = Math.min(guardianSignatures.length, (i + 1) * batchSize);
            const signatureStatus = new Array(MAX_LEN_GUARDIAN_KEYS).fill(-1);
            const signatures = [];
            const keys = [];
            for (let j = 0; j < end - start; ++j) {
                const item = guardianSignatures.at(j + start);
                signatures.push(Buffer.from(layout_items_1.signatureItem.custom.from(item.signature)));
                keys.push(guardianKeys.at(item.guardianIndex));
                signatureStatus[item.guardianIndex] = j;
            }
            instructions.push(connect_sdk_solana_1.utils.createSecp256k1Instruction(signatures, keys, Buffer.from(vaa.hash)));
            instructions.push(createVerifySignaturesInstruction(connection, wormholeProgramId, payer, vaa, signatureSet, signatureStatus));
        }
        return instructions;
    });
}
exports.createVerifySignaturesInstructions = createVerifySignaturesInstructions;
/**
 * Make {@link TransactionInstruction} for `verify_signatures` instruction.
 *
 * This is used in {@link createVerifySignaturesInstructions} for each batch of signatures being verified.
 * `signatureSet` is a {@link @solana/web3.Keypair} generated outside of this method, used
 * for writing signatures and the message hash to.
 *
 * https://github.com/certusone/wormhole/blob/main/solana/bridge/program/src/api/verify_signature.rs
 *
 * @param {PublicKeyInitData} wormholeProgramId - wormhole program address
 * @param {PublicKeyInitData} payer - transaction signer address
 * @param {SignedVaa | ParsedVaa} vaa - either signed VAA (Buffer) or parsed VAA (use {@link parseVaa} on signed VAA)
 * @param {PublicKeyInitData} signatureSet - key for signature set account
 * @param {Buffer} signatureStatus - array of guardian indices
 *
 */
function createVerifySignaturesInstruction(connection, wormholeProgramId, payer, vaa, signatureSet, signatureStatus) {
    const methods = (0, program_1.createReadOnlyWormholeProgramInterface)(wormholeProgramId, connection).methods.verifySignatures(signatureStatus);
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getVerifySignatureAccounts(wormholeProgramId, payer, signatureSet, vaa),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
function getVerifySignatureAccounts(wormholeProgramId, payer, signatureSet, vaa) {
    return {
        payer: new web3_js_1.PublicKey(payer),
        guardianSet: (0, accounts_1.deriveGuardianSetKey)(wormholeProgramId, vaa.guardianSet),
        signatureSet: new web3_js_1.PublicKey(signatureSet),
        instructions: web3_js_1.SYSVAR_INSTRUCTIONS_PUBKEY,
        rent: web3_js_1.SYSVAR_RENT_PUBKEY,
        systemProgram: web3_js_1.SystemProgram.programId,
    };
}
exports.getVerifySignatureAccounts = getVerifySignatureAccounts;
