import { expect } from "chai";
import { BridgeProgramData, CoreBridgeProgram } from ".";
import * as coreBridge from "../coreBridge";
import { expectDeepEqual } from "../utils";
import { Keypair, PublicKey, VersionedTransactionResponse } from "@solana/web3.js";
import { BN } from "@coral-xyz/anchor";

export async function expectEqualBridgeAccounts(
  program: CoreBridgeProgram,
  forkedProgram: CoreBridgeProgram
) {
  const connection = program.provider.connection;

  const [bridgeData, forkBridgeData] = await Promise.all([
    BridgeProgramData.fromPda(connection, program.programId),
    BridgeProgramData.fromPda(connection, forkedProgram.programId),
  ]);
  expectDeepEqual(bridgeData, forkBridgeData);
}

export async function expectEqualMessageAccounts(
  program: CoreBridgeProgram,
  messageSigner: Keypair,
  forkedMessageSigner: Keypair,
  unreliable: boolean
) {
  const connection = program.provider.connection;

  if (unreliable) {
    const [messageData, forkedMessageData] = await Promise.all([
      coreBridge.PostedMessageV1Unreliable.fromAccountAddress(connection, messageSigner.publicKey),
      coreBridge.PostedMessageV1Unreliable.fromAccountAddress(
        connection,
        forkedMessageSigner.publicKey
      ),
    ]);
    expectDeepEqual(messageData, forkedMessageData);
  } else {
    const [messageData, forkedMessageData] = await Promise.all([
      coreBridge.PostedMessageV1.fromAccountAddress(connection, messageSigner.publicKey),
      coreBridge.PostedMessageV1.fromAccountAddress(connection, forkedMessageSigner.publicKey),
    ]);
    expectDeepEqual(messageData, forkedMessageData);
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
  const { nonce, payload, finality } = args;

  const {
    status,
    finality: msgFinality,
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

  expect(msgFinality).equals(finality === 0 ? 1 : 32);
  expect(emitterAuthority.equals(PublicKey.default)).is.true;
  expect(status).equals(coreBridge.MessageStatus.Unset);
  expect(_gap0.equals(Buffer.alloc(3))).is.true;
  expect(postedTimestamp).equals(txDetails.blockTime!);
  expect(messageNonce).equals(nonce);
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
    program.programId,
    emitter
  ).then((tracker) => tracker.sequence);
  expect(emitterSequenceValue.eq(expectedSequence.addn(1))).is.true;
}
