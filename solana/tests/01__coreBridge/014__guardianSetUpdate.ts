import { parseVaa } from "@certusone/wormhole-sdk";
import {
  GovernanceEmitter,
  MockEmitter,
  MockGuardians,
} from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { expect } from "chai";
import {
  ETHEREUM_DEADBEEF_TOKEN_ADDRESS,
  GUARDIAN_KEYS,
  InvalidAccountConfig,
  SignatureSets,
  createIfNeeded,
  createInvalidCoreGovernanceVaaFromEth,
  createSigVerifyIx,
  expectIxErr,
  expectIxOkDetails,
  invokeVerifySignaturesAndPostVaa,
  parallelPostVaa,
  range,
  sleep,
  GOVERNANCE_EMITTER_ADDRESS,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";

// Mock governance emitter and guardian.
const GUARDIAN_SET_INDEX = 0;
const GOVERNANCE_SEQUENCE = 1_014_000;
const governance = new GovernanceEmitter(
  GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
  GOVERNANCE_SEQUENCE - 1
);
const dummyEmitter = new MockEmitter(Buffer.alloc(32, "deadbeef").toString("hex"), 69, -1);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Legacy Instruction: Guardian Set Update", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;
  const forkedProgram = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  describe("Invalid Interaction", () => {
    const accountConfigs: InvalidAccountConfig[] = [
      {
        label: "config",
        contextName: "config",
        errorMsg: "ConstraintSeeds",
        dataLength: 24,
        owner: program.programId,
      },
      {
        label: "claim",
        contextName: "claim",
        errorMsg: "ConstraintSeeds",
      },
    ];

    for (const cfg of accountConfigs) {
      it(`Account: ${cfg.label} (${cfg.errorMsg})`, async () => {
        const accounts = await createIfNeeded(program.provider.connection, cfg, payer, {
          payer: payer.publicKey,
        } as coreBridge.LegacyGuardianSetUpdateContext);

        const signedVaa = defaultVaa(
          guardians.setIndex + 1,
          guardians.getPublicKeys().slice(0, 1),
          range(0, 13)
        );
        await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

        await expectIxErr(
          connection,
          [coreBridge.legacyGuardianSetUpdateIx(program, accounts, parseVaa(signedVaa))],
          [payer],
          cfg.errorMsg
        );
      });
    }
  });

  describe("Ok", () => {
    it("Invoke `guardian_set_update`", async () => {
      const newGuardianSetIndex = guardians.setIndex + 1;
      const newGuardianKeys = guardians.getPublicKeys().slice(0, 2);

      // Create the signed VAA.
      const signedVaa = defaultVaa(newGuardianSetIndex, newGuardianKeys, range(0, 13));

      // Invoke the instruction.
      const txDetails = await parallelTxDetails(
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
      const newGuardianSetData = await coreBridge.GuardianSet.fromPda(
        connection,
        program.programId,
        newGuardianSetIndex
      );
      for (let i = 0; i < newGuardianKeys.length; i++) {
        expect(Buffer.from(newGuardianSetData.keys[i])).deep.equals(newGuardianKeys[i]);
      }
      expect(newGuardianSetData.index).equals(newGuardianSetIndex);
      expect(newGuardianSetData.creationTime).equals(parseVaa(signedVaa).timestamp);
      expect(newGuardianSetData.expirationTime).equals(0);

      // Validate guardian set accounts.
      await coreBridge.expectEqualGuardianSet(program, forkedProgram, newGuardianSetIndex);

      // Update mock guardians.
      guardians.updateGuardianSetIndex(newGuardianSetIndex);

      // Save Vaa to local variables.
      localVariables.set("signedVaa", signedVaa);
    });
  });

  describe("New implementation", () => {
    const currSetRange = range(0, 2);

    it("Cannot Invoke `guardian_set_update` with Zero Address Key", async () => {
      let newGuardians = guardians.getPublicKeys();

      // Set the second key to the zero address.
      newGuardians[1] = Buffer.alloc(32);

      // Create the signed VAA.
      const signedVaa = defaultVaa(guardians.setIndex + 1, newGuardians, currSetRange);

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyGuardianSetUpdateIx(
        program,
        { payer: payer.publicKey },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "GuardianZeroAddress");
    });

    it("Cannot Invoke `guardian_set_update` with One Guardian and Its Address is Zero Address", async () => {
      const newGuardians = [Buffer.alloc(32)];

      // Create the signed VAA.
      const signedVaa = defaultVaa(guardians.setIndex + 1, newGuardians, currSetRange);

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyGuardianSetUpdateIx(
        program,
        { payer: payer.publicKey },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "GuardianZeroAddress");
    });

    it("Cannot Invoke `guardian_set_update` with Zero Address As Last Key", async () => {
      let newGuardians = guardians.getPublicKeys();

      // Set the second key to the zero address.
      newGuardians[newGuardians.length - 1] = Buffer.alloc(32);

      // Create the signed VAA.
      const signedVaa = defaultVaa(guardians.setIndex + 1, newGuardians, currSetRange);

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyGuardianSetUpdateIx(
        program,
        { payer: payer.publicKey },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "GuardianZeroAddress");
    });

    it("Cannot Invoke `guardian_set_update` with Duplicate Key", async () => {
      let newGuardians = guardians.getPublicKeys();

      // Duplicate the first key
      newGuardians[1] = newGuardians[0];

      // Create the signed VAA.
      const signedVaa = defaultVaa(guardians.setIndex + 1, newGuardians, currSetRange);

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyGuardianSetUpdateIx(
        program,
        { payer: payer.publicKey },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "DuplicateGuardianAddress");
    });

    it("Cannot Invoke `guardian_set_update` with Invalid Guardian Set Index", async () => {
      const signedVaa = defaultVaa(guardians.setIndex + 2, guardians.getPublicKeys(), currSetRange);

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyGuardianSetUpdateIx(
        program,
        { payer: payer.publicKey },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "InvalidGuardianSetIndex");
    });

    it("Cannot Invoke `guardian_set_update` with Invalid Governance Emitter", async () => {
      // Create a bad governance emitter.
      const governance = new GovernanceEmitter(
        Buffer.from(ETHEREUM_DEADBEEF_TOKEN_ADDRESS).toString("hex"),
        GOVERNANCE_SEQUENCE - 1
      );
      const invalidGuardians = new MockGuardians(guardians.setIndex, GUARDIAN_KEYS);

      // Vaa info.
      const timestamp = 294967295;
      const published = governance.publishWormholeGuardianSetUpgrade(
        timestamp,
        guardians.setIndex + 1,
        guardians.getPublicKeys()
      );
      const signedVaa = invalidGuardians.addSignatures(published, currSetRange);

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyGuardianSetUpdateIx(
        program,
        { payer: payer.publicKey },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "InvalidGovernanceEmitter");
    });

    it("Cannot Invoke `guardian_set_update` with Invalid Governance Action", async () => {
      // Vaa info.
      const timestamp = 12345678;
      const chain = 1;

      // Publish the wrong VAA type.
      const published = governance.publishWormholeSetMessageFee(timestamp, chain, BigInt(69));

      const signedVaa = guardians.addSignatures(published, currSetRange);

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyGuardianSetUpdateIx(
        program,
        { payer: payer.publicKey },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "InvalidGovernanceAction");
    });

    it("Cannot Invoke `guardian_set_update` with Invalid Governance Vaa", async () => {
      const signedVaa = createInvalidCoreGovernanceVaaFromEth(
        guardians,
        currSetRange,
        GOVERNANCE_SEQUENCE + 200,
        {
          governanceModule: Buffer.from(
            "00000000000000000000000000000000000000000000000000000000deadbeef",
            "hex"
          ),
        }
      );

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyGuardianSetUpdateIx(
        program,
        { payer: payer.publicKey },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "InvalidGovernanceVaa");
    });

    it("Cannot Invoke `verify_signatures` on Expired Guardian Set", async () => {
      const oldSetRange = [0];

      // Sleep for 5 seconds to expire the guardian set.
      await sleep(5000);

      // Make sure the guardian set was updated before this test.
      expect(GUARDIAN_SET_INDEX != guardians.setIndex).to.be.true;

      // Create sigVerify instruction.
      const message = Buffer.from("Ello M8");
      const sigVerifyIx = await createSigVerifyIx(
        program,
        guardians,
        GUARDIAN_SET_INDEX, // Use the old guardian set index.
        message,
        oldSetRange
      );

      const signerIndices = new Array(19).fill(-1);
      let count = 0;
      for (const i of oldSetRange) {
        signerIndices[i] = count;
        ++count;
      }

      // Create verify instruction.
      const { signatureSet } = new SignatureSets();
      const guardianSet = coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX);

      const verifyIx = coreBridge.legacyVerifySignaturesIx(
        program,
        { payer: payer.publicKey, guardianSet, signatureSet: signatureSet.publicKey },
        {
          signerIndices,
        }
      );

      await expectIxErr(
        connection,
        [sigVerifyIx, verifyIx],
        [payer, signatureSet],
        "GuardianSetExpired"
      );
    });

    it.skip("Cannot Invoke `verify_signatures` with Different Guardian Set on Same Signature Set", async () => {
      // TODO
    });

    it("Invoke `guardian_set_update` Again to Set Original Guardian Keys", async () => {
      const newGuardianSetIndex = guardians.setIndex + 1;
      const newGuardianKeys = guardians.getPublicKeys();

      // Create the signed VAA.
      const signedVaa = defaultVaa(newGuardianSetIndex, newGuardianKeys, range(0, 2));

      // Invoke the instruction.
      await parallelTxDetails(
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

      // Validate guardian set data.
      const newGuardianSetData = await coreBridge.GuardianSet.fromPda(
        connection,
        program.programId,
        newGuardianSetIndex
      );
      for (let i = 0; i < newGuardianKeys.length; i++) {
        expect(Buffer.from(newGuardianSetData.keys[i])).deep.equals(newGuardianKeys[i]);
      }
      expect(newGuardianSetData.index).equals(newGuardianSetIndex);
      expect(newGuardianSetData.creationTime).equals(parseVaa(signedVaa).timestamp);
      expect(newGuardianSetData.expirationTime).equals(0);

      // Validate the new guardian set accounts.
      await coreBridge.expectEqualGuardianSet(program, forkedProgram, newGuardianSetIndex);

      // Update mock guardians.
      guardians.updateGuardianSetIndex(newGuardianSetIndex);
    });

    it("Cannot Invoke `guardian_set_update` with Same VAA", async () => {
      const signedVaa = localVariables.get("signedVaa") as Buffer;

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
        "ConstraintSeeds"
      );
    });
  });
});

function defaultVaa(newIndex: number, newKeys: Buffer[], keyRange: number[]): Buffer {
  const timestamp = 294967295;
  const published = governance.publishWormholeGuardianSetUpgrade(timestamp, newIndex, newKeys);
  return guardians.addSignatures(published, keyRange);
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
  await parallelPostVaa(connection, payer, signedVaa);

  // Create the transferFees instruction.
  const ix = coreBridge.legacyGuardianSetUpdateIx(program, accounts, parsedVaa);
  const forkedIx = coreBridge.legacyGuardianSetUpdateIx(forkedProgram, accounts, parsedVaa);

  // Invoke the instruction.
  return expectIxOkDetails(connection, [ix, forkedIx], [payer]);
}
