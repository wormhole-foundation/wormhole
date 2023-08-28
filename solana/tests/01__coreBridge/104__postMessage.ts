import * as anchor from "@coral-xyz/anchor";
import {
  createAccountIx,
  expectDeepEqual,
  expectIxErr,
  expectIxOk,
  expectIxOkDetails,
} from "../helpers";
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

      await everythingOk(program, payer, message, new anchor.BN(3), payer);
    });

    it("Invoke Legacy `post_message` With Prepared Message", async () => {
      const message = Buffer.alloc(5 * 1_024, "All your base are belong to us. ");

      const { draftMessage, emitterAuthority } = await everythingOk(
        program,
        payer,
        message,
        new anchor.BN(0)
      );

      // Save for next test.
      localVariables.set("draftMessage", draftMessage);
      localVariables.set("emitterAuthority", emitterAuthority);
    });

    it("Cannot Invoke Legacy `post_message` With Same Prepared Message", async () => {
      const draftMessage = localVariables.get("draftMessage") as anchor.web3.Keypair;
      const emitterAuthority = localVariables.get("emitterAuthority") as anchor.web3.Keypair;

      const nonce = 420;
      const finality = coreBridge.Finality.Confirmed;
      const ix = coreBridge.legacyPostMessageIx(
        program,
        {
          message: draftMessage.publicKey,
          emitter: emitterAuthority.publicKey,
          payer: payer.publicKey,
        },
        { nonce, finality, payload: Buffer.alloc(0) }
      );
      await expectIxErr(
        connection,
        [ix],
        [payer, emitterAuthority, draftMessage],
        "MessageAlreadyPublished"
      );
    });
  });
});

async function everythingOk(
  program: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  message: Buffer,
  sequence: anchor.BN,
  emitterAuthority?: anchor.web3.Keypair
) {
  if (emitterAuthority === undefined) {
    emitterAuthority = anchor.web3.Keypair.generate();
  }

  const { draftMessage } = await initAndProcessMessageV1(program, payer, message, emitterAuthority);

  const nonce = 420;
  const finality = coreBridge.Finality.Confirmed;
  await coreBridge.expectOkPostMessage(
    program,
    { payer, message: draftMessage, emitter: emitterAuthority },
    { nonce, finality, payload: Buffer.alloc(0) },
    sequence,
    message
  );

  sequence.iaddn(1);

  return { draftMessage, emitterAuthority };
}

async function initAndProcessMessageV1(
  program: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  message: Buffer,
  emitterAuthority?: anchor.web3.Keypair
) {
  if (emitterAuthority === undefined) {
    emitterAuthority = anchor.web3.Keypair.generate();
  }

  const messageLen = message.length;

  const connection = program.provider.connection;

  const draftMessage = anchor.web3.Keypair.generate();
  const createIx = await createAccountIx(
    program.provider.connection,
    program.programId,
    payer,
    draftMessage,
    95 + message.length
  );

  const initIx = await coreBridge.initMessageV1Ix(
    program,
    { emitterAuthority: emitterAuthority.publicKey, draftMessage: draftMessage.publicKey },
    { cpiProgramId: null }
  );

  const endAfterInit = 745;
  const firstProcessIx = await coreBridge.processMessageV1Ix(
    program,
    {
      emitterAuthority: emitterAuthority.publicKey,
      draftMessage: draftMessage.publicKey,
      closeAccountDestination: null,
    },
    { write: { index: 0, data: message.subarray(0, endAfterInit) } }
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

      const writeIx = await coreBridge.processMessageV1Ix(
        program,
        {
          emitterAuthority: emitterAuthority.publicKey,
          draftMessage: draftMessage.publicKey,
          closeAccountDestination: null,
        },
        { write: { index: start, data: message.subarray(start, end) } }
      );

      if (end == messageLen) {
        const finalizeIx = await coreBridge.processMessageV1Ix(
          program,
          {
            emitterAuthority: emitterAuthority.publicKey,
            draftMessage: draftMessage.publicKey,
            closeAccountDestination: null,
          },
          { finalize: {} }
        );
        await expectIxOk(connection, [writeIx, finalizeIx], [payer, emitterAuthority]);
      } else {
        await expectIxOk(connection, [writeIx], [payer, emitterAuthority]);
      }
    }
  } else {
    const finalizeIx = await coreBridge.processMessageV1Ix(
      program,
      {
        emitterAuthority: emitterAuthority.publicKey,
        draftMessage: draftMessage.publicKey,
        closeAccountDestination: null,
      },
      { finalize: {} }
    );

    await expectIxOk(
      connection,
      [createIx, initIx, firstProcessIx, finalizeIx],
      [payer, emitterAuthority, draftMessage]
    );
  }

  return { draftMessage, emitterAuthority };
}
