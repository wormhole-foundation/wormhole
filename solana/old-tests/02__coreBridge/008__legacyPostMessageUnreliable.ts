import { BN, web3 } from "@coral-xyz/anchor";
import { expect } from "chai";
import { coreBridge } from "wormhole-solana-sdk";
import {
  COMMON_EMITTER,
  COMMON_UNRELIABLE_MESSAGE_SIGNER,
  CORE_BRIDGE_PROGRAM_ID,
  LOCALHOST,
  airdrop,
  expectLegacyPostMessageUnreliableErr,
  expectLegacyPostMessageUnreliableOk,
} from "../helpers";

describe("Core Bridge: Legacy Post Message Unreliable", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const payerSigner = web3.Keypair.generate();
  const payer = payerSigner.publicKey;

  let payerSequence = new BN(0);
  let commonEmitterSequence = new BN(2);

  before("Airdrop Payer", async () => {
    await airdrop(connection, payer);
  });

  describe("Known Issues", async () => {
    it("Cannot Invoke `post_message_unreliable` Without Clock Sysvar", async () => {
      const messageSigner = web3.Keypair.generate();

      // Accounts.
      const accounts = coreBridge.LegacyPostMessageUnreliableContext.new(
        CORE_BRIDGE_PROGRAM_ID,
        messageSigner.publicKey,
        payer, // emitter
        payer,
        { clock: false, rent: false, feeCollector: true }
      );

      // And for the heck of it, show that we do not need these accounts.
      expect(accounts._rent).is.null;

      // Data.
      const nonce = 420;
      const payload = Buffer.alloc(803);
      const finalityRepr = 0;

      // We must pay the fee collector prior publishing a message.
      const preInstructions = await Promise.all([
        coreBridge.transferMessageFeeIx(
          connection,
          CORE_BRIDGE_PROGRAM_ID,
          payer
        ),
      ]);

      await expectLegacyPostMessageUnreliableErr(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, messageSigner],
        "Transaction too large: 1233 > 1232",
        preInstructions
      );
    });
  });

  describe("Ok", async () => {
    it("Invoke `post_message_unreliable` with Common Message Signer", async () => {
      const emitterSigner = COMMON_EMITTER;
      const messageSigner = COMMON_UNRELIABLE_MESSAGE_SIGNER;

      // Accounts.
      const accounts = coreBridge.LegacyPostMessageUnreliableContext.new(
        CORE_BRIDGE_PROGRAM_ID,
        messageSigner.publicKey,
        emitterSigner.publicKey,
        payer,
        { clock: true, rent: false, feeCollector: true }
      );

      // And for the heck of it, show that we do not need these accounts.
      expect(accounts._rent).is.null;

      // Data.
      const nonce = 420;
      const finalityRepr = 0;
      const payload = Buffer.alloc(64);
      payload.write("All your base are belong to us.", 0);

      await expectLegacyPostMessageUnreliableOk(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, emitterSigner, messageSigner],
        commonEmitterSequence
      );
      commonEmitterSequence.iaddn(1);

      // Send again but with different payload and different finality.
      const newFinalityRepr = 1;
      payload.fill(0);
      payload.write("Just kidding.", 0);

      await expectLegacyPostMessageUnreliableOk(
        connection,
        accounts,
        { nonce, payload, finalityRepr: newFinalityRepr },
        [payerSigner, emitterSigner, messageSigner],
        commonEmitterSequence
      );
      commonEmitterSequence.iaddn(1);
    });

    it("Cannot Invoke `legacy_post_message_unreliable` using Different Length Payload", async () => {
      const emitterSigner = COMMON_EMITTER;
      const messageSigner = COMMON_UNRELIABLE_MESSAGE_SIGNER;

      // Accounts.
      const accounts = coreBridge.LegacyPostMessageUnreliableContext.new(
        CORE_BRIDGE_PROGRAM_ID,
        messageSigner.publicKey,
        emitterSigner.publicKey,
        payer,
        { clock: true, rent: false, feeCollector: true }
      );

      // And for the heck of it, show that we do not need these accounts.
      expect(accounts._rent).is.null;

      // Data.
      const nonce = 420;
      const payload = Buffer.from("Womp womp.");
      const finalityRepr = 0;

      // We must pay the fee collector prior publishing a message.
      const preInstructions = await Promise.all([
        coreBridge.transferMessageFeeIx(
          connection,
          CORE_BRIDGE_PROGRAM_ID,
          payer
        ),
      ]);

      await expectLegacyPostMessageUnreliableErr(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, emitterSigner, messageSigner],
        "Error: Custom(18)", // InvalidPayloadLength
        preInstructions
      );
    });

    it("Invoke `legacy_post_message_unreliable` using Old Message", async () => {
      const emitterSigner = COMMON_EMITTER;
      const messageSigner = COMMON_UNRELIABLE_MESSAGE_SIGNER;

      // Accounts.
      const message = messageSigner.publicKey;
      const accounts = coreBridge.LegacyPostMessageUnreliableContext.new(
        CORE_BRIDGE_PROGRAM_ID,
        message,
        emitterSigner.publicKey,
        payer,
        { clock: true, rent: false, feeCollector: true }
      );

      // And for the heck of it, show that we do not need these accounts.
      expect(accounts._rent).is.null;

      // Fetch existing message.
      const [existingFinalityRepr, existingNonce, existingPayload] =
        await coreBridge.PostedMessageV1Unreliable.fromAccountAddress(
          connection,
          message
        ).then((msg): [number, number, Buffer] => [
          msg.finality,
          msg.nonce,
          msg.payload,
        ]);

      const nonce = 69;
      expect(nonce).not.equals(existingNonce);

      const finalityRepr = 0;
      expect(finalityRepr).not.equals(existingFinalityRepr);

      const payload = Buffer.alloc(existingPayload.length);
      payload.fill(0);
      payload.write("So fresh and so clean clean.");
      expect(payload.equals(existingPayload)).is.false;

      await expectLegacyPostMessageUnreliableOk(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, emitterSigner, messageSigner],
        commonEmitterSequence
      );
      commonEmitterSequence.iaddn(1);
    });

    it("Invoke `legacy_post_message_unreliable` with New Message", async () => {
      const emitterSigner = COMMON_EMITTER;
      const messageSigner = web3.Keypair.generate();

      // Accounts.
      const accounts = coreBridge.LegacyPostMessageUnreliableContext.new(
        CORE_BRIDGE_PROGRAM_ID,
        messageSigner.publicKey,
        emitterSigner.publicKey,
        payer,
        { clock: true, rent: false, feeCollector: true }
      );

      // And for the heck of it, show that we do not need these accounts.
      expect(accounts._rent).is.null;

      // Data.
      const nonce = 69;
      const payload = Buffer.from("Somebody set up us the bomb.");
      const finalityRepr = 1;

      await expectLegacyPostMessageUnreliableOk(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, emitterSigner, messageSigner],
        commonEmitterSequence
      );
      commonEmitterSequence.iaddn(1);
    });

    it("Invoke `legacy_post_message_unreliable` with Payer as Emitter", async () => {
      const messageSigner = web3.Keypair.generate();

      // Accounts.
      const accounts = coreBridge.LegacyPostMessageUnreliableContext.new(
        CORE_BRIDGE_PROGRAM_ID,
        messageSigner.publicKey,
        payer, // emitter
        payer,
        { clock: true, rent: false, feeCollector: true }
      );

      // And for the heck of it, show that we do not need these accounts.
      expect(accounts._rent).is.null;

      // Data.
      const nonce = 420;
      const payload = Buffer.from("I'm the captain now.");
      const finalityRepr = 1;

      await expectLegacyPostMessageUnreliableOk(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, messageSigner],
        payerSequence
      );
      payerSequence.iaddn(1);
    });
  });
});