import {
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  SYSVAR_INSTRUCTIONS_PUBKEY,
  SYSVAR_RENT_PUBKEY,
  TransactionInstruction,
} from '@solana/web3.js';
import { utils } from '@wormhole-foundation/connect-sdk-solana';
import {
  getGuardianSet,
  deriveGuardianSetKey,
  getWormholeBridgeData,
} from '../accounts';
import { createReadOnlyWormholeProgramInterface } from '../program';
import { VAA } from '@wormhole-foundation/connect-sdk';
import { signatureItem } from '@wormhole-foundation/sdk-definitions/src/layout-items';

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
export async function createVerifySignaturesInstructions(
  connection: Connection,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: VAA<any>,
  signatureSet: PublicKeyInitData,
  commitment?: Commitment,
): Promise<TransactionInstruction[]> {
  const guardianSetIndex = vaa.guardianSet;
  const info = await getWormholeBridgeData(connection, wormholeProgramId);

  if (guardianSetIndex != info.guardianSetIndex)
    throw new Error('guardianSetIndex != config.guardianSetIndex');

  const guardianSetData = await getGuardianSet(
    connection,
    wormholeProgramId,
    guardianSetIndex,
    commitment,
  );

  const guardianSignatures = vaa.signatures;
  const guardianKeys = guardianSetData.keys;

  const batchSize = 7;
  const instructions: TransactionInstruction[] = [];
  for (let i = 0; i < Math.ceil(guardianSignatures.length / batchSize); ++i) {
    const start = i * batchSize;
    const end = Math.min(guardianSignatures.length, (i + 1) * batchSize);

    const signatureStatus = new Array(MAX_LEN_GUARDIAN_KEYS).fill(-1);
    const signatures: Buffer[] = [];
    const keys: Buffer[] = [];
    for (let j = 0; j < end - start; ++j) {
      const item = guardianSignatures.at(j + start)!;

      signatures.push(Buffer.from(signatureItem.custom.from(item.signature)));
      keys.push(guardianKeys.at(item.guardianIndex)!);

      signatureStatus[item.guardianIndex] = j;
    }

    instructions.push(
      utils.createSecp256k1Instruction(signatures, keys, Buffer.from(vaa.hash)),
    );

    instructions.push(
      createVerifySignaturesInstruction(
        connection,
        wormholeProgramId,
        payer,
        vaa,
        signatureSet,
        signatureStatus,
      ),
    );
  }
  return instructions;
}

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
function createVerifySignaturesInstruction(
  connection: Connection,
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  vaa: VAA,
  signatureSet: PublicKeyInitData,
  signatureStatus: number[],
): TransactionInstruction {
  const methods = createReadOnlyWormholeProgramInterface(
    wormholeProgramId,
    connection,
  ).methods.verifySignatures(signatureStatus);

  // @ts-ignore
  return methods._ixFn(...methods._args, {
    accounts: getVerifySignatureAccounts(
      wormholeProgramId,
      payer,
      signatureSet,
      vaa,
    ) as any,
    signers: undefined,
    remainingAccounts: undefined,
    preInstructions: undefined,
    postInstructions: undefined,
  });
}

export interface VerifySignatureAccounts {
  payer: PublicKey;
  guardianSet: PublicKey;
  signatureSet: PublicKey;
  instructions: PublicKey;
  rent: PublicKey;
  systemProgram: PublicKey;
}

export function getVerifySignatureAccounts(
  wormholeProgramId: PublicKeyInitData,
  payer: PublicKeyInitData,
  signatureSet: PublicKeyInitData,
  vaa: VAA,
): VerifySignatureAccounts {
  return {
    payer: new PublicKey(payer),
    guardianSet: deriveGuardianSetKey(wormholeProgramId, vaa.guardianSet),
    signatureSet: new PublicKey(signatureSet),
    instructions: SYSVAR_INSTRUCTIONS_PUBKEY,
    rent: SYSVAR_RENT_PUBKEY,
    systemProgram: SystemProgram.programId,
  };
}
