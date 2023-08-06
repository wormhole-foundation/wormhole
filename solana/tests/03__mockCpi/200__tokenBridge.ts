import {
  CHAIN_ID_SOLANA,
  parseTokenTransferPayload,
  parseVaa,
  tryNativeToHexString,
} from "@certusone/wormhole-sdk";
import { MockGuardians, MockTokenBridge } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import {
  createAssociatedTokenAccount,
  getAccount,
  getAssociatedTokenAddressSync,
  mintTo,
  transfer,
} from "@solana/spl-token";
import { expect } from "chai";
import {
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  GUARDIAN_KEYS,
  MINT_INFO_9,
  WRAPPED_MINT_INFO_8,
  createAssociatedTokenAccountOffCurve,
  expectDeepEqual,
  expectIxOk,
  invokeVerifySignaturesAndPostVaa,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import * as mockCpi from "../helpers/mockCpi";
import * as tokenBridge from "../helpers/tokenBridge";

const GUARDIAN_SET_INDEX = 4;
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

      const tokenBridgeProgram = mockCpi.getTokenBridgeProgram(program);
      const {
        coreBridgeProgram,
        custodyToken: tokenBridgeCustodyToken,
        transferAuthority: tokenBridgeTransferAuthority,
        custodyAuthority: tokenBridgeCustodyAuthority,
        coreBridgeConfig,
        coreEmitter,
        coreEmitterSequence,
        coreFeeCollector,
      } = tokenBridge.legacyTransferTokensNativeAccounts(tokenBridgeProgram, {
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
        tokenBridgeProgram,
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
          coreBridgeProgram,
          tokenBridgeProgram: tokenBridgeProgram.programId,
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
      const tokenBridgeProgram = mockCpi.getTokenBridgeProgram(program);
      const {
        coreBridgeProgram,
        custodyToken: tokenBridgeCustodyToken,
        transferAuthority: tokenBridgeTransferAuthority,
        custodyAuthority: tokenBridgeCustodyAuthority,
        coreBridgeConfig,
        coreEmitter,
        coreEmitterSequence,
        coreFeeCollector,
      } = tokenBridge.legacyTransferTokensWithPayloadNativeAccounts(tokenBridgeProgram, {
        payer: payer.publicKey,
        srcToken,
        mint,
        coreMessage,
        senderAuthority: programSenderAuthority,
      });

      const nonce = 420;
      const amount = new anchor.BN(6942069);
      const redeemer = Array.from(
        Buffer.from("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "hex")
      );
      const redeemerChain = 69;
      const payload = Buffer.from("Where's the beef?");

      const approveIx = tokenBridge.approveTransferAuthorityIx(
        tokenBridgeProgram,
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
          coreBridgeProgram,
          tokenBridgeProgram: tokenBridgeProgram.programId,
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

    it("Invoke `mock_legacy_complete_transfer_native`", async () => {
      const { mint } = MINT_INFO_9;
      const recipient = anchor.web3.Keypair.generate().publicKey;

      const encodedAmount = new anchor.BN(694206);

      // Where's my money, foo?
      const encodedFee = encodedAmount.divn(10);

      const signedVaa = guardians.addSignatures(
        foreignTokenBridge.publishTransferTokens(
          tryNativeToHexString(mint.toString(), "solana"),
          CHAIN_ID_SOLANA,
          BigInt(encodedAmount.toString()),
          CHAIN_ID_SOLANA,
          recipient.toBuffer().toString("hex"),
          BigInt(encodedFee.toString())
        ),
        [0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 14]
      );

      await invokeVerifySignaturesAndPostVaa(
        mockCpi.getCoreBridgeProgram(program),
        payer,
        signedVaa
      );

      const parsed = parseVaa(signedVaa);

      const recipientToken = await createAssociatedTokenAccount(connection, payer, mint, recipient);

      const tokenBridgeProgram = mockCpi.getTokenBridgeProgram(program);

      // For the validator-loaded Token Bridge program, we have not registered Ethereum using the
      // new register chain instruction.
      const {
        coreBridgeProgram,
        payerToken,
        postedVaa,
        claim: tokenBridgeClaim,
        registeredEmitter: tokenBridgeRegisteredEmitter,
        custodyToken: tokenBridgeCustodyToken,
        custodyAuthority: tokenBridgeCustodyAuthority,
      } = tokenBridge.legacyCompleteTransferNativeAccounts(
        tokenBridgeProgram,
        {
          payer: payer.publicKey,
          recipientToken,
          mint,
          recipient,
        },
        parsed,
        {
          legacyRegisteredEmitterDerive: true,
        }
      );

      const ix = await program.methods
        .mockLegacyCompleteTransferNative()
        .accounts({
          payer: payer.publicKey,
          recipientToken,
          recipient,
          payerToken,
          postedVaa,
          tokenBridgeClaim,
          tokenBridgeRegisteredEmitter,
          tokenBridgeCustodyToken,
          mint,
          tokenBridgeCustodyAuthority,
          coreBridgeProgram,
          tokenBridgeProgram: tokenBridgeProgram.programId,
        })
        .instruction();

      const balanceBefore = await getAccount(connection, recipientToken).then(
        (acct) => acct.amount
      );

      await expectIxOk(connection, [ix], [payer]);

      const balanceAfter = await getAccount(connection, recipientToken).then((acct) => acct.amount);

      const expectedBalanceChange = BigInt(encodedAmount.sub(encodedFee).muln(10).toString());
      expect(balanceAfter - balanceBefore).equals(expectedBalanceChange);
    });

    it("Invoke `mock_legacy_complete_transfer_with_payload_native` where Redeemer == Program ID", async () => {
      const { mint } = MINT_INFO_9;

      const encodedAmount = new anchor.BN(694206);
      const payload = Buffer.from("Where's the beef?");
      const signedVaa = getSignedTransferNativeWithPayloadVaa(
        mint,
        encodedAmount,
        program.programId,
        payload
      );

      await invokeVerifySignaturesAndPostVaa(
        mockCpi.getCoreBridgeProgram(program),
        payer,
        signedVaa
      );

      const parsed = parseVaa(signedVaa);

      const dstToken = getAssociatedTokenAddressSync(mint, payer.publicKey);
      const tokenBridgeProgram = mockCpi.getTokenBridgeProgram(program);

      // For the validator-loaded Token Bridge program, we have not registered Ethereum using the
      // new register chain instruction.
      const legacyRegisteredEmitterDerive = true;
      const {
        coreBridgeProgram,
        postedVaa,
        claim: tokenBridgeClaim,
        registeredEmitter: tokenBridgeRegisteredEmitter,
        custodyToken: tokenBridgeCustodyToken,
        custodyAuthority: tokenBridgeCustodyAuthority,
      } = tokenBridge.legacyCompleteTransferWithPayloadNativeAccounts(
        tokenBridgeProgram,
        {
          payer: payer.publicKey,
          dstToken,
          mint,
        },
        parsed,
        legacyRegisteredEmitterDerive
      );

      const programRedeemerAuthority = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("redeemer")],
        program.programId
      )[0];

      const ix = await program.methods
        .mockLegacyCompleteTransferWithPayloadNative()
        .accounts({
          payer: payer.publicKey,
          tokenBridgeProgramRedeemerAuthority: programRedeemerAuthority,
          tokenBridgeCustomRedeemerAuthority: null,
          dstToken,
          postedVaa,
          tokenBridgeClaim,
          tokenBridgeRegisteredEmitter,
          tokenBridgeCustodyToken,
          mint,
          tokenBridgeCustodyAuthority,
          coreBridgeProgram,
          tokenBridgeProgram: tokenBridgeProgram.programId,
        })
        .instruction();

      const balanceBefore = await getAccount(connection, dstToken).then((acct) => acct.amount);

      await expectIxOk(connection, [ix], [payer]);

      const balanceAfter = await getAccount(connection, dstToken).then((acct) => acct.amount);

      const expectedBalanceChange = BigInt(encodedAmount.muln(10).toString());
      expect(balanceAfter - balanceBefore).equals(expectedBalanceChange);
    });

    it("Invoke `mock_legacy_complete_transfer_with_payload_native` where Redeemer != Program ID", async () => {
      const { mint } = MINT_INFO_9;

      const customRedeemerAuthority = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("custom_redeemer_authority")],
        program.programId
      )[0];

      // We need to create a token account for the custom redeemer authority.
      const dstToken = await createAssociatedTokenAccountOffCurve(
        connection,
        payer,
        mint,
        customRedeemerAuthority
      );

      const encodedAmount = new anchor.BN(694206);
      const payload = Buffer.from("Where's the beef?");
      const signedVaa = getSignedTransferNativeWithPayloadVaa(
        mint,
        encodedAmount,
        customRedeemerAuthority,
        payload
      );

      await invokeVerifySignaturesAndPostVaa(
        mockCpi.getCoreBridgeProgram(program),
        payer,
        signedVaa
      );

      const parsed = parseVaa(signedVaa);
      const tokenBridgeProgram = mockCpi.getTokenBridgeProgram(program);

      // For the validator-loaded Token Bridge program, we have not registered Ethereum using the
      // new register chain instruction.
      const legacyRegisteredEmitterDerive = true;
      const {
        coreBridgeProgram,
        postedVaa,
        claim: tokenBridgeClaim,
        registeredEmitter: tokenBridgeRegisteredEmitter,
        custodyToken: tokenBridgeCustodyToken,
        custodyAuthority: tokenBridgeCustodyAuthority,
      } = tokenBridge.legacyCompleteTransferWithPayloadNativeAccounts(
        tokenBridgeProgram,
        {
          payer: payer.publicKey,
          dstToken,
          mint,
        },
        parsed,
        legacyRegisteredEmitterDerive
      );

      const ix = await program.methods
        .mockLegacyCompleteTransferWithPayloadNative()
        .accounts({
          payer: payer.publicKey,
          tokenBridgeProgramRedeemerAuthority: null,
          tokenBridgeCustomRedeemerAuthority: customRedeemerAuthority,
          dstToken,
          postedVaa,
          tokenBridgeClaim,
          tokenBridgeRegisteredEmitter,
          tokenBridgeCustodyToken,
          mint,
          tokenBridgeCustodyAuthority,
          coreBridgeProgram,
          tokenBridgeProgram: tokenBridgeProgram.programId,
        })
        .instruction();
      const balanceBefore = await getAccount(connection, dstToken).then((acct) => acct.amount);

      await expectIxOk(connection, [ix], [payer]);

      const balanceAfter = await getAccount(connection, dstToken).then((acct) => acct.amount);

      const expectedBalanceChange = BigInt(encodedAmount.muln(10).toString());
      expect(balanceAfter - balanceBefore).equals(expectedBalanceChange);
    });

    it("Invoke `mock_legacy_complete_transfer_wrapped`", async () => {
      const { chain, address } = WRAPPED_MINT_INFO_8;
      const recipientSigner = anchor.web3.Keypair.generate();
      const recipient = recipientSigner.publicKey;

      const amount = new anchor.BN(6942069);

      // Where's my money, foo?
      const fee = amount.divn(10);

      const signedVaa = guardians.addSignatures(
        foreignTokenBridge.publishTransferTokens(
          Buffer.from(address).toString("hex"),
          chain,
          BigInt(amount.toString()),
          CHAIN_ID_SOLANA,
          recipient.toBuffer().toString("hex"),
          BigInt(fee.toString())
        ),
        [0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 14]
      );

      await invokeVerifySignaturesAndPostVaa(
        mockCpi.getCoreBridgeProgram(program),
        payer,
        signedVaa
      );

      const parsed = parseVaa(signedVaa);

      const tokenBridgeProgram = mockCpi.getTokenBridgeProgram(program);
      const wrappedMint = tokenBridge.wrappedMintPda(
        tokenBridgeProgram.programId,
        chain,
        Array.from(address)
      );

      const recipientToken = await createAssociatedTokenAccount(
        connection,
        payer,
        wrappedMint,
        recipient
      );

      // For the validator-loaded Token Bridge program, we have not registered Ethereum using the
      // new register chain instruction.
      const {
        coreBridgeProgram,
        payerToken,
        postedVaa,
        claim: tokenBridgeClaim,
        registeredEmitter: tokenBridgeRegisteredEmitter,
        wrappedAsset: tokenBridgeWrappedAsset,
        mintAuthority: tokenBridgeMintAuthority,
      } = tokenBridge.legacyCompleteTransferWrappedAccounts(
        tokenBridgeProgram,
        {
          payer: payer.publicKey,
          recipientToken,
          recipient,
        },
        parsed,
        {
          legacyRegisteredEmitterDerive: true,
        }
      );

      const ix = await program.methods
        .mockLegacyCompleteTransferWrapped()
        .accounts({
          payer: payer.publicKey,
          recipientToken,
          recipient,
          payerToken,
          postedVaa,
          tokenBridgeClaim,
          tokenBridgeRegisteredEmitter,
          tokenBridgeWrappedMint: wrappedMint,
          tokenBridgeWrappedAsset,
          tokenBridgeMintAuthority,
          coreBridgeProgram,
          tokenBridgeProgram: tokenBridgeProgram.programId,
        })
        .instruction();

      const balanceBefore = await getAccount(connection, recipientToken).then(
        (acct) => acct.amount
      );

      await expectIxOk(connection, [ix], [payer]);

      const balanceAfter = await getAccount(connection, recipientToken).then((acct) => acct.amount);

      const expectedBalanceChange = BigInt(amount.sub(fee).toString());
      expect(balanceAfter - balanceBefore).equals(expectedBalanceChange);

      // Transfer everything to payer.
      await transfer(
        connection,
        payer,
        recipientToken,
        getAssociatedTokenAddressSync(wrappedMint, payer.publicKey),
        recipientSigner,
        balanceAfter
      );
    });

    it("Invoke `mock_legacy_complete_transfer_with_payload_wrapped` where Redeemer == Program ID", async () => {
      const { chain, address } = WRAPPED_MINT_INFO_8;

      const tokenBridgeProgram = mockCpi.getTokenBridgeProgram(program);
      const wrappedMint = tokenBridge.wrappedMintPda(
        tokenBridgeProgram.programId,
        chain,
        Array.from(address)
      );

      const amount = new anchor.BN(6942069);
      const payload = Buffer.from("Where's the beef?");
      const signedVaa = getSignedTransferWrappedWithPayloadVaa(
        address,
        chain,
        amount,
        program.programId,
        payload
      );

      await invokeVerifySignaturesAndPostVaa(
        mockCpi.getCoreBridgeProgram(program),
        payer,
        signedVaa
      );

      const parsed = parseVaa(signedVaa);

      const dstToken = getAssociatedTokenAddressSync(wrappedMint, payer.publicKey);

      // For the validator-loaded Token Bridge program, we have not registered Ethereum using the
      // new register chain instruction.
      const legacyRegisteredEmitterDerive = true;
      const {
        coreBridgeProgram,
        postedVaa,
        claim: tokenBridgeClaim,
        registeredEmitter: tokenBridgeRegisteredEmitter,
        wrappedAsset: tokenBridgeWrappedAsset,
        mintAuthority: tokenBridgeMintAuthority,
      } = tokenBridge.legacyCompleteTransferWithPayloadWrappedAccounts(
        tokenBridgeProgram,
        {
          payer: payer.publicKey,
          dstToken,
        },
        parsed,
        legacyRegisteredEmitterDerive
      );

      const programRedeemerAuthority = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("redeemer")],
        program.programId
      )[0];

      const ix = await program.methods
        .mockLegacyCompleteTransferWithPayloadWrapped()
        .accounts({
          payer: payer.publicKey,
          tokenBridgeProgramRedeemerAuthority: programRedeemerAuthority,
          tokenBridgeCustomRedeemerAuthority: null,
          dstToken,
          postedVaa,
          tokenBridgeClaim,
          tokenBridgeRegisteredEmitter,
          tokenBridgeWrappedMint: wrappedMint,
          tokenBridgeWrappedAsset,
          tokenBridgeMintAuthority,
          coreBridgeProgram,
          tokenBridgeProgram: tokenBridgeProgram.programId,
        })
        .instruction();

      const balanceBefore = await getAccount(connection, dstToken).then((acct) => acct.amount);

      await expectIxOk(connection, [ix], [payer]);

      const balanceAfter = await getAccount(connection, dstToken).then((acct) => acct.amount);

      const expectedBalanceChange = BigInt(amount.toString());
      expect(balanceAfter - balanceBefore).equals(expectedBalanceChange);
    });

    it("Invoke `mock_legacy_complete_transfer_with_payload_wrapped` where Redeemer != Program ID", async () => {
      const { chain, address } = WRAPPED_MINT_INFO_8;

      const tokenBridgeProgram = mockCpi.getTokenBridgeProgram(program);
      const wrappedMint = tokenBridge.wrappedMintPda(
        tokenBridgeProgram.programId,
        chain,
        Array.from(address)
      );

      const customRedeemerAuthority = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("custom_redeemer_authority")],
        program.programId
      )[0];

      // We need to create a token account for the custom redeemer authority.
      const dstToken = await createAssociatedTokenAccountOffCurve(
        connection,
        payer,
        wrappedMint,
        customRedeemerAuthority
      );

      const amount = new anchor.BN(6942069);
      const payload = Buffer.from("Where's the beef?");
      const signedVaa = getSignedTransferWrappedWithPayloadVaa(
        address,
        chain,
        amount,
        customRedeemerAuthority,
        payload
      );

      await invokeVerifySignaturesAndPostVaa(
        mockCpi.getCoreBridgeProgram(program),
        payer,
        signedVaa
      );

      const parsed = parseVaa(signedVaa);

      // For the validator-loaded Token Bridge program, we have not registered Ethereum using the
      // new register chain instruction.
      const legacyRegisteredEmitterDerive = true;
      const {
        coreBridgeProgram,
        postedVaa,
        claim: tokenBridgeClaim,
        registeredEmitter: tokenBridgeRegisteredEmitter,
        wrappedAsset: tokenBridgeWrappedAsset,
        mintAuthority: tokenBridgeMintAuthority,
      } = tokenBridge.legacyCompleteTransferWithPayloadWrappedAccounts(
        tokenBridgeProgram,
        {
          payer: payer.publicKey,
          dstToken,
        },
        parsed,
        legacyRegisteredEmitterDerive
      );

      const ix = await program.methods
        .mockLegacyCompleteTransferWithPayloadWrapped()
        .accounts({
          payer: payer.publicKey,
          tokenBridgeProgramRedeemerAuthority: null,
          tokenBridgeCustomRedeemerAuthority: customRedeemerAuthority,
          dstToken,
          postedVaa,
          tokenBridgeClaim,
          tokenBridgeRegisteredEmitter,
          tokenBridgeWrappedMint: wrappedMint,
          tokenBridgeWrappedAsset,
          tokenBridgeMintAuthority,
          coreBridgeProgram,
          tokenBridgeProgram: tokenBridgeProgram.programId,
        })
        .instruction();

      const balanceBefore = await getAccount(connection, dstToken).then((acct) => acct.amount);

      await expectIxOk(connection, [ix], [payer]);

      const balanceAfter = await getAccount(connection, dstToken).then((acct) => acct.amount);

      const expectedBalanceChange = BigInt(amount.toString());
      expect(balanceAfter - balanceBefore).equals(expectedBalanceChange);

      // Give the monies to the payer.
      let withdrawIx = await program.methods
        .withdrawBalance()
        .accounts({
          customRedeemerAuthority,
          programToken: dstToken,
          dstToken: getAssociatedTokenAddressSync(wrappedMint, payer.publicKey),
        })
        .instruction();
      await expectIxOk(connection, [withdrawIx], [payer]);
    });

    it("Invoke `mock_legacy_transfer_tokens_wrapped`", async () => {
      const { chain, address } = WRAPPED_MINT_INFO_8;

      const tokenBridgeProgram = mockCpi.getTokenBridgeProgram(program);
      const wrappedMint = tokenBridge.wrappedMintPda(
        tokenBridgeProgram.programId,
        chain,
        Array.from(address)
      );
      const srcToken = getAssociatedTokenAddressSync(wrappedMint, payer.publicKey);

      const { payerSequence, coreMessage } = await getPayerSequenceAndMessage(
        program,
        payer.publicKey
      );

      const {
        coreBridgeProgram,
        wrappedAsset: tokenBridgeWrappedAsset,
        transferAuthority: tokenBridgeTransferAuthority,
        coreBridgeConfig,
        coreEmitter,
        coreEmitterSequence,
        coreFeeCollector,
      } = tokenBridge.legacyTransferTokensWrappedAccounts(tokenBridgeProgram, {
        payer: payer.publicKey,
        srcToken,
        wrappedMint,
        coreMessage,
      });

      const nonce = 420;
      const amount = new anchor.BN(6942069);
      const recipient = Array.from(
        Buffer.from("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "hex")
      );
      const recipientChain = 69;

      const approveIx = tokenBridge.approveTransferAuthorityIx(
        tokenBridgeProgram,
        srcToken,
        payer.publicKey,
        amount
      );

      const ix = await program.methods
        .mockLegacyTransferTokensWrapped({
          nonce,
          amount,
          recipient,
          recipientChain,
        })
        .accounts({
          payer: payer.publicKey,
          payerSequence,
          srcToken,
          tokenBridgeWrappedMint: wrappedMint,
          tokenBridgeWrappedAsset,
          tokenBridgeTransferAuthority,
          coreBridgeConfig,
          coreMessage,
          coreEmitter,
          coreEmitterSequence,
          coreFeeCollector,
          coreBridgeProgram,
          tokenBridgeProgram: tokenBridgeProgram.programId,
        })
        .instruction();

      const balanceBefore = await getAccount(connection, srcToken).then((acct) => acct.amount);

      await expectIxOk(connection, [approveIx, ix], [payer]);

      const balanceAfter = await getAccount(connection, srcToken).then((acct) => acct.amount);

      const expectedBalanceChange = BigInt(amount.toString());
      expect(balanceBefore - balanceAfter).equals(expectedBalanceChange);
    });

    it("Invoke `mock_legacy_transfer_tokens_with_payload_wrapped` where Sender == Program ID", async () => {
      const { chain, address } = WRAPPED_MINT_INFO_8;

      const tokenBridgeProgram = mockCpi.getTokenBridgeProgram(program);
      const wrappedMint = tokenBridge.wrappedMintPda(
        tokenBridgeProgram.programId,
        chain,
        Array.from(address)
      );
      const srcToken = getAssociatedTokenAddressSync(wrappedMint, payer.publicKey);

      const { payerSequence, coreMessage } = await getPayerSequenceAndMessage(
        program,
        payer.publicKey
      );

      const programSenderAuthority = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("sender")],
        program.programId
      )[0];

      const {
        coreBridgeProgram,
        wrappedAsset: tokenBridgeWrappedAsset,
        transferAuthority: tokenBridgeTransferAuthority,
        coreBridgeConfig,
        coreEmitter,
        coreEmitterSequence,
        coreFeeCollector,
      } = tokenBridge.legacyTransferTokensWithPayloadWrappedAccounts(tokenBridgeProgram, {
        payer: payer.publicKey,
        srcToken,
        wrappedMint,
        coreMessage,
        senderAuthority: programSenderAuthority,
      });

      const nonce = 420;
      const amount = new anchor.BN(6942069);
      const redeemer = Array.from(
        Buffer.from("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "hex")
      );
      const redeemerChain = 69;
      const payload = Buffer.from("Where's the beef?");

      const approveIx = tokenBridge.approveTransferAuthorityIx(
        tokenBridgeProgram,
        srcToken,
        payer.publicKey,
        amount
      );

      const ix = await program.methods
        .mockLegacyTransferTokensWithPayloadWrapped({
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
          tokenBridgeWrappedMint: wrappedMint,
          tokenBridgeWrappedAsset,
          tokenBridgeTransferAuthority,
          coreBridgeConfig,
          coreMessage,
          coreEmitter,
          coreEmitterSequence,
          coreFeeCollector,
          coreBridgeProgram,
          tokenBridgeProgram: tokenBridgeProgram.programId,
        })
        .instruction();

      const balanceBefore = await getAccount(connection, srcToken).then((acct) => acct.amount);

      await expectIxOk(connection, [approveIx, ix], [payer]);

      const balanceAfter = await getAccount(connection, srcToken).then((acct) => acct.amount);

      const expectedBalanceChange = BigInt(amount.toString());
      expect(balanceBefore - balanceAfter).equals(expectedBalanceChange);

      const transferMsg = await coreBridge.PostedMessageV1.fromAccountAddress(
        connection,
        coreMessage
      ).then((msg) => parseTokenTransferPayload(msg.payload));
      expectDeepEqual(new anchor.web3.PublicKey(transferMsg.fromAddress), program.programId);
    });

    it("Invoke `mock_legacy_transfer_tokens_with_payload_wrapped` where Sender != Program ID", async () => {
      const { chain, address } = WRAPPED_MINT_INFO_8;

      const tokenBridgeProgram = mockCpi.getTokenBridgeProgram(program);
      const wrappedMint = tokenBridge.wrappedMintPda(
        tokenBridgeProgram.programId,
        chain,
        Array.from(address)
      );
      const srcToken = getAssociatedTokenAddressSync(wrappedMint, payer.publicKey);

      const { payerSequence, coreMessage } = await getPayerSequenceAndMessage(
        program,
        payer.publicKey
      );

      const customSenderAuthority = anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from("custom_sender_authority")],
        program.programId
      )[0];

      const {
        coreBridgeProgram,
        wrappedAsset: tokenBridgeWrappedAsset,
        transferAuthority: tokenBridgeTransferAuthority,
        coreBridgeConfig,
        coreEmitter,
        coreEmitterSequence,
        coreFeeCollector,
      } = tokenBridge.legacyTransferTokensWithPayloadWrappedAccounts(tokenBridgeProgram, {
        payer: payer.publicKey,
        srcToken,
        wrappedMint,
        coreMessage,
        senderAuthority: customSenderAuthority,
      });

      const nonce = 420;
      const amount = new anchor.BN(6942069);
      const redeemer = Array.from(
        Buffer.from("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "hex")
      );
      const redeemerChain = 69;
      const payload = Buffer.from("Where's the beef?");

      const approveIx = tokenBridge.approveTransferAuthorityIx(
        tokenBridgeProgram,
        srcToken,
        payer.publicKey,
        amount
      );

      const ix = await program.methods
        .mockLegacyTransferTokensWithPayloadWrapped({
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
          tokenBridgeWrappedMint: wrappedMint,
          tokenBridgeWrappedAsset,
          tokenBridgeTransferAuthority,
          coreBridgeConfig,
          coreMessage,
          coreEmitter,
          coreEmitterSequence,
          coreFeeCollector,
          coreBridgeProgram,
          tokenBridgeProgram: tokenBridgeProgram.programId,
        })
        .instruction();

      const balanceBefore = await getAccount(connection, srcToken).then((acct) => acct.amount);

      await expectIxOk(connection, [approveIx, ix], [payer]);

      const balanceAfter = await getAccount(connection, srcToken).then((acct) => acct.amount);

      const expectedBalanceChange = BigInt(amount.toString());
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
    [Buffer.from("my_message"), payer.toBuffer(), payerSequenceValue.toBuffer("le", 16)],
    program.programId
  )[0];

  return { payerSequence, coreMessage };
}

function getSignedTransferNativeWithPayloadVaa(
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
    0 // nonce
  );
  return guardians.addSignatures(published, [0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 14]);
}

function getSignedTransferWrappedWithPayloadVaa(
  tokenAddress: Uint8Array,
  tokenChain: number,
  amount: anchor.BN,
  recipient: anchor.web3.PublicKey,
  payload: Buffer,
  targetChain?: number
): Buffer {
  const vaaBytes = foreignTokenBridge.publishTransferTokensWithPayload(
    Buffer.from(tokenAddress).toString("hex"),
    tokenChain,
    BigInt(amount.toString()),
    targetChain ?? CHAIN_ID_SOLANA,
    recipient.toBuffer().toString("hex"), // TARGET CONTRACT (redeemer)
    Buffer.from(tryNativeToHexString("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", 2), "hex"),
    payload,
    0 // nonce
  );
  return guardians.addSignatures(vaaBytes, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}
