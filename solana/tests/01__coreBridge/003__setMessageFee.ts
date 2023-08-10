import * as anchor from "@coral-xyz/anchor";
import {
  GUARDIAN_KEYS,
  expectIxErr,
  expectIxOkDetails,
  InvalidAccountConfig,
  verifySignaturesAndPostVaa,
  parallelPostVaa,
} from "../helpers";
import { GOVERNANCE_EMITTER_ADDRESS } from "../helpers/coreBridge";
import { parseVaa } from "@certusone/wormhole-sdk";
import { GovernanceEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as coreBridge from "../helpers/coreBridge";
import { expect } from "chai";

// Mock governance emitter and guardian.
const GUARDIAN_SET_INDEX = 0;
const GOVERNANCE_SEQUENCE = 2_003_000;
const governance = new GovernanceEmitter(
  GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
  GOVERNANCE_SEQUENCE - 1
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

describe("Core Bridge -- Instruction: Set Message Fee", () => {
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

  // Test variables.
  const localVariables = new Map<string, any>();

  describe("Invalid Interaction", () => {
    const accountConfigs: InvalidAccountConfig[] = [
      {
        label: "bridge",
        contextName: "bridge",
        address: anchor.web3.Keypair.generate().publicKey,
        errorMsg: "AccountNotInitialized",
      },
      {
        label: "posted_vaa",
        contextName: "postedVaa",
        address: anchor.web3.Keypair.generate().publicKey,
        errorMsg: "AccountNotInitialized",
      },
      {
        label: "claim",
        contextName: "claim",
        address: anchor.web3.Keypair.generate().publicKey,
        errorMsg: "AccountNotInitialized",
      },
    ];

    for (const cfg of accountConfigs) {
      it(`Account: ${cfg.label} (${cfg.errorMsg})`, async () => {
        const accounts = { payer: payer.publicKey };
        accounts[cfg.contextName] = cfg.address;

        await expectIxErr(
          connection,
          [
            coreBridge.legacySetMessageFeeIx(
              program,
              accounts,
              parseVaa(defaultVaa(new anchor.BN(69)))
            ),
          ],
          [payer],
          cfg.errorMsg
        );
      });
    }
  });

  describe("Ok", () => {
    it("Invoke `setMessageFee`", async () => {
      // New fee amount.
      const amount = new anchor.BN(6969);

      // Fetch the bridge data before executing the instruciton to verify that the
      // new fee amount is different than the current fee amount.
      const bridgeDataBefore = await coreBridge.BridgeProgramData.fromPda(
        connection,
        program.programId
      );
      expect(bridgeDataBefore.config.feeLamports.toString()).to.not.equal(amount.toString());

      // Fetch default VAA.
      const signedVaa = defaultVaa(amount);

      // Set the message fee for both programs.
      await parallelTxDetails(program, forkedProgram, { payer: payer.publicKey }, signedVaa, payer);

      // Make sure the bridge accounts are the same.
      await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      // Verify that the message fee was set correctly. We only need to check one program
      // since we already verified that the bridge accounts are the same.
      const bridgeDataAfter = await coreBridge.BridgeProgramData.fromPda(
        connection,
        program.programId
      );
      expect(bridgeDataAfter.config.feeLamports.toString()).to.equal(amount.toString());

      // Save the VAA.
      localVariables.set("signedVaa", signedVaa);
    });

    it("Cannot Invoke `setMessageFee` with Same VAA", async () => {
      const signedVaa: Buffer = localVariables.get("signedVaa");

      await expectIxErr(
        connection,
        [
          coreBridge.legacySetMessageFeeIx(
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

function defaultVaa(amount: anchor.BN): Buffer {
  // Vaa info.
  const timestamp = 12345678;
  const chain = 1;
  const published = governance.publishWormholeSetMessageFee(
    timestamp,
    chain,
    BigInt(amount.toString())
  );
  return guardians.addSignatures(published, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

async function parallelTxDetails(
  program: coreBridge.CoreBridgeProgram,
  forkedProgram: coreBridge.CoreBridgeProgram,
  accounts: coreBridge.LegacySetMessageFeeContext,
  signedVaa: Buffer,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;

  // Post the VAAs.
  await parallelPostVaa(connection, payer, signedVaa);

  // Parse the VAA.
  const parsedVaa = parseVaa(signedVaa);

  // Create the set fee instructions.
  const ix = coreBridge.legacySetMessageFeeIx(program, accounts, parsedVaa);
  const forkedIx = coreBridge.legacySetMessageFeeIx(forkedProgram, accounts, parsedVaa);

  return Promise.all([
    expectIxOkDetails(connection, [ix], [payer]),
    expectIxOkDetails(connection, [forkedIx], [payer]),
  ]);
}
