import { BN } from "@coral-xyz/anchor";
import { Keypair, PublicKey, VersionedTransactionResponse } from "@solana/web3.js";
import { expect } from "chai";
import { Config, CoreBridgeProgram } from ".";
import * as coreBridge from "../coreBridge";
import { expectDeepEqual, expectIxOkDetails } from "../utils";

export async function expectEqualBridgeAccounts(
  program: CoreBridgeProgram,
  forkedProgram: CoreBridgeProgram
) {
  const connection = program.provider.connection;

  const [bridgeData, forkBridgeData] = await Promise.all([
    Config.fromPda(connection, program.programId),
    Config.fromPda(connection, forkedProgram.programId),
  ]);

  // Do not check last lamports because these two values will diverge due to the change in message
  // fee handling.
  const { lastLamports, ...otherBridgeData } = bridgeData;
  const { lastLamports: forkedLastLamports, ...otherForkBridgeData } = forkBridgeData;
  expectDeepEqual(otherBridgeData, otherForkBridgeData);
}

export async function expectEqualMessageAccounts(
  program: CoreBridgeProgram,
  messageSigner: Keypair,
  forkedMessageSigner: Keypair,
  unreliable: boolean,
  sameEmitter: boolean = true
) {
  const connection = program.provider.connection;

  const [messageData, forkedMessageData] = await (async () => {
    if (unreliable) {
      return Promise.all([
        coreBridge.PostedMessageV1Unreliable.fromAccountAddress(
          connection,
          messageSigner.publicKey
        ),
        coreBridge.PostedMessageV1Unreliable.fromAccountAddress(
          connection,
          forkedMessageSigner.publicKey
        ),
      ]);
    } else {
      return Promise.all([
        coreBridge.PostedMessageV1.fromAccountAddress(connection, messageSigner.publicKey),
        coreBridge.PostedMessageV1.fromAccountAddress(connection, forkedMessageSigner.publicKey),
      ]);
    }
  })();

  if (sameEmitter) {
    expectDeepEqual(messageData, forkedMessageData);
  } else {
    const { emitter: _a, ...other } = messageData;
    const { emitter: _b, ...forkedOther } = forkedMessageData;
    expectDeepEqual(other, forkedOther);
  }
}

export async function expectEqualGuardianSet(
  program: CoreBridgeProgram,
  forkedProgram: CoreBridgeProgram,
  guardianSetIndex: number
) {
  const connection = program.provider.connection;

  const [guardianSet, forkedGuardianSet] = await Promise.all([
    coreBridge.GuardianSet.fromPda(connection, program.programId, guardianSetIndex),
    coreBridge.GuardianSet.fromPda(connection, forkedProgram.programId, guardianSetIndex),
  ]);
  expectDeepEqual(guardianSet, forkedGuardianSet);
}

export async function expectLegacyPostMessageAfterEffects(
  program: CoreBridgeProgram,
  txDetails: VersionedTransactionResponse,
  accounts: coreBridge.LegacyPostMessageContext,
  args: coreBridge.LegacyPostMessageArgs,
  expectedSequence: BN,
  unreliable: boolean,
  actualPayload: Buffer
) {
  const connection = program.provider.connection;

  const { emitter, message } = accounts;
  const { nonce, payload, commitment } = args;

  const {
    status,
    consistencyLevel,
    emitterAuthority,
    _gap0,
    postedTimestamp,
    nonce: messageNonce,
    sequence: messageSequence,
    solanaChainId,
    emitter: emitterAddress,
    payload: messagePayload,
  } = await (unreliable
    ? coreBridge.PostedMessageV1Unreliable.fromAccountAddress(connection, message)
    : coreBridge.PostedMessageV1.fromAccountAddress(connection, message));

  expect(consistencyLevel).equals(commitment === "confirmed" ? 1 : 32);
  expect(emitterAuthority.equals(PublicKey.default)).is.true;
  expect(status).equals(coreBridge.MessageStatus.Unset);
  expect(_gap0.equals(Buffer.alloc(3))).is.true;
  expect(postedTimestamp).equals(txDetails.blockTime!);
  expect(messageNonce).equals(nonce);
  expect(messageSequence.eq(expectedSequence)).is.true;
  expect(solanaChainId).equals(1);
  expect(emitterAddress.equals(emitter!)).is.true;

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
    program.programId,
    emitter!
  ).then((tracker) => tracker.sequence);
  expect(emitterSequenceValue.eq(expectedSequence.addn(1))).is.true;
}

export async function expectOkPostMessage(
  program: coreBridge.CoreBridgeProgram,
  signers: {
    payer: Keypair;
    message: Keypair | null;
    emitter: Keypair | null;
  },
  args: coreBridge.LegacyPostMessageArgs,
  sequence: BN,
  expected: {
    consistencyLevel: number;
    nonce: number;
    payload: Buffer;
    message?: PublicKey;
    emitter?: PublicKey;
  },
  nullAccounts?: { feeCollector: boolean; clock: boolean; rent: boolean },
  emitterSequence?: PublicKey,
  createTransferFeeIx: boolean = true
) {
  if (nullAccounts === undefined) {
    nullAccounts = { feeCollector: false, clock: false, rent: false };
  }

  const { payer, message: messageSigner, emitter: emitterSigner } = signers;
  const txSigners = [payer];

  if (messageSigner !== null) {
    txSigners.push(messageSigner);
  }

  if (emitterSigner !== null) {
    txSigners.push(emitterSigner);
  }

  let { message, emitter } = expected;
  if (message === undefined) {
    message = messageSigner?.publicKey;
  }
  if (emitter === undefined) {
    emitter = emitterSigner?.publicKey;
  }
  const accounts = {
    message,
    emitter: emitterSigner === null ? null : emitter,
    emitterSequence,
    payer: payer.publicKey,
  } as coreBridge.LegacyPostMessageContext;
  for (const [key, isNull] of Object.entries(nullAccounts)) {
    accounts[key] = isNull ? null : undefined;
  }

  const ix = coreBridge.legacyPostMessageIx(program, accounts, args, {
    message: messageSigner !== null,
  });

  // If any accounts are null, confirm they are "null" in the instruction.
  if (nullAccounts.feeCollector) {
    expectDeepEqual(ix.keys[5].pubkey, program.programId);
  }
  if (nullAccounts.clock) {
    expectDeepEqual(ix.keys[6].pubkey, program.programId);
  }
  if (nullAccounts.rent) {
    expectDeepEqual(ix.keys[8].pubkey, program.programId);
  }

  const connection = program.provider.connection;

  const txDetails = await expectIxOkDetails(
    connection,
    createTransferFeeIx
      ? [await coreBridge.transferMessageFeeIx(program, payer.publicKey), ix]
      : [ix],
    txSigners
  );

  const postedMessageData = await coreBridge.PostedMessageV1.fromAccountAddress(
    connection,
    message!
  );
  const { nonce, consistencyLevel, payload } = expected;
  expectDeepEqual(postedMessageData, {
    consistencyLevel,
    emitterAuthority: PublicKey.default,
    status: coreBridge.MessageStatus.Unset,
    _gap0: Buffer.alloc(3),
    postedTimestamp: txDetails!.blockTime!,
    nonce,
    sequence,
    solanaChainId: 1,
    emitter,
    payload,
  });

  const emitterSequenceData = await coreBridge.EmitterSequence.fromPda(
    connection,
    program.programId,
    emitter!
  );
  expectDeepEqual(emitterSequenceData, { sequence: sequence.addn(1) });

  const config = await coreBridge.Config.fromPda(connection, program.programId);

  return { postedMessageData, emitterSequence, config };
}
