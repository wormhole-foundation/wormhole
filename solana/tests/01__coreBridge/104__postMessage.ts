import * as anchor from "@coral-xyz/anchor";
import { createAccountIx, expectIxErr, expectIxOk } from "../helpers";
import * as coreBridge from "../helpers/coreBridge";

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Legacy Instruction: Post Message (Prepared)", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  describe("Invalid Accounts", () => {
    // TODO
  });

  describe("Ok", () => {
    it.skip("Cannot Invoke Legacy `post_message` with Non-Empty Payload on Prepared Message", async () => {
      // TODO
    });

    it.skip("Cannot Invoke Legacy `post_message` Different Emitter Authority on Prepared Message", async () => {
      // TODO
    });

    it("Invoke Legacy `post_message` With Payer as Emitter", async () => {
      const message = Buffer.from("I'm the captain now. ");

      const nonce = 420;
      const commitment = "confirmed";

      await everythingOk(program, payer, message, nonce, commitment, new anchor.BN(3), payer);
    });

    it("Invoke Legacy `post_message` With Prepared Message", async () => {
      const payload = Buffer.alloc(5 * 1_024, "All your base are belong to us. ");

      const nonce = 420;
      const commitment = "confirmed";

      const { message, emitter } = await everythingOk(
        program,
        payer,
        payload,
        nonce,
        commitment,
        new anchor.BN(0)
      );

      // Save for next test.
      localVariables.set("message", message);
      localVariables.set("emitter", emitter);
    });

    it("Cannot Invoke Legacy `post_message` With Same Prepared Message", async () => {
      const message = localVariables.get("message") as anchor.web3.PublicKey;
      const emitter = localVariables.get("emitter") as anchor.web3.PublicKey;
      const emitterSequence = coreBridge.EmitterSequence.address(program.programId, emitter);

      // Intentionally different from how the message was prepared.
      const nonce = 0;
      const commitment = "finalized";
      const ix = coreBridge.legacyPostMessageIx(
        program,
        {
          message,
          emitter: null,
          emitterSequence,
          payer: payer.publicKey,
        },
        { nonce, commitment, payload: Buffer.alloc(0) },
        { message: false }
      );
      await expectIxErr(connection, [ix], [payer], "NotReadyForPublishing");
    });

    it("Invoke `post_message` With 30Kb Payload", async () => {
      const { draftMessage, createIx, initIx } = await createAndInitMessageV1Ixes(
        program,
        payer,
        30 * 1_024,
        payer,
        420, // nonce
        "confirmed"
      );

      const finalizeIx = await coreBridge.finalizeMessageV1Ix(program, {
        emitterAuthority: payer.publicKey,
        draftMessage: draftMessage.publicKey,
      });

      const transferFeeIx = await coreBridge.transferMessageFeeIx(program, payer.publicKey);

      const emitterSequence = coreBridge.EmitterSequence.address(
        program.programId,
        payer.publicKey
      );

      const postIx = coreBridge.legacyPostMessageIx(
        program,
        {
          message: draftMessage.publicKey,
          emitter: null,
          emitterSequence,
          payer: payer.publicKey,
        },
        { nonce: 0, commitment: "confirmed", payload: Buffer.alloc(0) },
        { message: false }
      );
      await expectIxOk(
        connection,
        [createIx, initIx, finalizeIx, transferFeeIx, postIx],
        [payer, draftMessage]
      );
    });
  });
});

async function everythingOk(
  program: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  payload: Buffer,
  nonce: number,
  commitment: anchor.web3.Commitment,
  sequence: anchor.BN,
  emitterAuthority?: anchor.web3.Keypair
) {
  if (emitterAuthority === undefined) {
    emitterAuthority = anchor.web3.Keypair.generate();
  }

  const { draftMessage } = await initAndProcessMessageV1(
    program,
    payer,
    payload,
    nonce,
    commitment,
    emitterAuthority
  );

  const emitterSequence = await coreBridge.PostedMessageV1.fromAccountAddress(
    program.provider.connection,
    draftMessage
  ).then((msg) => coreBridge.EmitterSequence.address(program.programId, msg.emitter));

  await coreBridge.expectOkPostMessage(
    program,
    { payer, message: null, emitter: null },
    { nonce, commitment, payload: Buffer.alloc(0) },
    sequence,
    {
      nonce,
      consistencyLevel: 1,
      payload,
      message: draftMessage,
      emitter: emitterAuthority.publicKey,
    },
    undefined,
    emitterSequence,
    false // createTransferFeeIx
  );

  sequence.iaddn(1);

  return { message: draftMessage, emitter: emitterAuthority.publicKey };
}

async function createAndInitMessageV1Ixes(
  program: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  messageLen: number,
  emitterAuthority: anchor.web3.Keypair,
  nonce: number,
  commitment: anchor.web3.Commitment
) {
  const draftMessage = anchor.web3.Keypair.generate();
  const createIx = await createAccountIx(
    program.provider.connection,
    program.programId,
    payer,
    draftMessage,
    95 + messageLen
  );

  const initIx = await coreBridge.initMessageV1Ix(
    program,
    { emitterAuthority: emitterAuthority.publicKey, draftMessage: draftMessage.publicKey },
    { nonce, commitment, cpiProgramId: null }
  );

  return { draftMessage, createIx, initIx };
}

async function initAndProcessMessageV1(
  program: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  payload: Buffer,
  nonce: number,
  commitment: anchor.web3.Commitment,
  emitterAuthority?: anchor.web3.Keypair
) {
  if (emitterAuthority === undefined) {
    emitterAuthority = anchor.web3.Keypair.generate();
  }

  const messageLen = payload.length;

  const { draftMessage, createIx, initIx } = await createAndInitMessageV1Ixes(
    program,
    payer,
    messageLen,
    emitterAuthority,
    nonce,
    commitment
  );

  const connection = program.provider.connection;

  const endAfterInit = 740;
  const firstProcessIx = await coreBridge.writeMessageV1Ix(
    program,
    {
      emitterAuthority: emitterAuthority.publicKey,
      draftMessage: draftMessage.publicKey,
    },
    { index: 0, data: payload.subarray(0, endAfterInit) }
  );

  if (messageLen > endAfterInit) {
    await expectIxOk(
      connection,
      [createIx, initIx, firstProcessIx],
      [payer, emitterAuthority, draftMessage]
    );

    const chunkSize = 912;
    for (let start = endAfterInit; start < messageLen; start += chunkSize) {
      const end = Math.min(start + chunkSize, messageLen);

      const writeIx = await coreBridge.writeMessageV1Ix(
        program,
        {
          emitterAuthority: emitterAuthority.publicKey,
          draftMessage: draftMessage.publicKey,
        },
        { index: start, data: payload.subarray(start, end) }
      );

      if (end == messageLen) {
        const finalizeIx = await coreBridge.finalizeMessageV1Ix(program, {
          emitterAuthority: emitterAuthority.publicKey,
          draftMessage: draftMessage.publicKey,
        });
        await expectIxOk(connection, [writeIx, finalizeIx], [payer, emitterAuthority]);
      } else {
        await expectIxOk(connection, [writeIx], [payer, emitterAuthority]);
      }
    }
  } else {
    const finalizeIx = await coreBridge.finalizeMessageV1Ix(program, {
      emitterAuthority: emitterAuthority.publicKey,
      draftMessage: draftMessage.publicKey,
    });

    await expectIxOk(
      connection,
      [createIx, initIx, firstProcessIx, finalizeIx],
      [payer, emitterAuthority, draftMessage]
    );
  }

  return { draftMessage: draftMessage.publicKey, emitterAuthority };
}
