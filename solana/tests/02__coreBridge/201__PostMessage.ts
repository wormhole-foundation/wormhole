import { BN, web3 } from "@coral-xyz/anchor";
import { expect } from "chai";
import { coreBridge } from "wormhole-solana-sdk";
import {
  COMMON_EMITTER,
  CORE_BRIDGE_PROGRAM_ID,
  LOCALHOST,
  airdrop,
  expectLegacyPostMessageErr,
  expectLegacyPostMessageOk,
  expectIxTransactionDetails,
  expectIxErr,
  expectIxOk,
} from "../helpers";
import { Transaction } from "@solana/web3.js";

describe("Core Bridge: New Post Message Features", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const payerSigner = web3.Keypair.generate();
  const payer = payerSigner.publicKey;

  let payerSequence = new BN(0);
  let commonEmitterSequence = new BN(10);

  const localVariables = new Map<string, any>();

  before("Airdrop Payer", async () => {
    await airdrop(connection, payer);
  });

  describe("Ok", async () => {
    it("Invoke `init_message_v1` with Large Message", async () => {
      const emitterSigner = COMMON_EMITTER;
      const messageSigner = web3.Keypair.generate();

      // Accounts.
      const accounts = coreBridge.InitMessageV1Context.new(
        emitterSigner.publicKey,
        messageSigner.publicKey
      );

      // Data.
      const expectedMsgLength = 30 * 1024;
      const args = { cpiProgramId: null };

      const dataLength = 95 + expectedMsgLength;
      const createIx = await connection
        .getMinimumBalanceForRentExemption(dataLength)
        .then((lamports) =>
          web3.SystemProgram.createAccount({
            fromPubkey: payer,
            newAccountPubkey: messageSigner.publicKey,
            space: dataLength,
            lamports,
            programId: new web3.PublicKey(CORE_BRIDGE_PROGRAM_ID),
          })
        );
      const initIx = await coreBridge.initMessageV1Ix(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        accounts,
        args
      );

      await expectIxOk(
        connection,
        [createIx, initIx],
        [payerSigner, emitterSigner, messageSigner]
      );

      const { emitterAuthority: emitter, draftMessage } = accounts;

      const messageData = await coreBridge.PostedMessageV1.fromAccountAddress(
        connection,
        draftMessage
      );
      expect(messageData.finality).equals(0);
      expect(messageData.emitterAuthority.equals(emitter)).is.true;
      expect(messageData.status).equals(coreBridge.MessageStatus.Writing);
      expect(messageData._gap0.equals(Buffer.alloc(3))).is.true;
      expect(messageData.postedTimestamp).equals(0);
      expect(messageData.nonce).equals(0);
      expect(messageData.sequence.eqn(0)).is.true;
      expect(messageData.solanaChainId).equals(1);
      expect(messageData.emitter.equals(emitter)).is.true;
      expect(messageData.payload).has.length(expectedMsgLength);

      // Save for later.
      localVariables.set("messageSigner", messageSigner);
      localVariables.set("draftMessage", draftMessage);
      localVariables.set("expectedMsgLength", expectedMsgLength);
    });

    it("Invoke `process_message_v1` With Large Message", async () => {
      const emitterSigner = COMMON_EMITTER;
      const draftMessage: web3.PublicKey = localVariables.get("draftMessage")!;
      const expectedMsgLength: number =
        localVariables.get("expectedMsgLength")!;

      const repeatedMessage = "All your base are belong to us. ";
      const messagePayload = Buffer.alloc(expectedMsgLength, repeatedMessage);
      let messageIndex = 0;

      const accounts = coreBridge.ProcessMessageV1Context.new(
        emitterSigner.publicKey,
        draftMessage,
        null
      );

      // Break up into chunks. Max chunk size is 914 (due to transaction size).
      const maxChunkSize = 914;
      while (messageIndex < messagePayload.length) {
        const dataLength = Math.min(
          messagePayload.length - messageIndex,
          maxChunkSize
        );
        const data = messagePayload.subarray(
          messageIndex,
          messageIndex + dataLength
        );

        const ix = await coreBridge.processMessageV1Ix(
          connection,
          CORE_BRIDGE_PROGRAM_ID,
          accounts,
          { write: { index: messageIndex, data } }
        );

        await expectIxOk(connection, [ix], [payerSigner, emitterSigner]);

        messageIndex += dataLength;
      }

      const messageData = await coreBridge.PostedMessageV1.fromAccountAddress(
        connection,
        draftMessage
      );
      expect(messageData.status).equals(coreBridge.MessageStatus.Writing);
      expect(messageData.payload.equals(messagePayload)).is.true;

      // Save for later.
      localVariables.set("messagePayload", messagePayload);
    });

    it("Invoke `legacy_post_message` With Processed Message", async () => {
      const emitterSigner = COMMON_EMITTER;
      const messageSigner: web3.Keypair = localVariables.get("messageSigner")!;
      const messagePayload: Buffer = localVariables.get("messagePayload")!;

      // Accounts.
      const accounts = coreBridge.LegacyPostMessageContext.new(
        CORE_BRIDGE_PROGRAM_ID,
        messageSigner.publicKey,
        emitterSigner.publicKey,
        payer,
        { clock: false, rent: false, feeCollector: true }
      );

      // And for the heck of it, show that we do not need these accounts.
      expect(accounts._clock).is.null;
      expect(accounts._rent).is.null;

      // Data.
      const nonce = 420;
      const payload = Buffer.alloc(0);
      const finalityRepr = 0;

      await expectLegacyPostMessageOk(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, emitterSigner, messageSigner],
        commonEmitterSequence,
        { actualPayload: messagePayload }
      );
      commonEmitterSequence.iaddn(1);
    });

    it("Cannot Invoke `legacy_post_message` With Same Processed Message", async () => {
      const emitterSigner = COMMON_EMITTER;
      const messageSigner: web3.Keypair = localVariables.get("messageSigner")!;

      // Accounts.
      const accounts = coreBridge.LegacyPostMessageContext.new(
        CORE_BRIDGE_PROGRAM_ID,
        messageSigner.publicKey,
        emitterSigner.publicKey,
        payer,
        { clock: false, rent: false, feeCollector: true }
      );

      // And for the heck of it, show that we do not need these accounts.
      expect(accounts._clock).is.null;
      expect(accounts._rent).is.null;

      // Data.
      const nonce = 420;
      const payload = Buffer.alloc(0);
      const finalityRepr = 0;

      await expectLegacyPostMessageErr(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, emitterSigner, messageSigner],
        "RequireKeysEqViolated"
      );
    });
  });
});
