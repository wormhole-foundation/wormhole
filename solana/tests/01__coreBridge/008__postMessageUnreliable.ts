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

describe("Core Bridge -- Instruction: Post Message Unreliable", () => {
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
  const commonEmitter = anchor.web3.Keypair.generate();
  const messageSigner = anchor.web3.Keypair.generate();
  const forkedMessageSigner = anchor.web3.Keypair.generate();

  describe("Invalid Interaction", () => {});

  describe("Ok", () => {
    it("Invoke `post_message_unreliable`", async () => {
      // Fetch default args.
      const { nonce, payload, finality } = defaultArgs();

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
        { nonce, payload, finality },
        payer
      );

      // Validate bridge data account.
      // await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      // Confirm that the message data accounts are the same.
      //await coreBridge.expectEqualMessageAccounts(program, messageSigner, forkedMessageSigner);

      // Validate data in the message accounts.
      // await coreBridge.expectLegacyPostMessageAfterEffects(
      //   program,
      //   txDetails,
      //   accounts,
      //   { nonce, payload, finality },
      //   commonEmitterSequence,
      //   false,
      //   payload
      // );
      // await coreBridge.expectLegacyPostMessageAfterEffects(
      //   forkedProgram,
      //   forkTxDetails,
      //   accounts,
      //   { nonce, payload, finality },
      //   commonEmitterSequence,
      //   false,
      //   payload
      // );
      // // Validate emitter sequences.
      // commonEmitterSequence.iaddn(1);
      // const sequence = await coreBridge.EmitterSequence.fromPda(
      //   connection,
      //   program.programId,
      //   payer.publicKey
      // );
      // const forkSequence = await coreBridge.EmitterSequence.fromPda(
      //   connection,
      //   forkedProgram.programId,
      //   payer.publicKey
      // );
      // expectDeepEqual(sequence, forkSequence);
      // expectDeepEqual(sequence.sequence, commonEmitterSequence);
      // // Validate fee collector.
      // const feeCollectorData = await connection.getAccountInfo(
      //   coreBridge.FeeCollector.address(program.programId)
      // );
      // expect(feeCollectorData).is.not.null;
      // const forkFeeCollectorData = await connection.getAccountInfo(
      //   coreBridge.FeeCollector.address(program.programId)
      // );
      // expect(feeCollectorData.lamports).to.equal(forkFeeCollectorData.lamports);
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

  // Pay the fee collector prior to publishing each message.
  await Promise.all([
    expectIxOkDetails(
      connection,
      [await transferMessageFeeIx(args.new.program, payer.publicKey)],
      [payer]
    ),
    expectIxOkDetails(
      connection,
      [await transferMessageFeeIx(args.fork.program, payer.publicKey)],
      [payer]
    ),
  ]);

  return Promise.all([
    expectIxOkDetails(connection, [ix], [payer, args.new.emitterSigner, args.new.messageSigner]),
    expectIxOkDetails(
      connection,
      [forkedIx],
      [payer, args.fork.emitterSigner, args.fork.messageSigner]
    ),
  ]);
}
