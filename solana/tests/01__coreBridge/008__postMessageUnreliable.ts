import * as anchor from "@coral-xyz/anchor";
import { expect } from "chai";
import {
  InvalidAccountConfig,
  createIfNeeded,
  expectDeepEqual,
  expectIxErr,
  expectIxOk,
  expectIxOkDetails,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import { transferMessageFeeIx } from "../helpers/coreBridge/utils";

describe("Core Bridge -- Instruction: Post Message Unreliable", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;
  const forkedProgram = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  const commonEmitterSequence = new anchor.BN(0);
  const commonEmitter = anchor.web3.Keypair.generate();
  const messageSigner = anchor.web3.Keypair.generate();
  const forkedMessageSigner = anchor.web3.Keypair.generate();

  describe("Invalid Interaction", () => {
    const accountConfigs: InvalidAccountConfig[] = [
      {
        label: "config",
        contextName: "config",
        errorMsg: "ConstraintSeeds",
        dataLength: 24,
        owner: program.programId,
      },
      {
        label: "fee_collector",
        contextName: "feeCollector",
        errorMsg: "ConstraintSeeds",
        dataLength: 0,
        owner: anchor.web3.PublicKey.default,
      },
      {
        label: "emitter_sequence",
        contextName: "emitterSequence",
        errorMsg: "ConstraintSeeds",
        dataLength: 8,
        owner: program.programId,
      },
    ];

    for (const cfg of accountConfigs) {
      it(`Account: ${cfg.label} (${cfg.errorMsg})`, async () => {
        const message = anchor.web3.Keypair.generate();
        const emitter = anchor.web3.Keypair.generate();
        const accounts = await createIfNeeded(program.provider.connection, cfg, payer, {
          message: message.publicKey,
          emitter: emitter.publicKey,
          payer: payer.publicKey,
        } as coreBridge.LegacyPostMessageUnreliableContext);

        // Create the post message instruction.
        const ix = coreBridge.legacyPostMessageUnreliableIx(program, accounts, defaultArgs());
        await expectIxErr(connection, [ix], [payer, emitter, message], cfg.errorMsg);
      });
    }
  });

  describe("Ok", () => {
    it("Invoke `post_message_unreliable`", async () => {
      // Fetch default args.
      const { nonce, payload, commitment } = defaultArgs();

      // Create parallel transaction args.
      const args: parallelTxArgs = {
        new: {
          program,
          messageSigner,
          emitterSigner: commonEmitter,
        },
        fork: {
          program: forkedProgram,
          messageSigner: forkedMessageSigner,
          emitterSigner: commonEmitter,
        },
      };

      // Invoke `postMessage`.
      const [txDetails, forkTxDetails] = await parallelTxDetails(
        args,
        { nonce, payload, commitment },
        payer
      );

      // Validate bridge data account.
      await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      // Confirm that the message data accounts are the same.
      await coreBridge.expectEqualMessageAccounts(
        program,
        messageSigner,
        forkedMessageSigner,
        true
      );

      // Validate data in the message accounts.
      await coreBridge.expectLegacyPostMessageAfterEffects(
        program,
        txDetails!,
        {
          payer: payer.publicKey,
          message: messageSigner.publicKey,
          emitter: commonEmitter.publicKey,
        },
        { nonce, payload, commitment },
        commonEmitterSequence,
        true,
        payload
      );

      await coreBridge.expectLegacyPostMessageAfterEffects(
        forkedProgram,
        forkTxDetails!,
        {
          payer: payer.publicKey,
          message: forkedMessageSigner.publicKey,
          emitter: commonEmitter.publicKey,
        },
        { nonce, payload, commitment },
        commonEmitterSequence,
        true,
        payload
      );

      // Up tick emitter sequences.
      commonEmitterSequence.iaddn(1);

      // Validate fee collector.
      const feeCollectorData = await connection.getAccountInfo(
        coreBridge.feeCollectorPda(program.programId)
      );
      expect(feeCollectorData).is.not.null;
      const forkFeeCollectorData = await connection.getAccountInfo(
        coreBridge.feeCollectorPda(program.programId)
      );
      expect(feeCollectorData!.lamports).to.equal(forkFeeCollectorData!.lamports);
    });

    it("Invoke `post_message_unreliable` Using Same Message Signer", async () => {
      // Fetch existing message from the program. Since we are using the same
      // signer, the message data account should be the same.
      const [existingCommitment, existingNonce, existingPayload] =
        await coreBridge.PostedMessageV1Unreliable.fromAccountAddress(
          connection,
          messageSigner.publicKey
        ).then((msg): [anchor.web3.Commitment, number, Buffer] => [
          coreBridge.fromConsistencyLevel(msg.consistencyLevel),
          msg.nonce,
          msg.payload,
        ]);

      // Create parallel transaction args.
      const args: parallelTxArgs = {
        new: {
          program,
          messageSigner,
          emitterSigner: commonEmitter,
        },
        fork: {
          program: forkedProgram,
          messageSigner: forkedMessageSigner,
          emitterSigner: commonEmitter,
        },
      };

      // Construct a different message with the same size as the original.
      const nonce = 69;
      expect(nonce).not.equals(existingNonce);

      const commitment = "confirmed";
      expect(commitment).not.equals(existingCommitment);

      const payload = Buffer.alloc(existingPayload.length);
      payload.fill(0);
      payload.write("So fresh and so clean clean.");
      expect(payload.equals(existingPayload)).is.false;

      // Invoke `postMessage`.
      const [txDetails, forkTxDetails] = await parallelTxDetails(
        args,
        { nonce, payload, commitment },
        payer
      );

      // Validate bridge data account.
      await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      // Confirm that the message data accounts are the same.
      await coreBridge.expectEqualMessageAccounts(
        program,
        messageSigner,
        forkedMessageSigner,
        true
      );

      // Validate data in the message accounts.
      await coreBridge.expectLegacyPostMessageAfterEffects(
        program,
        txDetails!,
        {
          payer: payer.publicKey,
          message: messageSigner.publicKey,
          emitter: commonEmitter.publicKey,
        },
        { nonce, payload, commitment },
        commonEmitterSequence,
        true,
        payload
      );

      await coreBridge.expectLegacyPostMessageAfterEffects(
        forkedProgram,
        forkTxDetails!,
        {
          payer: payer.publicKey,
          message: forkedMessageSigner.publicKey,
          emitter: commonEmitter.publicKey,
        },
        { nonce, payload, commitment },
        commonEmitterSequence,
        true,
        payload
      );

      // Up tick emitter sequences.
      commonEmitterSequence.iaddn(1);

      // Validate fee collector.
      const feeCollectorData = await connection.getAccountInfo(
        coreBridge.feeCollectorPda(program.programId)
      );
      expect(feeCollectorData).is.not.null;
      const forkFeeCollectorData = await connection.getAccountInfo(
        coreBridge.feeCollectorPda(program.programId)
      );
      expect(feeCollectorData!.lamports).to.equal(forkFeeCollectorData!.lamports);
    });

    it("Invoke `post_message_unreliable` with New Message Signer", async () => {
      // Fetch default args.
      const { nonce, commitment } = defaultArgs();
      const payload = Buffer.from("Would you just look at that?");

      // Create two new message signers.
      const newMessageSigner = anchor.web3.Keypair.generate();
      const newForkedMessageSigner = anchor.web3.Keypair.generate();

      // Create parallel transaction args.
      const args: parallelTxArgs = {
        new: {
          program,
          messageSigner: newMessageSigner,
          emitterSigner: commonEmitter,
        },
        fork: {
          program: forkedProgram,
          messageSigner: newForkedMessageSigner,
          emitterSigner: commonEmitter,
        },
      };

      // Invoke `postMessage`.
      const [txDetails, forkTxDetails] = await parallelTxDetails(
        args,
        { nonce, payload, commitment },
        payer
      );

      // Validate bridge data account.
      await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      // Confirm that the message data accounts are the same.
      await coreBridge.expectEqualMessageAccounts(
        program,
        newMessageSigner,
        newForkedMessageSigner,
        true
      );

      // Validate data in the message accounts.
      await coreBridge.expectLegacyPostMessageAfterEffects(
        program,
        txDetails!,
        {
          payer: payer.publicKey,
          message: newMessageSigner.publicKey,
          emitter: commonEmitter.publicKey,
        },
        { nonce, payload, commitment },
        commonEmitterSequence,
        true,
        payload
      );

      await coreBridge.expectLegacyPostMessageAfterEffects(
        forkedProgram,
        forkTxDetails!,
        {
          payer: payer.publicKey,
          message: newForkedMessageSigner.publicKey,
          emitter: commonEmitter.publicKey,
        },
        { nonce, payload, commitment },
        commonEmitterSequence,
        true,
        payload
      );

      // Up tick emitter sequences.
      commonEmitterSequence.iaddn(1);

      // Validate fee collector.
      const feeCollectorData = await connection.getAccountInfo(
        coreBridge.feeCollectorPda(program.programId)
      );
      expect(feeCollectorData).is.not.null;
      const forkFeeCollectorData = await connection.getAccountInfo(
        coreBridge.feeCollectorPda(program.programId)
      );
      expect(feeCollectorData!.lamports).to.equal(forkFeeCollectorData!.lamports);
    });

    it("Invoke `post_message_unreliable` with Payer as Emitter", async () => {
      // Fetch default args.
      const { nonce, commitment } = defaultArgs();
      const payload = Buffer.from("Would you just look at that?");

      // Create two new message signers.
      const newMessageSigner = anchor.web3.Keypair.generate();
      const newForkedMessageSigner = anchor.web3.Keypair.generate();

      // Create parallel transaction args.
      const args: parallelTxArgs = {
        new: {
          program,
          messageSigner: newMessageSigner,
          emitterSigner: payer,
        },
        fork: {
          program: forkedProgram,
          messageSigner: newForkedMessageSigner,
          emitterSigner: payer,
        },
      };

      // Fetch the sequence before invoking the instruction.
      const sequenceBefore = await coreBridge.EmitterSequence.fromPda(
        connection,
        program.programId,
        payer.publicKey
      );

      // Invoke `postMessage`.
      const [txDetails, forkTxDetails] = await parallelTxDetails(
        args,
        { nonce, payload, commitment },
        payer
      );

      // Validate bridge data account.
      await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      // Confirm that the message data accounts are the same.
      await coreBridge.expectEqualMessageAccounts(
        program,
        newMessageSigner,
        newForkedMessageSigner,
        true
      );

      // Validate data in the message accounts.
      await coreBridge.expectLegacyPostMessageAfterEffects(
        program,
        txDetails!,
        {
          payer: payer.publicKey,
          message: newMessageSigner.publicKey,
          emitter: payer.publicKey,
        },
        { nonce, payload, commitment },
        sequenceBefore.sequence,
        true,
        payload
      );

      await coreBridge.expectLegacyPostMessageAfterEffects(
        forkedProgram,
        forkTxDetails!,
        {
          payer: payer.publicKey,
          message: newForkedMessageSigner.publicKey,
          emitter: payer.publicKey,
        },
        { nonce, payload, commitment },
        sequenceBefore.sequence,
        true,
        payload
      );

      // Validate fee collector.
      const feeCollectorData = await connection.getAccountInfo(
        coreBridge.feeCollectorPda(program.programId)
      );
      expect(feeCollectorData).is.not.null;
      const forkFeeCollectorData = await connection.getAccountInfo(
        coreBridge.feeCollectorPda(program.programId)
      );
      expect(feeCollectorData!.lamports).to.equal(forkFeeCollectorData!.lamports);
    });

    it("Invoke `post_message_unreliable` with System program at Index == 8", async () => {
      const emitter = anchor.web3.Keypair.generate();
      const message = anchor.web3.Keypair.generate();
      const forkMessage = anchor.web3.Keypair.generate();

      const forkTransferFeeIx = await coreBridge.transferMessageFeeIx(
        forkedProgram,
        payer.publicKey
      );

      const ix = coreBridge.legacyPostMessageUnreliableIx(
        program,
        { payer: payer.publicKey, message: message.publicKey, emitter: emitter.publicKey },
        defaultArgs()
      );
      expectDeepEqual(ix.keys[7].pubkey, anchor.web3.SystemProgram.programId);
      ix.keys[7].pubkey = ix.keys[8].pubkey;
      ix.keys[8].pubkey = anchor.web3.SystemProgram.programId;

      const forkIx = coreBridge.legacyPostMessageUnreliableIx(
        forkedProgram,
        { payer: payer.publicKey, message: forkMessage.publicKey, emitter: emitter.publicKey },
        defaultArgs()
      );
      expectDeepEqual(forkIx.keys[7].pubkey, anchor.web3.SystemProgram.programId);
      forkIx.keys[7].pubkey = forkIx.keys[8].pubkey;
      forkIx.keys[8].pubkey = anchor.web3.SystemProgram.programId;

      await expectIxOk(
        connection,
        [forkTransferFeeIx, ix, forkIx],
        [payer, emitter, message, forkMessage]
      );
    });

    it("Invoke `post_message_unreliable` with Num Accounts == 8", async () => {
      const emitter = anchor.web3.Keypair.generate();
      const message = anchor.web3.Keypair.generate();
      const forkMessage = anchor.web3.Keypair.generate();

      const forkTransferFeeIx = await coreBridge.transferMessageFeeIx(
        forkedProgram,
        payer.publicKey
      );

      const ix = coreBridge.legacyPostMessageUnreliableIx(
        program,
        { payer: payer.publicKey, message: message.publicKey, emitter: emitter.publicKey },
        defaultArgs()
      );
      expect(ix.keys).has.length(9);
      ix.keys.pop();

      const forkIx = coreBridge.legacyPostMessageUnreliableIx(
        forkedProgram,
        { payer: payer.publicKey, message: forkMessage.publicKey, emitter: emitter.publicKey },
        defaultArgs()
      );
      expect(forkIx.keys).has.length(9);
      forkIx.keys.pop();

      await expectIxOk(
        connection,
        [forkTransferFeeIx, ix, forkIx],
        [payer, emitter, message, forkMessage]
      );
    });
  });

  describe("New implementation", () => {
    it("Cannot Invoke `post_message_unreliable` With Invalid Payload", async () => {
      // Create the post message instruction.
      const messageSigner = anchor.web3.Keypair.generate();
      const emitter = anchor.web3.Keypair.generate();
      const accounts = {
        message: messageSigner.publicKey,
        emitter: emitter.publicKey,
        payer: payer.publicKey,
      };
      const { nonce, commitment } = defaultArgs();
      const payload = Buffer.alloc(0);

      const ix = coreBridge.legacyPostMessageUnreliableIx(program, accounts, {
        nonce,
        payload,
        commitment,
      });
      await expectIxErr(
        connection,
        [ix],
        [payer, emitter, messageSigner],
        "InvalidInstructionArgument"
      );
    });
  });
});

