import * as anchor from "@coral-xyz/anchor";
import {
  GUARDIAN_KEYS,
  expectIxErr,
  expectIxOkDetails,
  InvalidAccountConfig,
  verifySignaturesAndPostVaa,
  expectDeepEqual,
} from "../helpers";
import { GOVERNANCE_EMITTER_ADDRESS } from "../helpers/coreBridge";
import { parseVaa } from "@certusone/wormhole-sdk";
import { GovernanceEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as coreBridge from "../helpers/coreBridge";
import { expect } from "chai";

// Mock governance emitter and guardian.
const GUARDIAN_SET_INDEX = 0;
const GOVERNANCE_SEQUENCE = 2_005_000;
const governance = new GovernanceEmitter(
  GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
  GOVERNANCE_SEQUENCE - 1
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Instruction: Guardian Set Update", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = coreBridge.getAnchorProgram(
    connection,
    coreBridge.getProgramId("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o")
  );
  const payer = (provider.wallet as anchor.Wallet).payer;
  const forkedProgram = coreBridge.getAnchorProgram(
    connection,
    coreBridge.getProgramId("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth")
  );

  describe("Invalid Interaction", () => {});

  describe("Ok", () => {
    it("Invoke `guardianSetUpdate`", async () => {
      const newGuardianSetIndex = guardians.setIndex + 1;
      const newGuardianCount = 2;
      const newGuardianKeys = guardians.getPublicKeys().slice(0, newGuardianCount);

      // Create the signed VAA.
      const signedVaa = defaultVaa(newGuardianSetIndex, newGuardianKeys);

      // Invoke the instruction.
      const [txDetails, txForkDetails] = await parallelTxDetails(
        program,
        forkedProgram,
        {
          payer: payer.publicKey,
        },
        payer,
        signedVaa
      );

      // Validate bridge data account.
      await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      // Validate guardian data.
      const guardianSetData = await coreBridge.GuardianSet.fromPda(
        connection,
        program.programId,
        newGuardianSetIndex
      );
      for (let i = 0; i < newGuardianCount; i++) {
        expect(Buffer.from(guardianSetData.keys[i])).deep.equals(newGuardianKeys[i]);
      }
      expect(guardianSetData.index).equals(newGuardianSetIndex);
      expect(guardianSetData.creationTime).equals(parseVaa(signedVaa).timestamp);
      expect(guardianSetData.expirationTime).equals(0);

      // Validate guardian set accounts.
      await coreBridge.expectEqualGuardianSet(program, forkedProgram, newGuardianSetIndex);

      // Save Vaa to local variables.
      localVariables.set("signedVaa", signedVaa);
    });

    it("Cannot Invoke `guardianSetUpdate` with Same VAA", async () => {
      const signedVaa: Buffer = localVariables.get("signedVaa");

      // Invoke the instruction.
      await expectIxErr(
        connection,
        [
          coreBridge.legacyGuardianSetUpdateIx(
            program,
            { payer: payer.publicKey },
            parseVaa(signedVaa)
          ),
        ],
        [payer],
        "already in use"
      );
    });
  });
});

function defaultVaa(newIndex: number, newKeys: Buffer[]): Buffer {
  const timestamp = 4294967295;
  const published = governance.publishWormholeGuardianSetUpgrade(timestamp, newIndex, newKeys);
  return guardians.addSignatures(published, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

async function parallelTxDetails(
  program: coreBridge.CoreBridgeProgram,
  forkedProgram: coreBridge.CoreBridgeProgram,
  accounts: coreBridge.LegacyGuardianSetUpdateContext,
  payer: anchor.web3.Keypair,
  signedVaa: Buffer
) {
  const connection = program.provider.connection;

  // Parse the signed VAA.
  const parsedVaa = parseVaa(signedVaa);

  // Verify and Post
  await Promise.all([
    verifySignaturesAndPostVaa(program, payer, signedVaa),
    verifySignaturesAndPostVaa(forkedProgram, payer, signedVaa),
  ]);

  // Create the transferFees instruction.
  const ix = await coreBridge.legacyGuardianSetUpdateIx(program, accounts, parsedVaa);
  const forkedIx = await coreBridge.legacyGuardianSetUpdateIx(forkedProgram, accounts, parsedVaa);

  // Invoke the instruction.
  return Promise.all([
    expectIxOkDetails(connection, [ix], [payer]),
    expectIxOkDetails(connection, [forkedIx], [payer]),
  ]);
}
