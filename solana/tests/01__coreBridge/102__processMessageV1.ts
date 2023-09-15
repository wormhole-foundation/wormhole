import * as anchor from "@coral-xyz/anchor";
import { assert, expect } from "chai";
import { createAccountIx, expectDeepEqual, expectIxErr, expectIxOk } from "../helpers";
import * as coreBridge from "../helpers/coreBridge";

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Instruction: Process Message V1", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  describe("Invalid Interaction", () => {
    const messageSize = 69;

    it("Cannot Invoke `write_message_v1` with Different Emitter Authority", async () => {
      const someoneElse = anchor.web3.Keypair.generate();

      const { draftMessage } = await initMessageV1(program, payer, messageSize);

      const ix = await coreBridge.writeMessageV1Ix(
        program,
        {
          emitterAuthority: someoneElse.publicKey,
          draftMessage,
        },
        { index: 0, data: Buffer.from("Nope.") }
      );
      await expectIxErr(connection, [ix], [payer, someoneElse], "EmitterAuthorityMismatch");
    });

    it("Cannot Invoke `write_message_v1` with Nonsensical Index", async () => {
      const { draftMessage, emitterAuthority } = await initMessageV1(program, payer, messageSize);

      const ix = await coreBridge.writeMessageV1Ix(
        program,
        {
          emitterAuthority: emitterAuthority.publicKey,
          draftMessage,
        },
        { index: messageSize, data: Buffer.from("Nope.") }
      );
      await expectIxErr(connection, [ix], [payer, emitterAuthority], "DataOverflow");
    });

    it("Cannot Invoke `write_message_v1` with Too Much Data", async () => {
      const { draftMessage, emitterAuthority } = await initMessageV1(program, payer, messageSize);

      const ix = await coreBridge.writeMessageV1Ix(
        program,
        {
          emitterAuthority: emitterAuthority.publicKey,
          draftMessage,
        },
        { index: 0, data: Buffer.alloc(messageSize + 1, "Nope.") }
      );
      await expectIxErr(connection, [ix], [payer, emitterAuthority], "DataOverflow");
    });

    it("Cannot Invoke `write_message_v1` with No Data", async () => {
      const { draftMessage, emitterAuthority } = await initMessageV1(program, payer, messageSize);

      const ix = await coreBridge.writeMessageV1Ix(
        program,
        {
          emitterAuthority: emitterAuthority.publicKey,
          draftMessage,
        },
        { index: 0, data: Buffer.alloc(0) }
      );
      await expectIxErr(connection, [ix], [payer, emitterAuthority], "InvalidInstructionArgument");
    });
  });

  describe("Ok", () => {
    const messageSize = 2 * 1_024;
    const message = Buffer.alloc(messageSize, "All your base are belong to us. ");
    const chunkSize = 912; // Max that can fit in a transaction.

    it(`Invoke \`init_message_v1\` on Message Size == ${messageSize}`, async () => {
      const { draftMessage, emitterAuthority } = await initMessageV1(program, payer, messageSize);

      localVariables.set("draftMessage", draftMessage);
      localVariables.set("emitterAuthority", emitterAuthority);
    });

    for (let start = 0; start < messageSize; start += chunkSize) {
      const end = Math.min(start + chunkSize, messageSize);

      it(`Invoke \`write_message_v1\` to Write (Range: ${start}..${end})`, async () => {
        const emitterAuthority = localVariables.get("emitterAuthority") as anchor.web3.Keypair;
        const draftMessage = localVariables.get("draftMessage") as anchor.web3.PublicKey;

        const ix = await coreBridge.writeMessageV1Ix(
          program,
          {
            emitterAuthority: emitterAuthority.publicKey,
            draftMessage,
          },
          { index: start, data: message.subarray(start, end) }
        );
        await expectIxOk(connection, [ix], [payer, emitterAuthority]);

        const expectedPayload = Buffer.alloc(messageSize);
        expectedPayload.set(message.subarray(0, end));

        const draftMessageData = await coreBridge.PostedMessageV1.fromAccountAddress(
          connection,
          draftMessage
        );
        expectDeepEqual(draftMessageData, {
          consistencyLevel: 32,
          emitterAuthority: emitterAuthority.publicKey,
          status: coreBridge.MessageStatus.Writing,
          _gap0: Buffer.alloc(3),
          postedTimestamp: 0,
          nonce: 420,
          sequence: new anchor.BN(0),
          solanaChainId: 1,
          emitter: emitterAuthority.publicKey,
          payload: expectedPayload,
        });
      });
    }

    it("Invoke `finalize_message_v1` to Finalize", async () => {
      const emitterAuthority = localVariables.get("emitterAuthority") as anchor.web3.Keypair;
      const draftMessage = localVariables.get("draftMessage") as anchor.web3.PublicKey;

      const ix = await coreBridge.finalizeMessageV1Ix(program, {
        emitterAuthority: emitterAuthority.publicKey,
        draftMessage,
      });
      await expectIxOk(connection, [ix], [payer, emitterAuthority]);
    });

    it("Cannot Invoke `finalize_message_v1` to Finalize Again", async () => {
      const emitterAuthority = localVariables.get("emitterAuthority") as anchor.web3.Keypair;
      const draftMessage = localVariables.get("draftMessage") as anchor.web3.PublicKey;

      const ix = await coreBridge.finalizeMessageV1Ix(program, {
        emitterAuthority: emitterAuthority.publicKey,
        draftMessage,
      });
      await expectIxErr(connection, [ix], [payer, emitterAuthority], "NotInWritingStatus");
    });

    it("Cannot Invoke `write_message_v1` to Write After Finalize", async () => {
      const emitterAuthority = localVariables.get("emitterAuthority") as anchor.web3.Keypair;
      const draftMessage = localVariables.get("draftMessage") as anchor.web3.PublicKey;

      const ix = await coreBridge.writeMessageV1Ix(
        program,
        {
          emitterAuthority: emitterAuthority.publicKey,
          draftMessage,
        },
        { index: 0, data: Buffer.from("Nope.") }
      );
      await expectIxErr(connection, [ix], [payer, emitterAuthority], "NotInWritingStatus");
    });

    it("Invoke `close_message_v1` to Close Message", async () => {
      const emitterAuthority = localVariables.get("emitterAuthority") as anchor.web3.Keypair;
      const draftMessage = localVariables.get("draftMessage") as anchor.web3.PublicKey;

      const closeAccountDestination = anchor.web3.Keypair.generate().publicKey;
      const balanceBefore = await connection.getBalance(closeAccountDestination);

      const expectedLamports = await connection
        .getAccountInfo(draftMessage)
        .then((acct) => acct.lamports);

      const ix = await coreBridge.closeMessageV1Ix(program, {
        emitterAuthority: emitterAuthority.publicKey,
        draftMessage,
        closeAccountDestination,
      });
      await expectIxOk(connection, [ix], [payer, emitterAuthority]);

      const balanceAfter = await connection.getBalance(closeAccountDestination);
      expect(balanceAfter - balanceBefore).to.equal(expectedLamports);

      const draftMessageData = await connection.getAccountInfo(draftMessage);
      expect(draftMessageData).is.null;
    });
  });
});

async function initMessageV1(
  program: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  messageSize: number
) {
  const draftMessage = anchor.web3.Keypair.generate();
  const createIx = await createAccountIx(
    program.provider.connection,
    program.programId,
    payer,
    draftMessage,
    95 + messageSize
  );

  const emitterAuthority = anchor.web3.Keypair.generate();
  const initIx = await coreBridge.initMessageV1Ix(
    program,
    { emitterAuthority: emitterAuthority.publicKey, draftMessage: draftMessage.publicKey },
    { nonce: 420, commitment: "finalized", cpiProgramId: null }
  );

  await expectIxOk(
    program.provider.connection,
    [createIx, initIx],
    [payer, emitterAuthority, draftMessage]
  );

  return { draftMessage: draftMessage.publicKey, emitterAuthority };
}
