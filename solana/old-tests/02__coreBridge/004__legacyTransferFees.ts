import { GovernanceEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as coreBridgeSDK from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";
import { web3 } from "@coral-xyz/anchor";
import { expect } from "chai";
import { coreBridge } from "wormhole-solana-sdk";
import {
  CORE_BRIDGE_PROGRAM_ID,
  GOVERNANCE_EMITTER_ADDRESS,
  GUARDIAN_KEYS,
  LOCALHOST,
  airdrop,
  expectIxErr,
  expectIxOk,
  invokeVerifySignaturesAndPostVaa,
} from "../helpers";

const GUARDIAN_SET_INDEX = 0;
const GOVERNANCE_SEQUENCE = 2_004_000;

describe("Core Bridge: Legacy Transfer Fees (Governance)", () => {
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
    it("Invoke `transfer_fees`", async () => {
      const recipient = web3.Keypair.generate().publicKey;
      {
        const balance = await connection.getBalance(recipient);
        expect(balance).equals(0);
      }

      // First send lamports over to the fee collector.
      const amount = 42069420;
      const transferIx = web3.SystemProgram.transfer({
        fromPubkey: payer,
        toPubkey: coreBridge.FeeCollector.address(CORE_BRIDGE_PROGRAM_ID),
        lamports: amount,
      });
      await expectIxOk(connection, [transferIx], [payerSigner]);

      const timestamp = 12345678;
      const chain = 1;
      const published = governance.publishWormholeTransferFees(
        timestamp,
        chain,
        BigInt(amount.toString()),
        recipient.toBuffer()
      );
      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
      );

      // Verify and Post
      await invokeVerifySignaturesAndPostVaa(connection, payerSigner, signedVaa);

      await expectIxOk(
        connection,
        [
          coreBridgeSDK.createTransferFeesInstruction(
            CORE_BRIDGE_PROGRAM_ID,
            payer,
            recipient,
            signedVaa
          ),
        ],
        [payerSigner]
      );

      {
        const balance = await connection.getBalance(recipient);
        expect(balance).equals(amount);
      }

      // Save for later.
      localVariables.set("signedVaa", signedVaa);
      localVariables.set("recipient", recipient);
    });

    it("Cannot Invoke `transfer_fees` with Same VAA", async () => {
      const signedVaa: Buffer = localVariables.get("signedVaa")!;
      const recipient: Buffer = localVariables.get("recipient")!;

      await expectIxErr(
        connection,
        [
          coreBridgeSDK.createTransferFeesInstruction(
            CORE_BRIDGE_PROGRAM_ID,
            payer,
            recipient,
            signedVaa
          ),
        ],
        [payerSigner],
        "AlreadyInitialized"
      );
    });
  });
});
