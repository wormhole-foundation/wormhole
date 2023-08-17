import * as anchor from "@coral-xyz/anchor";
import { expect } from "chai";
import { createAccountIx, expectDeepEqual, expectIxErr, expectIxOk } from "../helpers";
import * as coreBridge from "../helpers/coreBridge";

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Instruction: Process Encoded Vaa", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  describe("Invalid Interaction", () => {
    const messageSize = 69;

    it.skip("Cannot Invoke `process_encoded_vaa` with Different Emitter Authority", async () => {
      const someoneElse = anchor.web3.Keypair.generate();

      const encodedVaa = await initEncodedVaa(program, payer, messageSize);

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: someoneElse.publicKey,
          encodedVaa,
          guardianSet: null,
        },
        { write: { index: 0, data: Buffer.from("Nope.") } }
      );
      await expectIxErr(connection, [ix], [payer, someoneElse], "EmitterAuthorityMismatch");
    });

    it.skip("Cannot Invoke `process_encoded_vaa` with Nonsensical Index", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, messageSize);

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
          guardianSet: null,
        },
        { write: { index: messageSize, data: Buffer.from("Nope.") } }
      );
      await expectIxErr(connection, [ix], [payer], "DataOverflow");
    });

    it.skip("Cannot Invoke `process_encoded_vaa` with Too Much Data", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, messageSize);

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
          guardianSet: null,
        },
        { write: { index: 0, data: Buffer.alloc(messageSize + 1, "Nope.") } }
      );
      await expectIxErr(connection, [ix], [payer], "DataOverflow");
    });

    it.skip("Cannot Invoke `process_encoded_vaa` with No Data", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, messageSize);

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
          guardianSet: null,
        },
        { write: { index: 0, data: Buffer.alloc(0) } }
      );
      await expectIxErr(connection, [ix], [payer], "InvalidInstructionArgument");
    });
  });

  describe("Ok", () => {
    const messageSize = 2 * 1_024;
    const message = Buffer.alloc(messageSize, "All your base are belong to us. ");
    const chunkSize = 912; // Max that can fit in a transaction.

    it.skip(`Invoke \`init_message_v1\` on VAA Size == ${messageSize}`, async () => {
      const encodedVaa = await initEncodedVaa(program, payer, messageSize);

      localVariables.set("encodedVaa", encodedVaa);
    });

    for (let start = 0; start < message.length; start += chunkSize) {
      const end = Math.min(start + chunkSize, message.length);

      it.skip(`Invoke \`process_encoded_vaa\` to Write Part of VAA (Range: ${start}..${end})`, async () => {
        const writeAuthority: anchor.web3.Keypair = localVariables.get("writeAuthority")!;
        const encodedVaa: anchor.web3.PublicKey = localVariables.get("encodedVaa")!;

        const ix = await coreBridge.processEncodedVaaIx(
          program,
          {
            writeAuthority: writeAuthority.publicKey,
            encodedVaa,
            guardianSet: null,
          },
          { write: { index: start, data: message.subarray(start, end) } }
        );
        await expectIxOk(connection, [ix], [payer, writeAuthority]);

        const expectedPayload = Buffer.alloc(messageSize);
        expectedPayload.set(message.subarray(0, end));

        // const draftMessageData = await coreBridge.PostedMessageV1.fromAccountAddress(
        //   connection,
        //   encodedVaa
        // );
        // expectDeepEqual(draftMessageData, {
        //   finality: 0,
        //   writeAuthority: writeAuthority.publicKey,
        //   status: coreBridge.MessageStatus.Writing,
        //   _gap0: Buffer.alloc(3),
        //   postedTimestamp: 0,
        //   nonce: 0,
        //   sequence: new anchor.BN(0),
        //   solanaChainId: 1,
        //   emitter: writeAuthority.publicKey,
        //   payload: expectedPayload,
        // });
      });
    }

    it.skip("Invoke `process_encoded_vaa` to Close Encoded VAA", async () => {
      const writeAuthority: anchor.web3.Keypair = localVariables.get("writeAuthority")!;
      const encodedVaa: anchor.web3.PublicKey = localVariables.get("encodedVaa")!;

      const guardianSet = anchor.web3.Keypair.generate().publicKey;
      const balanceBefore = await connection.getBalance(guardianSet);

      const expectedLamports = await connection
        .getAccountInfo(encodedVaa)
        .then((acct) => acct.lamports);

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: writeAuthority.publicKey,
          encodedVaa,
          guardianSet,
        },
        { closeVaaAccount: {} }
      );
      await expectIxOk(connection, [ix], [payer, writeAuthority]);

      const balanceAfter = await connection.getBalance(guardianSet);
      expect(balanceAfter - balanceBefore).to.equal(expectedLamports);

      const draftMessageData = await connection.getAccountInfo(encodedVaa);
      expect(draftMessageData).is.null;
    });
  });
});

async function initEncodedVaa(
  program: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  messageSize: number
) {
  const encodedVaa = anchor.web3.Keypair.generate();
  const createIx = await createAccountIx(
    program.provider.connection,
    program.programId,
    payer,
    encodedVaa,
    95 + messageSize
  );

  const initIx = await coreBridge.initEncodedVaaIx(program, {
    writeAuthority: payer.publicKey,
    encodedVaa: encodedVaa.publicKey,
  });

  await expectIxOk(program.provider.connection, [createIx, initIx], [payer, encodedVaa]);

  return encodedVaa.publicKey;
}
