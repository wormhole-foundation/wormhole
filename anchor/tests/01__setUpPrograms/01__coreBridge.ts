import { BN, web3 } from "@coral-xyz/anchor";
import { expect } from "chai";
import { ethers } from "ethers";
import { readFileSync } from "fs";
import { coreBridge } from "wormhole-solana-sdk";
import {
  CORE_BRIDGE_PROGRAM_ID,
  GUARDIAN_KEYS,
  LOCALHOST,
  artifactsPath,
  coreBridgeKeyPath,
  deployProgram,
  expectIxTransactionDetails,
  expectIxErr,
  tmpPath,
} from "../helpers";

describe("Set Up Programs: Core Bridge", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const deployerSigner = web3.Keypair.fromSecretKey(
    Uint8Array.from(
      JSON.parse(readFileSync(`${tmpPath()}/deployer.json`, "utf-8"))
    )
  );
  const deployer = deployerSigner.publicKey;

  describe("Ok", async () => {
    it("Invoke `initialize` with 19 Devnet Guardians", async () => {
      const guardianSetIndex = 0;

      // Accounts.
      const accounts = coreBridge.InitializeContext.new(
        CORE_BRIDGE_PROGRAM_ID,
        deployer
      );

      // Instruction handler args.
      const guardianSetTtlSeconds = 15;
      const feeLamports = new BN(42069);
      const initialGuardians = GUARDIAN_KEYS.map(
        (privateKey): Array<20> =>
          Array.from(
            ethers.utils.arrayify(new ethers.Wallet(privateKey).address)
          ) as 20[]
      );

      const ix = await coreBridge.initializeIx(
        connection,
        CORE_BRIDGE_PROGRAM_ID,
        accounts,
        {
          guardianSetTtlSeconds,
          feeLamports,
          initialGuardians,
        }
      );

      const txDetails = await expectIxTransactionDetails(
        connection,
        [ix],
        [deployerSigner]
      );

      const { guardianSet } = accounts;
      const guardianSetData = await coreBridge.GuardianSet.fromAccountAddress(
        connection,
        guardianSet
      );
      expect(guardianSetData.creationTime).equals(txDetails?.blockTime);
      expect(guardianSetData.expirationTime).equals(0);
      expect(guardianSetData.index).equals(0);

      const numGuardians = initialGuardians.length;
      expect(guardianSetData.keys).has.length(numGuardians);
      for (let i = 0; i < numGuardians; ++i) {
        const actual = Buffer.from(guardianSetData.keys[i]);
        const expected = Buffer.from(initialGuardians[i]);
        expect(actual.equals(expected)).is.true;
      }
    });

    // it("Cannot Invoke `initialize` again", async () => {
    //   // Accounts.
    //   const accounts = coreBridge.InitializeContext.new(
    //     CORE_BRIDGE_PROGRAM_ID,
    //     deployer
    //   );

    //   // Instruction handler args.
    //   const guardianSetTtlSeconds = 69;
    //   const feeLamports = new BN(420);
    //   const initialGuardians = [Array.from(Buffer.alloc(20, 69)) as 20[]];

    //   // This invocation should fail because we cannot create any of the accounts that have been
    //   // created before.
    //   const ix = await coreBridge.initializeIx(
    //     connection,
    //     CORE_BRIDGE_PROGRAM_ID,
    //     accounts,
    //     {
    //       guardianSetTtlSeconds,
    //       feeLamports,
    //       initialGuardians,
    //     }
    //   );

    //   await expectIxErr(connection, [ix], [deployerSigner], "already in use");
    // });

    // it("Cannot Upgrade Core Bridge without Upgrade Authority", async () => {
    //   const deployerKeypath = `${tmpPath()}/deployer.json`;

    //   try {
    //     deployProgram(
    //       deployerKeypath,
    //       `${artifactsPath()}/solana_wormhole_core_bridge.so`,
    //       coreBridgeKeyPath()
    //     );
    //     throw new Error("borked");
    //   } catch (err) {
    //     expect(err.stderr).includes("does not match authority provided");
    //   }
    // });
  });
});
