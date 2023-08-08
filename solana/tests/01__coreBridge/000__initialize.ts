import * as anchor from "@coral-xyz/anchor";
import { ethers } from "ethers";
import {
  GUARDIAN_KEYS,
  InvalidAccountConfig,
  InvalidArgConfig,
  expectDeepEqual,
  expectIxErr,
  expectIxOkDetails,
  sleep,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import { expect } from "chai";

// TODO: Need to add negative tests for GuardianZeroAddress, DuplicateGuardians, etc.

describe("Core Bridge -- Instruction: Initialize", () => {
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

  describe("Invalid Interaction", () => {
    const accountConfigs: InvalidAccountConfig[] = [
      {
        label: "bridge",
        contextName: "bridge",
        address: anchor.web3.Keypair.generate().publicKey,
        errorMsg: "ConstraintSeeds",
      },
      {
        label: "guardian_set",
        contextName: "guardianSet",
        address: anchor.web3.Keypair.generate().publicKey,
        errorMsg: "ConstraintSeeds",
      },
      {
        label: "fee_collector",
        contextName: "feeCollector",
        address: anchor.web3.Keypair.generate().publicKey,
        errorMsg: "ConstraintSeeds",
      },
    ];

    for (const cfg of accountConfigs) {
      it(`Account: ${cfg.label} (${cfg.errorMsg})`, async () => {
        const accounts = { payer: payer.publicKey };
        accounts[cfg.contextName] = cfg.address;
        const ix = coreBridge.legacyInitializeIx(program, accounts, defaultArgs());
        await expectIxErr(connection, [ix], [payer], cfg.errorMsg);
      });
    }

    const argConfigs: InvalidArgConfig[] = [
      {
        label: "initial_guardians",
        argName: "initialGuardians",
        value: [],
        errorMsg: "ZeroGuardians",
      },
    ];

    for (const cfg of argConfigs) {
      it(`Instruction Data: ${cfg.label} (${cfg.errorMsg})`, async () => {
        const args = defaultArgs();
        args[cfg.argName] = cfg.value;
        const ix = coreBridge.legacyInitializeIx(
          program,
          { payer: payer.publicKey },
          args
        );
        await expectIxErr(connection, [ix], [payer], cfg.errorMsg);
      });
    }
  });

  describe("Ok", () => {
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
        coreBridge.FeeCollector.address(program.programId)
      );
      expect(feeCollectorData).is.not.null;
      const forkFeeCollectorData = await connection.getAccountInfo(
        coreBridge.FeeCollector.address(program.programId)
      );
      expect(feeCollectorData.lamports).to.equal(forkFeeCollectorData.lamports);
    });

    it("Cannot Invoke `initialize` again", async () => {
      // Create the initialize instruction using the default args.
      const ix = coreBridge.legacyInitializeIx(
        program,
        { payer: payer.publicKey },
        defaultArgs()
      );

      // Sleep to avoid anchor validator error.
      await sleep(10000);

      // Confirm that we cannot invoke initialize again.
      await expectIxErr(connection, [ix], [payer], "already in use");
    });
  });
});

function defaultArgs() {
  return {
    guardianSetTtlSeconds: 15,
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
