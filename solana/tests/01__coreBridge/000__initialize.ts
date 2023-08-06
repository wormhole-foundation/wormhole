import * as anchor from "@coral-xyz/anchor";
import { expect } from "chai";
import { ethers } from "ethers";
import {
  GUARDIAN_KEYS,
  InvalidAccountConfig,
  expectDeepEqual,
  expectIxErr,
  expectIxOkDetails,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";

// TODO: Need to add negative tests for GuardianZeroAddress, DuplicateGuardians, etc.

describe("Core Bridge -- Legacy Instruction: Initialize", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;
  const forkedProgram = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  describe("Invalid Interaction", () => {
    const accountConfigs: InvalidAccountConfig[] = [
      {
        label: "bridge",
        contextName: "bridge",
        owner: anchor.web3.Keypair.generate().publicKey,
        errorMsg: "ConstraintSeeds",
      },
      {
        label: "guardian_set",
        contextName: "guardianSet",
        owner: anchor.web3.Keypair.generate().publicKey,
        errorMsg: "ConstraintSeeds",
      },
      {
        label: "fee_collector",
        contextName: "feeCollector",
        owner: anchor.web3.Keypair.generate().publicKey,
        errorMsg: "ConstraintSeeds",
      },
    ];

    for (const cfg of accountConfigs) {
      it(`Account: ${cfg.label} (${cfg.errorMsg})`, async () => {
        const accounts = { payer: payer.publicKey };
        accounts[cfg.contextName] = cfg.owner;
        const ix = coreBridge.legacyInitializeIx(program, accounts, defaultArgs());
        await expectIxErr(connection, [ix], [payer], cfg.errorMsg);
      });
    }
  });

  describe("Ok", () => {
    it("Cannot Invoke `initialize` With an Empty Guardian Set", async () => {
      const args = defaultArgs();
      args["initialGuardians"] = [];
      const ix = coreBridge.legacyInitializeIx(program, { payer: payer.publicKey }, args);
      await expectIxErr(connection, [ix], [payer], "ZeroGuardians");
    });

    it("Cannot Invoke `initialize` With Zero Address Guardian", async () => {
      const args = defaultArgs();
      args["initialGuardians"][0] = new Array(20).fill(0);
      const ix = coreBridge.legacyInitializeIx(program, { payer: payer.publicKey }, args);
      await expectIxErr(connection, [ix], [payer], "GuardianZeroAddress");
    });

    it("Cannot Invoke `initialize` With Duplicate Guardian Key", async () => {
      const args = defaultArgs();
      args["initialGuardians"][0] = args["initialGuardians"][12];
      const ix = coreBridge.legacyInitializeIx(program, { payer: payer.publicKey }, args);
      await expectIxErr(connection, [ix], [payer], "DuplicateGuardianAddress");
    });

    it("Invoke `initialize`", async () => {
      const { guardianSetTtlSeconds, feeLamports, initialGuardians } = defaultArgs();

      const [txDetails, forkTxDetails] = await parallelTxDetails(
        program,
        forkedProgram,
        { payer: payer.publicKey },
        { guardianSetTtlSeconds, feeLamports, initialGuardians },
        payer
      );

      await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      const guardianSetData = await coreBridge.GuardianSet.fromPda(
        connection,
        program.programId,
        0
      );
      const expectedGuardianSetData: coreBridge.GuardianSet = {
        index: 0,
        keys: initialGuardians,
        creationTime: txDetails?.blockTime,
        expirationTime: 0,
      };
      expectDeepEqual(guardianSetData, expectedGuardianSetData);

      const forkGuardianSetData = await coreBridge.GuardianSet.fromPda(
        connection,
        forkedProgram.programId,
        0
      );
      const expectedForkGuardianSetData: coreBridge.GuardianSet = {
        index: 0,
        keys: initialGuardians,
        creationTime: forkTxDetails?.blockTime,
        expirationTime: 0,
      };
      expectDeepEqual(forkGuardianSetData, expectedForkGuardianSetData);

      // Now check between the two guardian sets.
      expectDeepEqual(
        {
          index: guardianSetData.index,
          keys: guardianSetData.keys,
          expirationTime: guardianSetData.expirationTime,
        },
        {
          index: forkGuardianSetData.index,
          keys: forkGuardianSetData.keys,
          expirationTime: forkGuardianSetData.expirationTime,
        }
      );

      const feeCollectorData = await connection.getAccountInfo(
        coreBridge.feeCollectorPda(program.programId)
      );
      expect(feeCollectorData).is.not.null;
      const forkFeeCollectorData = await connection.getAccountInfo(
        coreBridge.feeCollectorPda(program.programId)
      );
      expect(feeCollectorData.lamports).to.equal(forkFeeCollectorData.lamports);
    });
  });

  describe("New implementation", () => {
    it("Cannot Invoke `initialize` again", async () => {
      // Create the initialize instruction using the default args.
      const ix = coreBridge.legacyInitializeIx(program, { payer: payer.publicKey }, defaultArgs());

      // Confirm that we cannot invoke initialize again.
      await expectIxErr(connection, [ix], [payer], "already in use");
    });
  });
});

function defaultArgs() {
  return {
    guardianSetTtlSeconds: 5,
    feeLamports: new anchor.BN(42069),
    initialGuardians: GUARDIAN_KEYS.map((privateKey) =>
      Array.from(ethers.utils.arrayify(new ethers.Wallet(privateKey).address))
    ),
  };
}

async function parallelTxDetails(
  program: coreBridge.CoreBridgeProgram,
  forkedProgram: coreBridge.CoreBridgeProgram,
  accounts: coreBridge.LegacyInitializeContext,
  args: coreBridge.LegacyInitializeArgs,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;
  const ix = coreBridge.legacyInitializeIx(program, accounts, args);

  const forkedIx = coreBridge.legacyInitializeIx(forkedProgram, accounts, args);
  return Promise.all([
    expectIxOkDetails(connection, [ix], [payer]),
    expectIxOkDetails(connection, [forkedIx], [payer]),
  ]);
}
