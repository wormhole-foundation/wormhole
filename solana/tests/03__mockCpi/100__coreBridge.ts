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

describe("Core Bridge -- Legacy Instruction: Post Message", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = mockCpi.getAnchorProgram(connection, mockCpi.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const payerSequenceValue = new anchor.BN(0);

  describe("Legacy", () => {
    it("Invoke `mock_legacy_post_message`", async () => {
      const message = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("my_message"), payerSequenceValue.toBuffer("le", 16)],
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

      payerSequenceValue.iaddn(1);
    });
  });
});
