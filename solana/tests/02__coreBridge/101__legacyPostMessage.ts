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
} from "../helpers";

describe("Core Bridge: Legacy Post Message", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const payerSigner = web3.Keypair.generate();
  const payer = payerSigner.publicKey;

  let payerSequence = new BN(0);
  let commonEmitterSequence = new BN(6);

  before("Airdrop Payer", async () => {
    await airdrop(connection, payer);
  });

  describe("Ok", async () => {
    it("Invoke `legacy_post_message` With Small Payload", async () => {
      const emitterSigner = COMMON_EMITTER;
      const messageSigner = web3.Keypair.generate();

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
      const payload = Buffer.from("All your base are belong to us.");
      const finalityRepr = 0;

      await expectLegacyPostMessageOk(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, emitterSigner, messageSigner],
        commonEmitterSequence
      );
      commonEmitterSequence.iaddn(1);
    });

    it("Invoke `legacy_post_message` Again With Same Emitter", async () => {
      const emitterSigner = COMMON_EMITTER;
      const messageSigner = web3.Keypair.generate();

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
      const nonce = 69;
      const payload = Buffer.from("Somebody set up us the bomb.");
      const finalityRepr = 1;

      await expectLegacyPostMessageOk(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, emitterSigner, messageSigner],
        commonEmitterSequence
      );
      commonEmitterSequence.iaddn(1);
    });

    it("Invoke `legacy_post_message` With Payer as Emitter", async () => {
      const messageSigner = web3.Keypair.generate();

      // Accounts.
      const accounts = coreBridge.LegacyPostMessageContext.new(
        CORE_BRIDGE_PROGRAM_ID,
        messageSigner.publicKey,
        payer, // emitter
        payer,
        { clock: false, rent: false, feeCollector: true }
      );

      // And for the heck of it, show that we do not need these accounts.
      expect(accounts._clock).is.null;
      expect(accounts._rent).is.null;

      // Data.
      const nonce = 420;
      const payload = Buffer.from("I'm the captain now.");
      const finalityRepr = 0;

      await expectLegacyPostMessageOk(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, messageSigner],
        payerSequence
      );
      payerSequence.iaddn(1);
    });

    it("Cannot Invoke `legacy_post_message` Without Fee Collector If Fee Exists", async () => {
      const emitterSigner = web3.Keypair.generate();
      const messageSigner = web3.Keypair.generate();

      // Accounts.
      const accounts = coreBridge.LegacyPostMessageContext.new(
        CORE_BRIDGE_PROGRAM_ID,
        messageSigner.publicKey,
        emitterSigner.publicKey,
        payer,
        { clock: false, rent: false, feeCollector: false }
      );

      // And for the heck of it, show that we do not need these accounts.
      expect(accounts._clock).is.null;
      expect(accounts._rent).is.null;
      expect(accounts.feeCollector).is.null;

      // Data.
      const nonce = 420;
      const payload = Buffer.from("All your base are belong to us.");
      const finalityRepr = 0;

      await expectLegacyPostMessageErr(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, emitterSigner, messageSigner],
        "AccountNotEnoughKeys"
      );
    });

    it("Cannot Invoke `legacy_post_message` Without Transferring Fee If Fee Exists", async () => {
      const emitterSigner = web3.Keypair.generate();
      const messageSigner = web3.Keypair.generate();

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
      const payload = Buffer.from("All your base are belong to us.");
      const finalityRepr = 0;

      await expectLegacyPostMessageErr(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, emitterSigner, messageSigner],
        "InsufficientMessageFee"
      );
    });
  });
});
