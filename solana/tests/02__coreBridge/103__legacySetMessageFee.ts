import { BN, web3 } from "@coral-xyz/anchor";
import {
  CORE_BRIDGE_PROGRAM_ID,
  GOVERNANCE_EMITTER_ADDRESS,
  GUARDIAN_KEYS,
  LOCALHOST,
  airdrop,
  expectLegacyPostMessageOk,
  expectIxErr,
  expectIxOk,
  verifySignaturesAndPostVaa,
} from "../helpers";
import {
  GovernanceEmitter,
  MockGuardians,
} from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as coreBridgeSDK from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";
import { coreBridge } from "wormhole-solana-sdk";
import { expect } from "chai";

const GUARDIAN_SET_INDEX = 3;
const GOVERNANCE_SEQUENCE = 2_103_000;

describe("Core Bridge: Legacy Set Message Fee (Governance)", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const governance = new GovernanceEmitter(
    GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
    GOVERNANCE_SEQUENCE
  );
  const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

  const payerSigner = web3.Keypair.generate();
  const payer = payerSigner.publicKey;

  const localVariables = new Map<string, any>();

  before("Airdrop Payer", async () => {
    await airdrop(connection, payer);
  });

  describe("Ok", async () => {
    it("Invoke `set_message_fee` To Set Fee == 0", async () => {
      const amount = new BN(0);

      const timestamp = 12345678;
      const chain = 1;
      const published = governance.publishWormholeSetMessageFee(
        timestamp,
        chain,
        BigInt(amount.toString())
      );
      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
      );

      // Verify and Post
      await verifySignaturesAndPostVaa(connection, payerSigner, signedVaa);

      // Set message fee.
      await expectIxOk(
        connection,
        [
          coreBridgeSDK.createSetFeesInstruction(
            CORE_BRIDGE_PROGRAM_ID,
            payer,
            signedVaa
          ),
        ],
        [payerSigner]
      );

      // TODO: Check bridge program data to see if message fee was set correctly.

      localVariables.set("signedVaa", signedVaa);
    });

    it("Cannot Invoke `set_message_fee` with Same VAA", async () => {
      const signedVaa: Buffer = localVariables.get("signedVaa")!;

      await expectIxErr(
        connection,
        [
          coreBridgeSDK.createSetFeesInstruction(
            CORE_BRIDGE_PROGRAM_ID,
            payer,
            signedVaa
          ),
        ],
        [payerSigner],
        "already in use"
      );
    });

    it("Invoke `legacy_post_message` Without Requiring Fee Collector Pubkey", async () => {
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

      await expectLegacyPostMessageOk(
        connection,
        accounts,
        { nonce, payload, finalityRepr },
        [payerSigner, emitterSigner, messageSigner],
        new BN(0)
      );
    });

    it("Invoke `set_message_fee` To Set Fee == 42069", async () => {
      const amount = new BN(42069);

      const timestamp = 12345678;
      const chain = 1;
      const published = governance.publishWormholeSetMessageFee(
        timestamp,
        chain,
        BigInt(amount.toString())
      );
      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
      );

      // Verify and Post
      await verifySignaturesAndPostVaa(connection, payerSigner, signedVaa);

      // Set message fee.
      await expectIxOk(
        connection,
        [
          coreBridgeSDK.createSetFeesInstruction(
            CORE_BRIDGE_PROGRAM_ID,
            payer,
            signedVaa
          ),
        ],
        [payerSigner]
      );

      // TODO: Check bridge program data to see if message fee was set correctly.
    });
  });
});
