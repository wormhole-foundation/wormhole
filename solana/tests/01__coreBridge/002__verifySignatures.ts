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

describe("Core Bridge -- Legacy Instruction: Verify Signatures", () => {
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
    // TODO
  });

  describe("Ok", () => {
    it.skip("Invoke `verify_signatures`", async () => {
      // TODO
    });
  });
});

function defaultArgs() {
  return {};
}

async function parallelIxOk(
  program: coreBridge.CoreBridgeProgram,
  forkedProgram: coreBridge.CoreBridgeProgram,
  accounts: coreBridge.LegacyInitializeContext,
  args: coreBridge.LegacyInitializeArgs,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;
  // const ix = coreBridge.legacyInitializeIx(program, accounts, args);

  // const forkedIx = coreBridge.legacyInitializeIx(forkedProgram, accounts, args);
  // return Promise.all([
  //   expectIxOkDetails(connection, [ix], [payer]),
  //   expectIxOkDetails(connection, [forkedIx], [payer]),
  // ]);
}
