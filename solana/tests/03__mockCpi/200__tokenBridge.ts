import * as anchor from "@coral-xyz/anchor";
import {
  InvalidAccountConfig,
  MINT_INFO_9,
  NullableAccountConfig,
  createIfNeeded,
  expectDeepEqual,
  expectIxErr,
  expectIxOk,
} from "../helpers";
import * as mockCpi from "../helpers/mockCpi";
import * as coreBridge from "../helpers/coreBridge";
import * as tokenBridge from "../helpers/tokenBridge";
import {
  createAssociatedTokenAccount,
  getAccount,
  getAssociatedTokenAddressSync,
  mintTo,
} from "@solana/spl-token";
import { parseTokenTransferPayload } from "@certusone/wormhole-sdk";
import { expect } from "chai";

describe("Core Bridge -- Legacy Instruction: Post Message", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = mockCpi.getAnchorProgram(connection, mockCpi.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  before("Set Up Mints and Token Accounts", async () => {
    const { mint } = MINT_INFO_9;
    //const token = getAssociatedTokenAddressSync(mint, payer.publicKey);
    const token = await createAssociatedTokenAccount(connection, payer, mint, payer.publicKey);

    await mintTo(connection, payer, mint, token, payer, BigInt("1000000000000000000"));
  });

  describe("Legacy", () => {
    it("Invoke `mock_legacy_transfer_tokens_with_payload_native` where Sender == Program ID", async () => {
      const { mint } = MINT_INFO_9;
      const srcToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

      const { payerSequence, coreMessage } = await getPayerSequenceAndMessage(
        program,
        payer.publicKey
      );

      const programSenderAuthority = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("sender")],
        program.programId
      )[0];
      const {
        custodyToken: tokenBridgeCustodyToken,
        transferAuthority: tokenBridgeTransferAuthority,
        custodyAuthority: tokenBridgeCustodyAuthority,
        coreBridgeConfig,
        coreEmitter,
        coreEmitterSequence,
        coreFeeCollector,
      } = tokenBridge.legacyTransferTokensWithPayloadNativeAccounts(
        mockCpi.getTokenBridgeProgram(program),
        {
          payer: payer.publicKey,
          srcToken,
          mint,
          coreMessage,
          senderAuthority: programSenderAuthority,
        }
      );

      const nonce = 420;
      const amount = new anchor.BN(6942069);
      const redeemer = Array.from(
        Buffer.from("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "hex")
      );
      const redeemerChain = 69;
      const payload = Buffer.from("Where's the beef?");

      const approveIx = tokenBridge.approveTransferAuthorityIx(
        mockCpi.getTokenBridgeProgram(program),
        srcToken,
        payer.publicKey,
        amount
      );

      const ix = await program.methods
        .mockLegacyTransferTokensWithPayloadNative({
          nonce,
          amount,
          redeemer,
          redeemerChain,
          payload,
        })
        .accounts({
          payer: payer.publicKey,
          payerSequence,
          tokenBridgeProgramSenderAuthority: programSenderAuthority,
          tokenBridgeCustomSenderAuthority: null,
          srcToken,
          mint,
          tokenBridgeCustodyToken,
          tokenBridgeTransferAuthority,
          tokenBridgeCustodyAuthority,
          coreBridgeConfig,
          coreMessage,
          coreEmitter,
          coreEmitterSequence,
          coreFeeCollector,
          coreBridgeProgram: mockCpi.coreBridgeProgramId(program),
          tokenBridgeProgram: mockCpi.tokenBridgeProgramId(program),
        })
        .instruction();

      const balanceBefore = await getAccount(connection, srcToken).then((acct) => acct.amount);

      await expectIxOk(connection, [approveIx, ix], [payer]);

      const balanceAfter = await getAccount(connection, srcToken).then((acct) => acct.amount);

      const expectedBalanceChange = BigInt(amount.divn(10).muln(10).toString());
      expect(balanceBefore - balanceAfter).equals(expectedBalanceChange);

      const transferMsg = await coreBridge.PostedMessageV1.fromAccountAddress(
        connection,
        coreMessage
      ).then((msg) => parseTokenTransferPayload(msg.payload));
      expectDeepEqual(new anchor.web3.PublicKey(transferMsg.fromAddress), program.programId);
    });

    it("Invoke `mock_legacy_transfer_tokens_with_payload_native` where Sender != Program ID", async () => {
      const { mint } = MINT_INFO_9;
      const srcToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

      const { payerSequence, coreMessage } = await getPayerSequenceAndMessage(
        program,
        payer.publicKey
      );

      const customSenderAuthority = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("custom_sender_authority")],
        program.programId
      )[0];
      const {
        custodyToken: tokenBridgeCustodyToken,
        transferAuthority: tokenBridgeTransferAuthority,
        custodyAuthority: tokenBridgeCustodyAuthority,
        coreBridgeConfig,
        coreEmitter,
        coreEmitterSequence,
        coreFeeCollector,
      } = tokenBridge.legacyTransferTokensWithPayloadNativeAccounts(
        mockCpi.getTokenBridgeProgram(program),
        {
          payer: payer.publicKey,
          srcToken,
          mint,
          coreMessage,
          senderAuthority: customSenderAuthority,
        }
      );

      const nonce = 420;
      const amount = new anchor.BN(6942069);
      const redeemer = Array.from(
        Buffer.from("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "hex")
      );
      const redeemerChain = 69;
      const payload = Buffer.from("Where's the beef?");

      const approveIx = tokenBridge.approveTransferAuthorityIx(
        mockCpi.getTokenBridgeProgram(program),
        srcToken,
        payer.publicKey,
        amount
      );

      const ix = await program.methods
        .mockLegacyTransferTokensWithPayloadNative({
          nonce,
          amount,
          redeemer,
          redeemerChain,
          payload,
        })
        .accounts({
          payer: payer.publicKey,
          payerSequence,
          tokenBridgeProgramSenderAuthority: null,
          tokenBridgeCustomSenderAuthority: customSenderAuthority,
          srcToken,
          mint,
          tokenBridgeCustodyToken,
          tokenBridgeTransferAuthority,
          tokenBridgeCustodyAuthority,
          coreBridgeConfig,
          coreMessage,
          coreEmitter,
          coreEmitterSequence,
          coreFeeCollector,
          coreBridgeProgram: mockCpi.coreBridgeProgramId(program),
          tokenBridgeProgram: mockCpi.tokenBridgeProgramId(program),
        })
        .instruction();

      const balanceBefore = await getAccount(connection, srcToken).then((acct) => acct.amount);

      await expectIxOk(connection, [approveIx, ix], [payer]);

      const balanceAfter = await getAccount(connection, srcToken).then((acct) => acct.amount);

      const expectedBalanceChange = BigInt(amount.divn(10).muln(10).toString());
      expect(balanceBefore - balanceAfter).equals(expectedBalanceChange);

      const transferMsg = await coreBridge.PostedMessageV1.fromAccountAddress(
        connection,
        coreMessage
      ).then((msg) => parseTokenTransferPayload(msg.payload));
      expectDeepEqual(new anchor.web3.PublicKey(transferMsg.fromAddress), customSenderAuthority);
    });
  });
});

async function getPayerSequenceAndMessage(
  program: mockCpi.MockCpiProgram,
  payer: anchor.web3.PublicKey
) {
  const payerSequence = anchor.web3.PublicKey.findProgramAddressSync(
    [Buffer.from("seq"), payer.toBuffer()],
    program.programId
  )[0];

  const payerSequenceValue = await program.account.signerSequence
    .fetch(payerSequence)
    .then((acct) => acct.value);

  const coreMessage = anchor.web3.PublicKey.findProgramAddressSync(
    [Buffer.from("my_message"), payerSequenceValue.toBuffer("le", 16)],
    program.programId
  )[0];

  return { payerSequence, coreMessage };
}
