import * as anchor from "@coral-xyz/anchor";
import { ethers } from "ethers";
import {
  InvalidAccountConfig,
  InvalidArgConfig,
  expectDeepEqual,
  expectIxErr,
  expectIxOkDetails,
} from "../helpers";
import { transferMessageFeeIx } from "../helpers/coreBridge/utils";
import * as coreBridge from "../helpers/coreBridge";
import { expect } from "chai";

describe("Core Bridge -- Legacy Instruction: Post Message", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;
  const forkedProgram = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  const commonEmitterSequence = new anchor.BN(0);

  describe("Invalid Interaction", () => {
    const accountConfigs: InvalidAccountConfig[] = [
      {
        label: "bridge",
        contextName: "bridge",
        address: anchor.web3.Keypair.generate().publicKey,
        errorMsg: "AccountNotInitialized",
      },
      {
        label: "fee_collector",
        contextName: "feeCollector",
        address: anchor.web3.Keypair.generate().publicKey,
        errorMsg: "AccountNotInitialized",
      },
      {
        label: "emitter_sequence",
        contextName: "emitterSequence",
        address: anchor.web3.Keypair.generate().publicKey,
        errorMsg: "ConstraintSeeds",
      },
    ];

    for (const cfg of accountConfigs) {
      it(`Account: ${cfg.label} (${cfg.errorMsg})`, async () => {
        // Create the post message instruction.
        const messageSigner = anchor.web3.Keypair.generate();
        const emitter = anchor.web3.Keypair.generate();
        const accounts = {
          message: messageSigner.publicKey,
          emitter: emitter.publicKey,
          payer: payer.publicKey,
        };
        accounts[cfg.contextName] = cfg.address;
        const ix = coreBridge.legacyPostMessageIx(program, accounts, defaultArgs());
        await expectIxErr(connection, [ix], [payer, emitter, messageSigner], cfg.errorMsg);
      });
    }

    it("Instruction: Cannot Invoke `post_message` Without Paying Fee", async () => {
      // Create the post message instruction.
      const messageSigner = anchor.web3.Keypair.generate();
      const emitter = anchor.web3.Keypair.generate();
      const accounts = {
        message: messageSigner.publicKey,
        emitter: emitter.publicKey,
        payer: payer.publicKey,
      };
      const ix = coreBridge.legacyPostMessageIx(program, accounts, defaultArgs());
      await expectIxErr(connection, [ix], [payer, emitter, messageSigner], "InsufficientFees");
    });

    it("Instruction: Cannot Invoke `post_message` With Invalid Payload", async () => {
      // Create the post message instruction.
      const messageSigner = anchor.web3.Keypair.generate();
      const emitter = anchor.web3.Keypair.generate();
      const accounts = {
        message: messageSigner.publicKey,
        emitter: emitter.publicKey,
        payer: payer.publicKey,
      };
      let { nonce, payload, finality } = defaultArgs();
      payload = Buffer.alloc(0);

      const ix = coreBridge.legacyPostMessageIx(program, accounts, {
        nonce,
        payload,
        finality,
      });
      await expectIxErr(
        connection,
        [ix],
        [payer, emitter, messageSigner],
        "InvalidInstructionArgument"
      );
    });
  });

  describe("Ok", () => {
    it("Invoke `post_message`", async () => {
      // Fetch default args.
      const { nonce, payload, finality } = defaultArgs();
      const accounts = {
        message: null, // leave null for now
        emitter: payer.publicKey, // emitter
        payer: payer.publicKey,
      };

      // Invoke `postMessage`.
      const [txDetails, forkTxDetails, messageSigner, forkedMessageSigner] =
        await parallelTxDetails(
          program,
          forkedProgram,
          accounts,
          { nonce, payload, finality },
          payer,
          payer // as emitterSigner
        );

      // Validate bridge data account.
      await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      // Confirm that the message data accounts are the same.
      await coreBridge.expectEqualMessageAccounts(
        program,
        messageSigner,
        forkedMessageSigner,
        false
      );

      // Validate data in the message accounts.
      await coreBridge.expectLegacyPostMessageAfterEffects(
        program,
        txDetails,
        accounts,
        { nonce, payload, finality },
        commonEmitterSequence,
        false,
        payload
      );

      await coreBridge.expectLegacyPostMessageAfterEffects(
        forkedProgram,
        forkTxDetails,
        accounts,
        { nonce, payload, finality },
        commonEmitterSequence,
        false,
        payload
      );

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

    it("Invoke `post_message` Again With Same Emitter", async () => {
      // Fetch default args.
      let { nonce, payload, finality } = defaultArgs();
      const accounts = {
        message: null, // leave null for now
        emitter: payer.publicKey, // emitter
        payer: payer.publicKey,
      };

      // Change the payload.
      payload = Buffer.from("Somebody set up us the bomb.");

      // Invoke `postMessage`.
      const [txDetails, forkTxDetails, messageSigner, forkedMessageSigner] =
        await parallelTxDetails(
          program,
          forkedProgram,
          accounts,
          { nonce, payload, finality },
          payer,
          payer // as emitterSigner
        );

      // Validate bridge data account.
      await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      // Confirm that the message data accounts are the same.
      await coreBridge.expectEqualMessageAccounts(
        program,
        messageSigner,
        forkedMessageSigner,
        false
      );

      // Validate data in the message accounts.
      await coreBridge.expectLegacyPostMessageAfterEffects(
        program,
        txDetails,
        accounts,
        { nonce, payload, finality },
        commonEmitterSequence,
        false,
        payload
      );

      await coreBridge.expectLegacyPostMessageAfterEffects(
        forkedProgram,
        forkTxDetails,
        accounts,
        { nonce, payload, finality },
        commonEmitterSequence,
        false,
        payload
      );

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

    it("Invoke `post_message` (Emitter != Payer)", async () => {
      // Create new emitter.
      const emitterSigner = anchor.web3.Keypair.generate();

      // Fetch default args.
      let { nonce, payload, finality } = defaultArgs();
      const accounts = {
        message: null, // leave null for now
        emitter: emitterSigner.publicKey, // emitter
        payer: payer.publicKey,
      };

      // Invoke `postMessage`.
      const [txDetails, forkTxDetails, messageSigner, forkedMessageSigner] =
        await parallelTxDetails(
          program,
          forkedProgram,
          accounts,
          { nonce, payload, finality },
          payer,
          emitterSigner
        );

      // Validate bridge data account.
      await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      // Confirm that the message data accounts are the same.
      await coreBridge.expectEqualMessageAccounts(
        program,
        messageSigner,
        forkedMessageSigner,
        false
      );

      // We're testing a new emitter, set the sequence to zero.
      const startingSequence = new anchor.BN(0);

      // Validate data in the message accounts.
      await coreBridge.expectLegacyPostMessageAfterEffects(
        program,
        txDetails,
        accounts,
        { nonce, payload, finality },
        startingSequence,
        false,
        payload
      );

      await coreBridge.expectLegacyPostMessageAfterEffects(
        forkedProgram,
        forkTxDetails,
        accounts,
        { nonce, payload, finality },
        startingSequence,
        false,
        payload
      );

      // Validate emitter sequences.
      const sequence = await coreBridge.EmitterSequence.fromPda(
        connection,
        program.programId,
        emitterSigner.publicKey
      );
      const forkSequence = await coreBridge.EmitterSequence.fromPda(
        connection,
        forkedProgram.programId,
        emitterSigner.publicKey
      );
      expectDeepEqual(sequence, forkSequence);
      expectDeepEqual(sequence.sequence, startingSequence.iaddn(1));

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
  payer: anchor.web3.Keypair,
  emitterSigner: anchor.web3.Keypair
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
  await expectIxOkDetails(
    connection,
    await Promise.all(
      [program, forkedProgram].map((prog) => transferMessageFeeIx(prog, payer.publicKey))
    ),
    [payer]
  );

  return Promise.all([
    expectIxOkDetails(connection, [ix], [payer, emitterSigner, messageSigner]),
    expectIxOkDetails(connection, [forkedIx], [payer, emitterSigner, forkedMessageSigner]),
    messageSigner,
    forkedMessageSigner,
  ]);
}
