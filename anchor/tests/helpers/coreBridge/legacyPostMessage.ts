import { BN } from "@coral-xyz/anchor";
import {
  Connection,
  PublicKey,
  Signer,
  Transaction,
  TransactionInstruction,
  VersionedTransactionResponse,
} from "@solana/web3.js";
import { expect } from "chai";
import { coreBridge } from "wormhole-solana-sdk";
import { CORE_BRIDGE_PROGRAM_ID } from "../consts";
import { expectIxTransactionDetails, expectIxErr } from "../transaction";
import { MessageStatus } from "wormhole-solana-sdk/src/coreBridge";

export async function expectLegacyPostMessageErr(
  connection: Connection,
  accounts: coreBridge.LegacyPostMessageContext,
  args: coreBridge.LegacyPostMessageArgs,
  signers: Signer[],
  expectedError: string,
  preInstructions?: TransactionInstruction[]
) {
  const ixs: TransactionInstruction[] = [];
  if (preInstructions) {
    ixs.push(...preInstructions);
  }
  ixs.push(
    coreBridge.legacyPostMessageIx(CORE_BRIDGE_PROGRAM_ID, accounts, args)
  );

  await expectIxErr(connection, ixs, signers, expectedError);
}

// NOTE: This assumes that rent and clock pubkeys are passed into the account
// context, which account for an additional 64 bytes. With the new legacy post
// message implementation, these accounts are not required (so we can pass in
// the default pubkey, which is the same as the System Program's) so we will
// be able to send an additional 64 bytes. But for the purposes of these tests,
// we will assume that we are using the old implementation.
export function computeMaxPayloadSize(numSigners: number): number {
  const additional = numSigners > 2 ? (numSigners - 2) * 64 : 0;
  return 674 - additional;
}

type OkOptions = {
  actualPayload?: Buffer;
};

export async function expectLegacyPostMessageOk(
  connection: Connection,
  accounts: coreBridge.LegacyPostMessageContext,
  args: coreBridge.LegacyPostMessageArgs,
  signers: Signer[],
  expectedSequence: BN,
  okOptions: OkOptions = {}
) {
  const { payer } = accounts;
  const { payload, finalityRepr } = args;

  const maxLength = computeMaxPayloadSize(signers.length);
  expect(payload).length.lessThanOrEqual(maxLength);
  expect(finalityRepr).greaterThanOrEqual(0).and.lessThanOrEqual(1);

  // We must pay the fee collector prior publishing a message.
  const transferIx = await coreBridge.transferMessageFeeIx(
    connection,
    CORE_BRIDGE_PROGRAM_ID,
    payer
  );

  // Execute.
  const txDetails = await expectIxTransactionDetails(
    connection,
    [
      transferIx,
      coreBridge.legacyPostMessageIx(CORE_BRIDGE_PROGRAM_ID, accounts, args),
    ],
    signers
  );

  await expectLegacyPostMessageAfterEffects(
    connection,
    txDetails!,
    accounts,
    args,
    expectedSequence,
    false,
    okOptions.actualPayload ?? payload
  );
}

export async function expectLegacyPostMessageAfterEffects(
  connection: Connection,
  txDetails: VersionedTransactionResponse,
  accounts: coreBridge.LegacyPostMessageContext,
  args: coreBridge.LegacyPostMessageArgs,
  expectedSequence: BN,
  unreliable: boolean,
  actualPayload: Buffer
) {
  const { emitter, message } = accounts;
  const { nonce, payload, finalityRepr } = args;

  const {
    status,
    finality,
    emitterAuthority,
    _gap0,
    postedTimestamp,
    nonce: messageNonce,
    sequence: messageSequence,
    solanaChainId,
    emitter: emitterAddress,
    payload: messagePayload,
  } = await (unreliable
    ? coreBridge.PostedMessageV1Unreliable.fromAccountAddress(
        connection,
        message
      )
    : coreBridge.PostedMessageV1.fromAccountAddress(connection, message));
  expect(finality).equals(finalityRepr === 0 ? 1 : 32);
  expect(emitterAuthority.equals(PublicKey.default)).is.true;
  expect(status).equals(MessageStatus.Unset);
  expect(_gap0.equals(Buffer.alloc(3))).is.true;
  expect(postedTimestamp).equals(txDetails.blockTime!);
  expect(messageNonce).equals(nonce);
  console.log(
    messageSequence.toString(),
    "vs expected",
    expectedSequence.toString()
  );
  expect(messageSequence.eq(expectedSequence)).is.true;
  expect(solanaChainId).equals(1);
  expect(emitterAddress.equals(emitter)).is.true;
  if (actualPayload.equals(payload)) {
    expect(payload).has.length.greaterThan(0);
    expect(messagePayload.equals(payload)).is.true;
  } else {
    expect(payload).has.length(0);
    expect(messagePayload.equals(actualPayload)).is.true;
  }

  // Get emitter sequence.
  const emitterSequenceValue = await coreBridge.EmitterSequence.fromPda(
    connection,
    CORE_BRIDGE_PROGRAM_ID,
    emitter
  ).then((tracker) => tracker.sequence);
  expect(emitterSequenceValue.eq(expectedSequence.addn(1))).is.true;
}
