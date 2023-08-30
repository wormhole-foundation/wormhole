import * as anchor from "@coral-xyz/anchor";
import { expectIxOk, expectIxErr, MINT_INFO_6, MINT_INFO_8, MINT_INFO_9 } from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import * as tokenBridge from "../helpers/tokenBridge";
import { createAssociatedTokenAccount } from "@solana/spl-token";

describe("Token Bridge -- Instruction: Initialize", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  before("Set up Token Accounts", async () => {
    // Native mints.
    await Promise.all(
      [MINT_INFO_8, MINT_INFO_9].map((info) =>
        createAssociatedTokenAccount(connection, payer, info.mint, payer.publicKey)
      )
    );
  });

  describe("Ok", () => {
    it("Invoke `initialize`", async () => {
      await parallelTxOk(program, forkedProgram, { payer: payer.publicKey }, defaultArgs(), payer);
    });
  });

  describe("New Implentation", () => {
    it("Cannot Invoke `initialize` again", async () => {
      // Create the initialize instruction using the default args.
      const ix = tokenBridge.legacyInitializeIx(program, { payer: payer.publicKey }, defaultArgs());

      // Confirm that we cannot invoke initialize again.
      await expectIxErr(connection, [ix], [payer], "already in use");
    });
  });
});

function defaultArgs() {
  return {
    coreBridgeProgram: coreBridge.getProgramId(),
  };
}

async function parallelTxOk(
  program: tokenBridge.TokenBridgeProgram,
  forkedProgram: tokenBridge.TokenBridgeProgram,
  accounts: tokenBridge.LegacyInitializeContext,
  args: tokenBridge.LegacyInitializeArgs,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;
  const ix = tokenBridge.legacyInitializeIx(program, accounts, args);

  const forkedIx = tokenBridge.legacyInitializeIx(forkedProgram, accounts, args);
  return expectIxOk(connection, [ix, forkedIx], [payer]);
}
