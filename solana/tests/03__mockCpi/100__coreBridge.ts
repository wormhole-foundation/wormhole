import * as anchor from "@coral-xyz/anchor";
import {
  InvalidAccountConfig,
  NullableAccountConfig,
  createIfNeeded,
  expectDeepEqual,
  expectIxErr,
  expectIxOk,
} from "../helpers";
import * as mockCpi from "../helpers/mockCpi";
import * as coreBridge from "../helpers/coreBridge";
import { expect } from "chai";

const UNRELIABLE_PAYLOAD_SIZE = 128;

const localVariables = new Map<string, any>();

describe("Mock CPI -- Core Bridge", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = mockCpi.getAnchorProgram(connection, mockCpi.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const payerSequenceValue = new anchor.BN(0);

  describe("Legacy", () => {
    it("Invoke `mock_legacy_post_message`", async () => {
      const message = anchor.web3.PublicKey.findProgramAddressSync(
        [
          Buffer.from("my_message"),
          payer.publicKey.toBuffer(),
          payerSequenceValue.toBuffer("le", 16),
        ],
        program.programId
      )[0];
      const emitter = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("my_legacy_emitter")],
        program.programId
      )[0];
      const payerSequence = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("seq"), payer.publicKey.toBuffer()],
        program.programId
      )[0];

      const {
        config: coreBridgeConfig,
        emitterSequence: coreEmitterSequence,
        feeCollector: coreFeeCollector,
      } = coreBridge.legacyPostMessageAccounts(mockCpi.getCoreBridgeProgram(program), {
        message,
        emitter,
        payer: payer.publicKey,
      });

      const nonce = 420;
      const payload = Buffer.from("Where's the beef?");

      const ix = await program.methods
        .mockLegacyPostMessage({ nonce, payload })
        .accounts({
          payer: payer.publicKey,
          payerSequence,
          coreBridgeConfig,
          coreMessage: message,
          coreEmitter: emitter,
          coreEmitterSequence,
          coreFeeCollector,
          coreBridgeProgram: mockCpi.coreBridgeProgramId(program),
        })
        .instruction();

      await expectIxOk(connection, [ix], [payer]);

      const published = await coreBridge.PostedMessageV1.fromAccountAddress(
        connection,
        message
      ).then((msg) => msg.payload);
      expectDeepEqual(published, payload);

      payerSequenceValue.iaddn(1);
    });

    it("Invoke `mock_legacy_post_message_unreliable`", async () => {
      const nonce = 420;
      const encodedNonce = Buffer.alloc(4);
      encodedNonce.writeUInt32LE(nonce, 0);

      const message = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("my_unreliable_message"), payer.publicKey.toBuffer(), encodedNonce],
        program.programId
      )[0];
      const emitter = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("my_unreliable_emitter")],
        program.programId
      )[0];

      // Same accounts as post message.
      const {
        config: coreBridgeConfig,
        emitterSequence: coreEmitterSequence,
        feeCollector: coreFeeCollector,
      } = coreBridge.legacyPostMessageAccounts(mockCpi.getCoreBridgeProgram(program), {
        message,
        emitter,
        payer: payer.publicKey,
      });

      const payload = Buffer.alloc(UNRELIABLE_PAYLOAD_SIZE);
      payload.set(Buffer.from("Where's the beef?"));

      const ix = await program.methods
        .mockLegacyPostMessageUnreliable({ nonce, payload })
        .accounts({
          payer: payer.publicKey,
          coreBridgeConfig,
          coreMessage: message,
          coreEmitter: emitter,
          coreEmitterSequence,
          coreFeeCollector,
          coreBridgeProgram: mockCpi.coreBridgeProgramId(program),
        })
        .instruction();
      await expectIxOk(connection, [ix], [payer]);

      localVariables.set("nonce", nonce);

      const published = await coreBridge.PostedMessageV1Unreliable.fromAccountAddress(
        connection,
        message
      ).then((msg) => msg.payload);
      expectDeepEqual(published, payload);
    });

    it("Invoke `mock_legacy_post_message_unreliable` on Same Message", async () => {
      const nonce = localVariables.get("nonce") as number;
      const encodedNonce = Buffer.alloc(4);
      encodedNonce.writeUInt32LE(nonce, 0);

      const message = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("my_unreliable_message"), payer.publicKey.toBuffer(), encodedNonce],
        program.programId
      )[0];
      const emitter = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("my_unreliable_emitter")],
        program.programId
      )[0];

      const {
        config: coreBridgeConfig,
        emitterSequence: coreEmitterSequence,
        feeCollector: coreFeeCollector,
      } = coreBridge.legacyPostMessageAccounts(mockCpi.getCoreBridgeProgram(program), {
        message,
        emitter,
        payer: payer.publicKey,
      });

      const payload = Buffer.alloc(UNRELIABLE_PAYLOAD_SIZE);
      payload.set(Buffer.from("Not here, m8."));

      const anotherIx = await program.methods
        .mockLegacyPostMessageUnreliable({ nonce, payload })
        .accounts({
          payer: payer.publicKey,
          coreBridgeConfig,
          coreMessage: message,
          coreEmitter: emitter,
          coreEmitterSequence,
          coreFeeCollector,
          coreBridgeProgram: mockCpi.coreBridgeProgramId(program),
        })
        .instruction();
      await expectIxOk(connection, [anotherIx], [payer]);

      {
        const published = await coreBridge.PostedMessageV1Unreliable.fromAccountAddress(
          connection,
          message
        ).then((msg) => msg.payload);
        expectDeepEqual(published, payload);
      }
    });

    it("Invoke `mock_prepare_message_v1`", async () => {
      const payerSequence = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("seq"), payer.publicKey.toBuffer()],
        program.programId
      )[0];

      const payerSequenceValue = await program.account.signerSequence
        .fetch(payerSequence)
        .then((seq) => seq.value);

      const message = anchor.web3.PublicKey.findProgramAddressSync(
        [
          Buffer.from("my_draft_message"),
          payer.publicKey.toBuffer(),
          payerSequenceValue.toBuffer("le", 16),
        ],
        program.programId
      )[0];

      const emitterAuthority = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("emitter")],
        program.programId
      )[0];

      const nonce = 420;
      const data = Buffer.from("What's on draft tonight?");

      const ix = await program.methods
        .mockPrepareMessageV1({ nonce, data })
        .accounts({
          payer: payer.publicKey,
          payerSequence,
          message,
          emitterAuthority,
          coreBridgeProgram: mockCpi.coreBridgeProgramId(program),
        })
        .instruction();
      await expectIxOk(connection, [ix], [payer]);

      const draftMessageData = await coreBridge.PostedMessageV1.fromAccountAddress(
        connection,
        message
      );
      expectDeepEqual(draftMessageData, {
        consistencyLevel: 32,
        emitterAuthority,
        status: coreBridge.MessageStatus.Finalized,
        _gap0: Buffer.alloc(3),
        postedTimestamp: 0,
        nonce,
        sequence: new anchor.BN(0),
        solanaChainId: 1,
        emitter: program.programId,
        payload: data,
      });

      localVariables.set("message", message);
    });

    it.skip("Invoke Legacy `post_message` with Prepared Message", async () => {
      const message = localVariables.get("message") as anchor.web3.PublicKey;

      const nonce = 69;
      const ix = coreBridge.legacyPostMessageIx(
        mockCpi.getCoreBridgeProgram(program),
        {
          payer: payer.publicKey,
          message,
          emitter: program.programId,
        },
        { nonce, commitment: "confirmed", payload: Buffer.alloc(0) }
      );
      await expectIxOk(connection, [ix], [payer]);

      const messageData = await coreBridge.PostedMessageV1.fromAccountAddress(connection, message);
      console.log(messageData);
    });
  });
});
