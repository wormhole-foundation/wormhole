import * as anchor from "@coral-xyz/anchor";
import {
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  GUARDIAN_KEYS,
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
import {
  CHAIN_ID_SOLANA,
  parseTokenTransferPayload,
  tryNativeToHexString,
} from "@certusone/wormhole-sdk";
import { expect } from "chai";
import { MockGuardians, MockTokenBridge } from "@certusone/wormhole-sdk/lib/cjs/mock";

const GUARDIAN_SET_INDEX = 2;
const foreignTokenBridge = new MockTokenBridge(
  tryNativeToHexString(ETHEREUM_TOKEN_BRIDGE_ADDRESS, 2),
  2,
  1,
  3_200_000
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

describe("Mock CPI -- Token Bridge", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = mockCpi.getAnchorProgram(connection, mockCpi.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  before("Set Up Mints and Token Accounts", async () => {
    const { mint } = MINT_INFO_9;
    const token = getAssociatedTokenAddressSync(mint, payer.publicKey);
    //const token = await createAssociatedTokenAccount(connection, payer, mint, payer.publicKey);

    await mintTo(connection, payer, mint, token, payer, BigInt("1000000000000000000"));
  });

  describe("Legacy", () => {
    it("Invoke `mock_legacy_transfer_tokens_native`", async () => {
      const { mint } = MINT_INFO_9;
      const srcToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

      const { payerSequence, coreMessage } = await getPayerSequenceAndMessage(
        program,
        payer.publicKey
      );

      const {
        custodyToken: tokenBridgeCustodyToken,
        transferAuthority: tokenBridgeTransferAuthority,
        custodyAuthority: tokenBridgeCustodyAuthority,
        coreBridgeConfig,
        coreEmitter,
        coreEmitterSequence,
        coreFeeCollector,
      } = tokenBridge.legacyTransferTokensNativeAccounts(mockCpi.getTokenBridgeProgram(program), {
        payer: payer.publicKey,
        srcToken,
        mint,
        coreMessage,
      });

      const nonce = 420;
      const amount = new anchor.BN(6942069);
      const recipient = Array.from(
        Buffer.from("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "hex")
      );
      const recipientChain = 69;

      const approveIx = tokenBridge.approveTransferAuthorityIx(
        mockCpi.getTokenBridgeProgram(program),
        srcToken,
        payer.publicKey,
        amount
      );

      const ix = await program.methods
        .mockLegacyTransferTokensNative({
          nonce,
          amount,
          recipient,
          recipientChain,
        })
        .accounts({
          payer: payer.publicKey,
          payerSequence,
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
    });

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

    it.skip("Invoke `mock_legacy_complete_transfer_native` where Redeemer == Program ID", async () => {
      const { mint } = MINT_INFO_9;

      const encodedAmount = new anchor.BN(694206);
      const payload = Buffer.from("Where's the beef?");
      const signedVaa = getSignedTransferWithPayloadVaa(
        mint,
        encodedAmount,
        program.programId,
        payload
      );
      // const dstToken = getAssociatedTokenAddressSync(mint, payer.publicKey);
      // const {
      //   custodyToken: tokenBridgeCustodyToken,
      //   custodyAuthority: tokenBridgeCustodyAuthority,
      // } = tokenBridge.legacyCompleteTransferWithPayloadNativeAccounts(
      //   mockCpi.getTokenBridgeProgram(program),
      //   {
      //     payer: payer.publicKey,
      //     dstToken,
      //     mint,
      //   }
      // );
      // const nonce = 420;
      // const amount = new anchor.BN(6942069);
      // const recipient = Array.from(
      //   Buffer.from("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "hex")
      // );
      // const recipientChain = 69;
      // const approveIx = tokenBridge.approveTransferAuthorityIx(
      //   mockCpi.getTokenBridgeProgram(program),
      //   srcToken,
      //   payer.publicKey,
      //   amount
      // );
      // const ix = await program.methods
      //   .mockLegacyTransferTokensNative({
      //     nonce,
      //     amount,
      //     recipient,
      //     recipientChain,
      //   })
      //   .accounts({
      //     payer: payer.publicKey,
      //     payerSequence,
      //     srcToken,
      //     mint,
      //     tokenBridgeCustodyToken,
      //     tokenBridgeTransferAuthority,
      //     tokenBridgeCustodyAuthority,
      //     coreBridgeConfig,
      //     coreMessage,
      //     coreEmitter,
      //     coreEmitterSequence,
      //     coreFeeCollector,
      //     coreBridgeProgram: mockCpi.coreBridgeProgramId(program),
      //     tokenBridgeProgram: mockCpi.tokenBridgeProgramId(program),
      //   })
      //   .instruction();
      // const balanceBefore = await getAccount(connection, srcToken).then((acct) => acct.amount);
      // await expectIxOk(connection, [approveIx, ix], [payer]);
      // const balanceAfter = await getAccount(connection, srcToken).then((acct) => acct.amount);
      // const expectedBalanceChange = BigInt(amount.divn(10).muln(10).toString());
      // expect(balanceBefore - balanceAfter).equals(expectedBalanceChange);
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
    [Buffer.from("my_message"), payer.toBuffer(), payerSequenceValue.toBuffer("le", 16)],
    program.programId
  )[0];

  return { payerSequence, coreMessage };
}

function getSignedTransferWithPayloadVaa(
  mint: anchor.web3.PublicKey,
  encodedAmount: anchor.BN,
  redeemer: anchor.web3.PublicKey,
  payload: Buffer,
  targetChain?: number
): Buffer {
  const published = foreignTokenBridge.publishTransferTokensWithPayload(
    mint.toBuffer().toString("hex"),
    CHAIN_ID_SOLANA,
    BigInt(encodedAmount.toString()),
    targetChain ?? CHAIN_ID_SOLANA,
    redeemer.toBuffer().toString("hex"),
    Buffer.from(tryNativeToHexString("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", 2), "hex"),
    payload,
    0 // Batch ID
  );
  return guardians.addSignatures(published, [0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 14]);
}
