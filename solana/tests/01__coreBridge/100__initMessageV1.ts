import * as anchor from "@coral-xyz/anchor";
import { createAccountIx, expectDeepEqual, expectIxErr, expectIxOk } from "../helpers";
import * as coreBridge from "../helpers/coreBridge";

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Instruction: Init Message V1", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  describe("Invalid Interaction", () => {
    it("Cannot Invoke `init_message_v1` without Created Account", async () => {
      const emitterAuthority = anchor.web3.Keypair.generate();
      const draftMessage = anchor.web3.Keypair.generate();

      const initIx = await coreBridge.initMessageV1Ix(
        program,
        { emitterAuthority: emitterAuthority.publicKey, draftMessage: draftMessage.publicKey },
        defaultArgs()
      );
      await expectIxErr(connection, [initIx], [payer, emitterAuthority], "ConstraintOwner");
    });

    it("Cannot Invoke `init_message_v1` with Some(cpi_program_id)", async () => {
      const invalidArgs = {
        nonce: 420,
        commitment: "finalized" as anchor.web3.Commitment,
        cpiProgramId: anchor.web3.Keypair.generate().publicKey,
      };
      const { draftMessage, emitterAuthority, instructions } = await createIxs(
        program,
        payer,
        69,
        invalidArgs
      );

      await expectIxErr(
        connection,
        instructions,
        [payer, emitterAuthority, draftMessage],
        "InvalidProgramEmitter"
      );
    });

    it("Cannot Invoke `init_message_v1` with Nonsensical Account Size", async () => {
      const emitterAuthority = anchor.web3.Keypair.generate();
      const cpiProgramId = anchor.web3.Keypair.generate().publicKey;

      const draftMessage = anchor.web3.Keypair.generate();
      const createIx = await createAccountIx(
        program.provider.connection,
        program.programId,
        payer,
        draftMessage,
        94 // one less than the minimum without a payload
      );

      const initIx = await coreBridge.initMessageV1Ix(
        program,
        { emitterAuthority: emitterAuthority.publicKey, draftMessage: draftMessage.publicKey },
        defaultArgs()
      );
      await expectIxErr(
        connection,
        [createIx, initIx],
        [payer, emitterAuthority, draftMessage],
        "InvalidCreatedAccountSize"
      );
    });

    it("Cannot Invoke `init_message_v1` with Expected Message Size == 0", async () => {
      const { draftMessage, emitterAuthority, instructions } = await createIxs(program, payer, 0);

      await expectIxErr(
        connection,
        instructions,
        [payer, emitterAuthority, draftMessage],
        "InvalidCreatedAccountSize"
      );
    });

    it("Cannot Invoke `init_message_v1` with Expected Message Size > 30KB", async () => {
      const { draftMessage, emitterAuthority, instructions } = await createIxs(
        program,
        payer,
        30 * 1_024 + 1
      );

      await expectIxErr(
        connection,
        instructions,
        [payer, emitterAuthority, draftMessage],
        "ExceedsMaxPayloadSize"
      );
    });
  });

  describe("Ok", () => {
    const messageSizes = [1, 69, 30 * 1_024];

    for (const messageSize of messageSizes) {
      it(`Invoke \`init_message_v1\` with Message Size == ${messageSize}`, async () => {
        const { draftMessage, emitterAuthority, instructions } = await createIxs(
          program,
          payer,
          messageSize
        );

        await expectIxOk(connection, instructions, [payer, emitterAuthority, draftMessage]);

        // This checks the discriminator, too.
        const draftMessageData = await coreBridge.PostedMessageV1.fromAccountAddress(
          connection,
          draftMessage.publicKey
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
          payload: Buffer.alloc(messageSize),
        });

        // Only pick one for the next test.
        if (messageSize == 1) {
          localVariables.set("draftMessage", draftMessage);
          localVariables.set("emitterAuthority", emitterAuthority);
        }
      });
    }

    it("Cannot Invoke `init_message_v1` with Same Draft Message", async () => {
      const draftMessage = localVariables.get("draftMessage") as anchor.web3.Keypair;
      const emitterAuthority = localVariables.get("emitterAuthority") as anchor.web3.Keypair;

      const initIx = await coreBridge.initMessageV1Ix(
        program,
        { emitterAuthority: emitterAuthority.publicKey, draftMessage: draftMessage.publicKey },
        defaultArgs()
      );
      await expectIxErr(connection, [initIx], [payer, emitterAuthority], "AccountNotZeroed");
    });
  });
});

function defaultArgs() {
  return { nonce: 420, commitment: "finalized" as anchor.web3.Commitment, cpiProgramId: null };
}

async function prepareDraftMessage(
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

  return {
    draftMessage,
    createIx,
  };
}

async function createIxs(
  program: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  messageSize: number,
  args: coreBridge.InitMessageV1Args = defaultArgs()
) {
  const { draftMessage, createIx } = await prepareDraftMessage(program, payer, messageSize);

  const emitterAuthority = anchor.web3.Keypair.generate();
  const initIx = await coreBridge.initMessageV1Ix(
    program,
    { emitterAuthority: emitterAuthority.publicKey, draftMessage: draftMessage.publicKey },
    args
  );

  return { draftMessage, emitterAuthority, instructions: [createIx, initIx] };
}
