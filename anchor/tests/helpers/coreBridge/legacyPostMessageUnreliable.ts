import { BN } from "@coral-xyz/anchor";
import {
  Connection,
  Signer,
  Transaction,
  TransactionInstruction,
} from "@solana/web3.js";
import { expect } from "chai";
import { coreBridge } from "wormhole-solana-sdk";
import { CORE_BRIDGE_PROGRAM_ID } from "../consts";
import { expectIxTransactionDetails, expectIxErr } from "../transaction";
import {
  computeMaxPayloadSize,
  expectLegacyPostMessageAfterEffects,
} from "./legacyPostMessage";

export async function expectLegacyPostMessageUnreliableErr(
  connection: Connection,
  accounts: coreBridge.LegacyPostMessageUnreliableContext,
  args: coreBridge.LegacyPostMessageUnreliableArgs,
  signers: Signer[],
  expectedError: string,
  preInstructions?: TransactionInstruction[]
) {
  const ixs: TransactionInstruction[] = [];
  if (preInstructions) {
    ixs.push(...preInstructions);
  }
  ixs.push(
    coreBridge.legacyPostMessageUnreliableIx(
      CORE_BRIDGE_PROGRAM_ID,
      accounts,
      args
    )
  );

  await expectIxErr(connection, ixs, signers, expectedError);
}

export async function expectLegacyPostMessageUnreliableOk(
  connection: Connection,
  accounts: coreBridge.LegacyPostMessageContext,
  args: coreBridge.LegacyPostMessageArgs,
  signers: Signer[],
  expectedSequence: BN
) {
  const { payer, message } = accounts;
  const { nonce, payload, finalityRepr } = args;

  // For the purposes of this post message unreliable test, we assume that the
  // number of signers is always the max (3).
  const maxLength = computeMaxPayloadSize(3);
  expect(payload).length.lessThanOrEqual(maxLength);
  expect(finalityRepr).greaterThanOrEqual(0).and.lessThanOrEqual(1);

  // Modify existing payload.
  const expectedPayload =
    await coreBridge.PostedMessageV1Unreliable.fromAccountAddress(
      connection,
      message
    )
      .then((posted) => posted.payload)
      .catch((_) => Buffer.alloc(maxLength));
  expect(expectedPayload.length).greaterThanOrEqual(payload.length);
  expectedPayload.fill(0);
  expectedPayload.write(payload.toString("hex"), 0, "hex");

  const modifiedArgs = {
    nonce,
    payload: expectedPayload,
    finalityRepr,
  };

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
      coreBridge.legacyPostMessageUnreliableIx(
        CORE_BRIDGE_PROGRAM_ID,
        accounts,
        modifiedArgs
      ),
    ],
    signers
  );

  await expectLegacyPostMessageAfterEffects(
    connection,
    txDetails!,
    accounts,
    modifiedArgs,
    expectedSequence,
    true,
    expectedPayload
  );
}
