import { parseVaa } from "@certusone/wormhole-sdk";
import { GovernanceEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { expect } from "chai";
import {
  GUARDIAN_KEYS,
  InvalidAccountConfig,
  createIfNeeded,
  expectIxErr,
  expectIxOkDetails,
  invokeVerifySignaturesAndPostVaa,
  parallelPostVaa,
  range,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import { GOVERNANCE_EMITTER_ADDRESS } from "../helpers/coreBridge";

// Mock governance emitter and guardian.
const GUARDIAN_SET_INDEX = 0;
const GOVERNANCE_SEQUENCE = 1_014_000;
const governance = new GovernanceEmitter(
  GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
  GOVERNANCE_SEQUENCE - 1
);
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
        dataLength: 1,
        owner: program.programId,
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

    it("Cannot Invoke `guardian_set_update` with Same VAA", async () => {
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

    it("Invoke `guardian_set_update` Again to Set Original Guardian Keys", async () => {
      const newGuardianSetIndex = guardians.setIndex + 1;
      const newGuardianKeys = guardians.getPublicKeys();

      // Create the signed VAA.
      const signedVaa = defaultVaa(newGuardianSetIndex, newGuardianKeys, range(0, 2));

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
  });

  describe("More Verify Signatures and Post VAA Tests", () => {
    it.skip("Cannot Invoke `verify_signatures` on Expired Guardian Set", async () => {
      // TODO
    });

    it.skip("Invoke `verify_signatures` with New Guardian Set", async () => {
      // TODO
    });

    it.skip("Cannot Invoke `verify_signatures` with Different Guardian Set on Same Signature Set", async () => {
      // TODO
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
