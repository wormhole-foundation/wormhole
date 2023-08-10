import * as anchor from "@coral-xyz/anchor";
import { NATIVE_MINT } from "@solana/spl-token";
import { PublicKey } from "@solana/web3.js";
import { expectIxOk } from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import * as tokenBridge from "../helpers/tokenBridge";

describe("Token Bridge -- Legacy Instruction: Attest Token", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  describe("Ok", () => {
    it("Invoke `attest_token`", async () => {
      const [coreMessage, forkedCoreMessage] = await parallelTxOk(
        program,
        forkedProgram,
        { payer: payer.publicKey, mint: NATIVE_MINT },
        defaultArgs(),
        payer
      );

      // TODO: Check message accounts.
    });

    it("Invoke `attest_token` again", async () => {
      await parallelTxOk(
        program,
        forkedProgram,
        { payer: payer.publicKey, mint: NATIVE_MINT },
        defaultArgs(),
        payer
      );
    });
  });
});

function defaultArgs() {
  return {
    nonce: 420,
  };
}

async function parallelTxOk(
  program: tokenBridge.TokenBridgeProgram,
  forkedProgram: tokenBridge.TokenBridgeProgram,
  accounts: { payer: PublicKey; mint: PublicKey },
  args: tokenBridge.LegacyAttestTokenArgs,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;
  const coreMessage = anchor.web3.Keypair.generate();
  const ix = tokenBridge.legacyAttestTokenIx(
    program,
    {
      coreMessage: coreMessage.publicKey,
      ...accounts,
    },
    args
  );

  const forkedCoreMessage = anchor.web3.Keypair.generate();
  const forkedIx = tokenBridge.legacyAttestTokenIx(
    forkedProgram,
    {
      coreMessage: forkedCoreMessage.publicKey,
      ...accounts,
    },
    args
  );

  await expectIxOk(connection, [ix, forkedIx], [payer, coreMessage, forkedCoreMessage]);

  return [coreMessage, forkedCoreMessage];
}
