import * as anchor from "@coral-xyz/anchor";
import { ethers } from "ethers";
import {
  GUARDIAN_KEYS,
  InvalidAccountConfig,
  InvalidArgConfig,
  expectDeepEqual,
  expectIxErr,
  expectIxOkDetails,
} from "../helpers";
import { transferMessageFeeIx } from "../helpers/coreBridge/utils";
import * as coreBridge from "../helpers/coreBridge";
import { expect } from "chai";

describe("Core Bridge -- Instruction: Post Message", () => {
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

  const commonEmitterSequence = new anchor.BN(0);

  describe("Invalid Interaction", () => {});

  describe("Ok", () => {
    it("Invoke `postMessage`", async () => {
      // Fetch default args.
      const { nonce, payload, finality } = defaultArgs();

      // Invoke `postMessage`.
      const [txDetails, forkTxDetails, messageSigner, forkedMessageSigner] =
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            message: null, // leave null for now
            emitter: payer.publicKey, // emitter
            payer: payer.publicKey,
          },
          { nonce, payload, finality },
          payer
        );

      // Validate bridge data account.
      await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      // Validate message account data.
      await coreBridge.expectEqualMessageAccounts(program, messageSigner, forkedMessageSigner);

      // Validate emitter sequences.
      commonEmitterSequence.iaddn(1);

      const sequence = await coreBridge.EmitterSequence.fromPda(
        connection,
        program.programId,
        payer.publicKey
      );
      const forkSequence = await coreBridge.EmitterSequence.fromPda(
        connection,
        forkedProgram.programId,
        payer.publicKey
      );
      expectDeepEqual(sequence, forkSequence);
      expectDeepEqual(sequence.sequence, commonEmitterSequence);

      // Validate fee collector.
      const feeCollectorData = await connection.getAccountInfo(
        coreBridge.FeeCollector.address(program.programId)
      );
      expect(feeCollectorData).is.not.null;
      const forkFeeCollectorData = await connection.getAccountInfo(
        coreBridge.FeeCollector.address(program.programId)
      );
      expect(feeCollectorData.lamports).to.equal(forkFeeCollectorData.lamports);
    });
  });
});

function defaultArgs() {
  return {
    nonce: 420,
    payload: Buffer.from("All your base are belong to us."),
    finality: 1,
  };
}

async function parallelTxDetails(
  program: coreBridge.CoreBridgeProgram,
  forkedProgram: coreBridge.CoreBridgeProgram,
  accounts: coreBridge.LegacyPostMessageContext,
  args: coreBridge.LegacyPostMessageArgs,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;

  // Create a message signer for both the main and forked program.
  const messageSigner = anchor.web3.Keypair.generate();
  const forkedMessageSigner = anchor.web3.Keypair.generate();

  // Create the post message instruction.
  accounts.message = messageSigner.publicKey;
  const ix = coreBridge.legacyPostMessageIx(program, accounts, args);

  // Create the post message instruction for the forked program.
  accounts.message = forkedMessageSigner.publicKey;
  const forkedIx = coreBridge.legacyPostMessageIx(forkedProgram, accounts, args);

  // Pay the fee collector prior to publishing each message.
  await Promise.all([
    expectIxOkDetails(connection, [await transferMessageFeeIx(program, payer.publicKey)], [payer]),
    expectIxOkDetails(
      connection,
      [await transferMessageFeeIx(forkedProgram, payer.publicKey)],
      [payer]
    ),
  ]);

  return Promise.all([
    expectIxOkDetails(connection, [ix], [payer, messageSigner]),
    expectIxOkDetails(connection, [forkedIx], [payer, forkedMessageSigner]),
    messageSigner,
    forkedMessageSigner,
  ]);
}
