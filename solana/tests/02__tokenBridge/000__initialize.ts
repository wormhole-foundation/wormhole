import * as anchor from "@coral-xyz/anchor";
import { expectIxOk } from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import * as tokenBridge from "../helpers/tokenBridge";

describe("Token Bridge -- Instruction: Initialize", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(
    connection,
    tokenBridge.getProgramId("B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE")
  );
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(
    connection,
    tokenBridge.getProgramId("wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb")
  );

  describe("Ok", () => {
    it("Invoke `initialize`", async () => {
      await parallelTxOk(
        program,
        forkedProgram,
        { payer: payer.publicKey },
        defaultArgs(),
        payer
      );
    });

    it.skip("Cannot Invoke `initialize` again", async () => {
      // TODO
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

  const forkedIx = tokenBridge.legacyInitializeIx(
    forkedProgram,
    accounts,
    args
  );
  return expectIxOk(connection, [ix, forkedIx], [payer]);
}
