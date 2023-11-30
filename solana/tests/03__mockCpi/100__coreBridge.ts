import * as anchor from "@coral-xyz/anchor";
import { expectDeepEqual, expectIxOk, expectIxOkDetails } from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import * as mockCpi from "../helpers/mockCpi";

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
    it("Invoke `mock_post_message` where Emitter == Program ID", async () => {
      const message = anchor.web3.PublicKey.findProgramAddressSync(
        [
          Buffer.from("my_message"),
          payer.publicKey.toBuffer(),
          payerSequenceValue.toBuffer("le", 16),
        ],
        program.programId
      )[0];
      const emitter = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("emitter")],
        program.programId
      )[0];
      const payerSequence = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("seq"), payer.publicKey.toBuffer()],
        program.programId
      )[0];

      const coreBridgeProgram = mockCpi.coreBridgeProgramId(program);
      const emitterSequence = coreBridge.EmitterSequence.address(
        coreBridgeProgram,
        program.programId
      );

      const { config: coreBridgeConfig, feeCollector: coreFeeCollector } =
        coreBridge.legacyPostMessageAccounts(mockCpi.getCoreBridgeProgram(program), {
          message,
          emitter: null,
          emitterSequence,
          payer: payer.publicKey,
        });

      const nonce = 420;
      const payload = Buffer.from("Where's the beef?");

      const ix = await program.methods
        .mockPostMessage({ nonce, payload })
        .accounts({
          payer: payer.publicKey,
          payerSequence,
          coreProgramEmitter: emitter,
          coreCustomEmitter: null,
          coreBridgeConfig,
          coreMessage: message,
          coreEmitterSequence: emitterSequence,
          coreFeeCollector,
          coreBridgeProgram: mockCpi.coreBridgeProgramId(program),
        })
        .instruction();

      const txDetails = await expectIxOkDetails(connection, [ix], [payer]);

      const messageData = await coreBridge.PostedMessageV1.fromAccountAddress(connection, message);
      expectDeepEqual(messageData, {
        consistencyLevel: 32,
        emitterAuthority: anchor.web3.PublicKey.default,
        status: coreBridge.MessageStatus.Published,
        _gap0: Buffer.alloc(3),
        postedTimestamp: txDetails!.blockTime!,
        nonce,
        sequence: new anchor.BN(0),
        solanaChainId: 1,
        emitter: program.programId,
        payload,
      });

      payerSequenceValue.iaddn(1);
    });

    it("Invoke `mock_post_message` where Emitter != Program ID", async () => {
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
        .mockPostMessage({ nonce, payload })
        .accounts({
          payer: payer.publicKey,
          payerSequence,
          coreProgramEmitter: null,
          coreCustomEmitter: emitter,
          coreBridgeConfig,
          coreMessage: message,
          coreEmitterSequence,
          coreFeeCollector,
          coreBridgeProgram: mockCpi.coreBridgeProgramId(program),
        })
        .instruction();

      const txDetails = await expectIxOkDetails(connection, [ix], [payer]);

      const messageData = await coreBridge.PostedMessageV1.fromAccountAddress(connection, message);
      expectDeepEqual(messageData, {
        consistencyLevel: 32,
        emitterAuthority: anchor.web3.PublicKey.default,
        status: coreBridge.MessageStatus.Published,
        _gap0: Buffer.alloc(3),
        postedTimestamp: txDetails!.blockTime!,
        nonce,
        sequence: new anchor.BN(0),
        solanaChainId: 1,
        emitter,
        payload,
      });

      payerSequenceValue.iaddn(1);
    });

    it("Invoke `mock_post_message_unreliable`", async () => {
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
        .mockPostMessageUnreliable({ nonce, payload })
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

    it("Invoke `mock_post_message_unreliable` on Same Message", async () => {
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
        .mockPostMessageUnreliable({ nonce, payload })
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

    it("Invoke `mock_prepare_message_v1` where Emitter == Program ID", async () => {
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
        status: coreBridge.MessageStatus.ReadyForPublishing,
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

    it("Invoke Legacy `post_message` with Prepared Message", async () => {
      const message = localVariables.get("message") as anchor.web3.PublicKey;

      const nonce = 69;

      const coreBridgeProgram = mockCpi.getCoreBridgeProgram(program);
      const transferIx = await coreBridge.transferMessageFeeIx(coreBridgeProgram, payer.publicKey);

      const ix = coreBridge.legacyPostMessageIx(
        coreBridgeProgram,
        {
          payer: payer.publicKey,
          message,
          emitter: null,
          emitterSequence: coreBridge.EmitterSequence.address(
            coreBridgeProgram.programId,
            program.programId
          ),
        },
        { nonce, commitment: "finalized", payload: Buffer.alloc(0) },
        {
          message: false,
        } // require other signers
      );
      const txDetails = await expectIxOkDetails(connection, [transferIx, ix], [payer]);

      const messageData = await coreBridge.PostedMessageV1.fromAccountAddress(connection, message);
      expectDeepEqual(messageData, {
        consistencyLevel: 32,
        emitterAuthority: anchor.web3.PublicKey.default,
        status: coreBridge.MessageStatus.Published,
        _gap0: Buffer.alloc(3),
        postedTimestamp: txDetails!.blockTime!,
        nonce: 420,
        sequence: new anchor.BN(1),
        solanaChainId: 1,
        emitter: program.programId,
        payload: Buffer.from("What's on draft tonight?"),
      });
    });
  });
});