function defaultArgs() {
  return {
    nonce: 420,
    payload: Buffer.from("All your base are belong to us."),
    commitment: "finalized" as anchor.web3.Commitment,
  };
}

interface parallelTxArgs {
  new: {
    program: coreBridge.CoreBridgeProgram;
    messageSigner: anchor.web3.Keypair;
    emitterSigner: anchor.web3.Keypair;
  };
  fork: {
    program: coreBridge.CoreBridgeProgram;
    messageSigner: anchor.web3.Keypair;
    emitterSigner: anchor.web3.Keypair;
  };
}

async function parallelTxDetails(
  args: parallelTxArgs,
  postUnreliableArgs: coreBridge.LegacyPostMessageArgs,
  payer: anchor.web3.Keypair
) {
  const connection = args.new.program.provider.connection;

  // Create the post message instruction.
  const ix = coreBridge.legacyPostMessageUnreliableIx(
    args.new.program,
    {
      payer: payer.publicKey,
      message: args.new.messageSigner.publicKey,
      emitter: args.new.emitterSigner.publicKey,
    },
    postUnreliableArgs
  );

  // Create the post message instruction for the forked program.
  const forkedIx = coreBridge.legacyPostMessageUnreliableIx(
    args.fork.program,
    {
      payer: payer.publicKey,
      message: args.fork.messageSigner.publicKey,
      emitter: args.fork.emitterSigner.publicKey,
    },
    postUnreliableArgs
  );

  const forkTransferFeeIx = await transferMessageFeeIx(args.fork.program, payer.publicKey);

  return Promise.all([
    expectIxOkDetails(connection, [ix], [payer, args.new.emitterSigner, args.new.messageSigner]),
    expectIxOkDetails(
      connection,
      [forkTransferFeeIx, forkedIx],
      [payer, args.fork.emitterSigner, args.fork.messageSigner]
    ),
  ]);
}
