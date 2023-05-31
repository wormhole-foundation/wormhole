import {
  GovernanceEmitter,
  MockGuardians,
} from "@certusone/wormhole-sdk/lib/cjs/mock";
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
  verifySignaturesAndPostVaa,
} from "../helpers";

const GUARDIAN_SET_INDEX = 0;
const GOVERNANCE_SEQUENCE = 2_006_000;

describe("Core Bridge: Legacy Guardian Set Update (Governance)", () => {
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

  describe("Known Issues", async () => {
    it("Invoke Core Bridge Governance Without Latest Guardian Set", async () => {
      const recipient = web3.Keypair.generate().publicKey;
      const amount = 42069420;

      // Post transfer fees VAA, which will be signed with a soon-to-be-expired Guardian set.
      const signedTransferFeesVaa = await (async () => {
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
        await verifySignaturesAndPostVaa(connection, payerSigner, signedVaa);

        return signedVaa;
      })();

      // Update guardian set.
      {
        const newGuardianSetIndex = guardians.setIndex + 1;
        const newGuardianKeys = guardians.getPublicKeys().slice(0, 1);

        const timestamp = 12345678;
        const published = governance.publishWormholeGuardianSetUpgrade(
          timestamp,
          newGuardianSetIndex,
          newGuardianKeys
        );
        const signedVaa = guardians.addSignatures(
          published,
          [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
        );

        // Verify and Post
        await verifySignaturesAndPostVaa(connection, payerSigner, signedVaa);

        await expectIxOk(
          connection,
          [
            coreBridgeSDK.createUpgradeGuardianSetInstruction(
              CORE_BRIDGE_PROGRAM_ID,
              payer,
              signedVaa
            ),
          ],
          [payerSigner]
        );

        // Update mock guardians.
        guardians.updateGuardianSetIndex(newGuardianSetIndex);
      }

      // Now use previously posted VAA to transfer fees.
      {
        {
          const balance = await connection.getBalance(recipient);
          expect(balance).equals(0);
        }

        // First send lamports over to the fee collector.
        const transferIx = web3.SystemProgram.transfer({
          fromPubkey: payer,
          toPubkey: coreBridge.FeeCollector.address(CORE_BRIDGE_PROGRAM_ID),
          lamports: amount,
        });
        await expectIxOk(connection, [transferIx], [payerSigner]);

        await expectIxOk(
          connection,
          [
            coreBridgeSDK.createTransferFeesInstruction(
              CORE_BRIDGE_PROGRAM_ID,
              payer,
              recipient,
              signedTransferFeesVaa
            ),
          ],
          [payerSigner]
        );

        {
          const balance = await connection.getBalance(recipient);
          expect(balance).equals(amount);
        }
      }
    });
  });

  describe("Ok", async () => {
    it("Invoke `guardian_set_update`", async () => {
      const newGuardianSetIndex = guardians.setIndex + 1;
      const newGuardianKeys = guardians.getPublicKeys().slice(0, 2);

      const timestamp = 12345678;
      const published = governance.publishWormholeGuardianSetUpgrade(
        timestamp,
        newGuardianSetIndex,
        newGuardianKeys
      );
      const signedVaa = guardians.addSignatures(published, [0]);

      // Verify and Post
      await verifySignaturesAndPostVaa(connection, payerSigner, signedVaa);

      await expectIxOk(
        connection,
        [
          coreBridgeSDK.createUpgradeGuardianSetInstruction(
            CORE_BRIDGE_PROGRAM_ID,
            payer,
            signedVaa
          ),
        ],
        [payerSigner]
      );

      // Update mock guardians.
      guardians.updateGuardianSetIndex(newGuardianSetIndex);

      // TODO: Verify guardian set.

      // Save for later.
      localVariables.set("signedVaa", signedVaa);
    });

    it("Cannot Invoke `guardian_set_update` with Same VAA", async () => {
      const signedVaa: Buffer = localVariables.get("signedVaa")!;

      await expectIxErr(
        connection,
        [
          coreBridgeSDK.createUpgradeGuardianSetInstruction(
            CORE_BRIDGE_PROGRAM_ID,
            payer,
            signedVaa
          ),
        ],
        [payerSigner],
        "AlreadyInitialized"
      );
    });

    it("Invoke `guardian_set_update` Again to Original Guardian Keys", async () => {
      const newGuardianSetIndex = guardians.setIndex + 1;
      const newGuardianKeys = guardians.getPublicKeys();

      const timestamp = 12345678;
      const published = governance.publishWormholeGuardianSetUpgrade(
        timestamp,
        newGuardianSetIndex,
        newGuardianKeys
      );
      const signedVaa = guardians.addSignatures(published, [0, 1]);

      // Verify and Post
      await verifySignaturesAndPostVaa(connection, payerSigner, signedVaa);

      await expectIxOk(
        connection,
        [
          coreBridgeSDK.createUpgradeGuardianSetInstruction(
            CORE_BRIDGE_PROGRAM_ID,
            payer,
            signedVaa
          ),
        ],
        [payerSigner]
      );

      // Update mock guardians.
      guardians.updateGuardianSetIndex(newGuardianSetIndex);

      // TODO: Verify guardian set.
    });
  });
});
