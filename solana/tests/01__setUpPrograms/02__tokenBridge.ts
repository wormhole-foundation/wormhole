import { web3 } from "@coral-xyz/anchor";
import { expect } from "chai";
import { readFileSync } from "fs";
import { tokenBridge } from "wormhole-solana-sdk";
import {
  CORE_BRIDGE_PROGRAM_ID,
  LOCALHOST,
  TOKEN_BRIDGE_PROGRAM_ID,
  artifactsPath,
  coreBridgeKeyPath,
  deployProgram,
  expectIxErr,
  expectIxOk,
  tmpPath,
} from "../helpers";

describe("Set Up Programs: Token Bridge", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const deployerSigner = web3.Keypair.fromSecretKey(
    Uint8Array.from(
      JSON.parse(readFileSync(`${tmpPath()}/deployer.json`, "utf-8"))
    )
  );
  const deployer = deployerSigner.publicKey;

  describe("Ok", async () => {
    it("Invoke `initialize`", async () => {
      // Accounts.
      const accounts = tokenBridge.InitializeContext.new(
        TOKEN_BRIDGE_PROGRAM_ID,
        deployer,
        CORE_BRIDGE_PROGRAM_ID
      );

      const ix = await tokenBridge.initializeIx(
        connection,
        TOKEN_BRIDGE_PROGRAM_ID,
        accounts
      );

      await expectIxOk(connection, [ix], [deployerSigner]);

      const configData = await tokenBridge.Config.fromPda(
        connection,
        TOKEN_BRIDGE_PROGRAM_ID
      );
      expect(
        configData.coreBridge.equals(new web3.PublicKey(CORE_BRIDGE_PROGRAM_ID))
      ).is.true;
    });

    it("Cannot Invoke `initialize` again", async () => {
      // Accounts.
      const accounts = tokenBridge.InitializeContext.new(
        TOKEN_BRIDGE_PROGRAM_ID,
        deployer,
        CORE_BRIDGE_PROGRAM_ID
      );

      // This invocation should fail because we cannot create any of the accounts that have been
      // created before.
      const ix = await tokenBridge.initializeIx(
        connection,
        TOKEN_BRIDGE_PROGRAM_ID,
        accounts
      );

      await expectIxErr(connection, [ix], [deployerSigner], "already in use");
    });

    it("Cannot Upgrade Token Bridge without Upgrade Authority", async () => {
      const deployerKeypath = `${tmpPath()}/deployer.json`;

      try {
        deployProgram(
          deployerKeypath,
          `${artifactsPath()}/solana_wormhole_token_bridge.so`,
          coreBridgeKeyPath()
        );
        throw new Error("borked");
      } catch (err) {
        expect(err.stderr).includes("does not match authority provided");
      }
    });
  });
});
