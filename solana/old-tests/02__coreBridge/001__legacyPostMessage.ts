import { BN, web3 } from "@coral-xyz/anchor";
import { coreBridge } from "wormhole-solana-sdk";
import {
  COMMON_EMITTER,
  CORE_BRIDGE_PROGRAM_ID,
  LOCALHOST,
  airdrop,
  expectLegacyPostMessageErr,
  expectLegacyPostMessageOk,
} from "../helpers";
import { expect } from "chai";

describe("Core Bridge: Legacy Post Message", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const payerSigner = web3.Keypair.generate();
  const payer = payerSigner.publicKey;

  let payerSequence = new BN(0);
  let commonEmitterSequence = new BN(0);

  before("Airdrop Payer", async () => {
    await airdrop(connection, payer);
  });

  describe("Known Issues", async () => {
    it("Cannot Invoke `post_message` Without Clock Sysvar", async () => {
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
      expect(accounts._rent).is.null;

      // Data.
      const nonce = 420;
      const payload = Buffer.alloc(69);
      const finalityRepr = 0;

      // We must pay the fee collector prior publishing a message.
      const preInstructions = await Promise.all([
        coreBridge.transferMessageFeeIx(
          connection,
          CORE_BRIDGE_PROGRAM_ID,
          payer
        ),
      ]);

      await expectLegacyPostMessageErr(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, messageSigner],
        "InvalidSysvar",
        preInstructions
      );
    });
  });

  describe("Ok", async () => {
    it("Invoke `post_message` With Small Payload", async () => {
      const emitterSigner = COMMON_EMITTER;
      const messageSigner = web3.Keypair.generate();

      // Accounts.
      const accounts = coreBridge.LegacyPostMessageContext.new(
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

    it("Invoke `post_message` Again With Same Emitter", async () => {
      const emitterSigner = COMMON_EMITTER;
      const messageSigner = web3.Keypair.generate();

      // Accounts.
      const accounts = coreBridge.LegacyPostMessageContext.new(
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

      await expectLegacyPostMessageOk(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, emitterSigner, messageSigner],
        commonEmitterSequence
      );
      commonEmitterSequence.iaddn(1);
    });

    it("Invoke `post_message` With Payer as Emitter", async () => {
      const messageSigner = web3.Keypair.generate();

      // Accounts.
      const accounts = coreBridge.LegacyPostMessageContext.new(
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
  });
});
