import * as anchor from "@coral-xyz/anchor";
import {
  getAssociatedTokenAddressSync,
  getOrCreateAssociatedTokenAccount,
} from "@solana/spl-token";
import {
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  MINT_INFO_8,
  MINT_INFO_9,
  MintInfo,
  expectIxOkDetails,
  getTokenBalances,
  invokeVerifySignaturesAndPostVaa,
  parallelPostVaa,
  expectIxErr,
  airdrop,
  ETHEREUM_DEADBEEF_TOKEN_ADDRESS,
} from "../helpers";
import {
  CHAIN_ID_SOLANA,
  CHAIN_ID_ETH,
  tryNativeToHexString,
  parseVaa,
  tryNativeToUint8Array,
} from "@certusone/wormhole-sdk";
import { GUARDIAN_KEYS } from "../helpers";
import * as tokenBridge from "../helpers/tokenBridge";
import * as coreBridge from "../helpers/coreBridge";
import { MockTokenBridge, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import { expect } from "chai";

const GUARDIAN_SET_INDEX = 4;
const dummyTokenBridge = new MockTokenBridge(
  tryNativeToHexString(ETHEREUM_TOKEN_BRIDGE_ADDRESS, 2),
  2,
  0, // consistency level
  69 // start sequence
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

describe("Token Bridge -- Legacy Instruction: Complete Transfer With Payload (Native)", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.localnet());
  const wormholeProgram = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  const mints: MintInfo[] = [MINT_INFO_8, MINT_INFO_9];

  describe("Ok", () => {
    for (const { mint, decimals } of mints) {
      it(`Invoke \`complete_transfer_with_payload_native\` (${decimals} Decimals, Redeemer == Redeemer Authority)`, async () => {
        // Create recipient token account.
        const payerToken = getAssociatedTokenAddressSync(mint, payer.publicKey);

        // Amounts.
        const amount = BigInt(699999);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(mint, amount, payer.publicKey, "0xdeadbeef");

        // Fetch balances before.
        const recipientBalancesBefore = await getTokenBalances(program, forkedProgram, payerToken);

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            payer: payer.publicKey,
            dstToken: payerToken,
            mint,
          },
          signedVaa,
          payer
        );

        // Check recipient and relayer token balance changes.
        await tokenBridge.expectCorrectTokenBalanceChanges(
          connection,
          payerToken,
          recipientBalancesBefore,
          tokenBridge.TransferDirection.In
        );
      });

      it(`Invoke \`complete_transfer_with_payload_native\` (${decimals} Decimals, Redeemer != Redeemer Authority)`, async () => {
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const dstToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          recipient.publicKey
        );

        // Amounts.
        const amount = BigInt(699999);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(mint, amount, recipient.publicKey, "0xdeadbeef");

        // Fetch balances before.
        const recipientBalancesBefore = await getTokenBalances(
          program,
          forkedProgram,
          dstToken.address
        );

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            payer: payer.publicKey,
            dstToken: dstToken.address,
            redeemerAuthority: recipient.publicKey,
            mint,
          },
          signedVaa,
          payer,
          recipient
        );

        // Check recipient and relayer token balance changes.
        await tokenBridge.expectCorrectTokenBalanceChanges(
          connection,
          dstToken.address,
          recipientBalancesBefore,
          tokenBridge.TransferDirection.In
        );
      });
    }
  });

  describe("New Implementation", () => {
    for (const { mint, decimals } of mints) {
      it(`Cannot Invoke \`complete_transfer_with_payload_native\` (${decimals} Decimals, Invalid Mint)`, async () => {
        // Create recipient token account.
        const payerToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          payer.publicKey
        );

        // Amounts.
        const amount = BigInt(699999);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(
          anchor.web3.Keypair.generate().publicKey, // Pass bogus mint
          amount,
          payer.publicKey,
          "0xdeadbeef"
        );

        // Post the VAA.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        // Create the complete transfer with payload instruction.
        const ix = tokenBridge.legacyCompleteTransferWithPayloadNativeIx(
          program,
          {
            payer: payer.publicKey,
            dstToken: payerToken.address,
            mint,
          },
          parseVaa(signedVaa)
        );

        await expectIxErr(connection, [ix], [payer], "InvalidMint");
      });

      it(`Cannot Invoke \`complete_transfer_with_payload_native\` (${decimals} Decimals, Invalid Redeemer Chain)`, async () => {
        // Create recipient token account.
        const payerToken = await getOrCreateAssociatedTokenAccount(
          connection,
          payer,
          mint,
          payer.publicKey
        );

        // Amounts.
        const amount = BigInt(699999);

        // Create the signed transfer VAA.
        const signedVaa = getSignedTransferVaa(
          mint,
          amount,
          payer.publicKey,
          "0xdeadbeef",
          CHAIN_ID_ETH // Pass invalid target chain.
        );

        // Post the VAA.
        await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

        // Create the complete transfer with payload instruction.
        const ix = tokenBridge.legacyCompleteTransferWithPayloadNativeIx(
          program,
          {
            payer: payer.publicKey,
            dstToken: payerToken.address,
            mint,
          },
          parseVaa(signedVaa)
        );

        await expectIxErr(connection, [ix], [payer], "RedeemerChainNotSolana");
      });
    }

    it(`Cannot Invoke \`complete_transfer_with_payload_native\` (Wrapped Mint)`, async () => {
      const mint = mints[0].mint;

      // Create recipient token account.
      const payerToken = await getOrCreateAssociatedTokenAccount(
        connection,
        payer,
        mint,
        payer.publicKey
      );

      // Amounts.
      const amount = BigInt(699999);

      // Create the signed transfer VAA. Pass a token chain that is not Solana.
      const signedVaa = getSignedTransferVaa(
        mint,
        amount,
        payer.publicKey,
        "0xdeadbeef",
        undefined,
        CHAIN_ID_ETH // Specify a token chain that is not Solana.
      );

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      // Create the complete transfer with payload instruction.
      const ix = tokenBridge.legacyCompleteTransferWithPayloadNativeIx(
        program,
        {
          payer: payer.publicKey,
          dstToken: payerToken.address,
          mint,
        },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "WrappedAsset");
    });

    it(`Cannot Invoke \`complete_transfer_with_payload_native\` (Invalid Program Redeemer)`, async () => {
      const mint = mints[0].mint;

      // Create recipient token account.
      const payerToken = await getOrCreateAssociatedTokenAccount(
        connection,
        payer,
        mint,
        payer.publicKey
      );

      // Amounts.
      const amount = BigInt(699999);

      // Create the signed transfer VAA with random "to" (redeemer).
      const signedVaa = getSignedTransferVaa(
        mint,
        amount,
        anchor.web3.Keypair.generate().publicKey, // Create random redeemer.
        "0xdeadbeef"
      );

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      // Create the complete transfer with payload instruction.
      const ix = tokenBridge.legacyCompleteTransferWithPayloadNativeIx(
        program,
        {
          payer: payer.publicKey,
          dstToken: payerToken.address,
          mint,
          redeemerAuthority: payer.publicKey,
        },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "InvalidProgramRedeemer");
    });

    it(`Cannot Invoke \`complete_transfer_with_payload_native\` (Constraint Token Owner)`, async () => {
      const mint = mints[0].mint;

      // Create random token account.
      const invalidTokenAccount = await getOrCreateAssociatedTokenAccount(
        connection,
        payer,
        mint,
        anchor.web3.Keypair.generate().publicKey
      );

      // Amounts.
      const amount = BigInt(699999);

      // Create the signed transfer VAA with random "to" (redeemer).
      const signedVaa = getSignedTransferVaa(mint, amount, payer.publicKey, "0xdeadbeef");

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      // Create the complete transfer with payload instruction.
      const ix = tokenBridge.legacyCompleteTransferWithPayloadNativeIx(
        program,
        {
          payer: payer.publicKey,
          dstToken: invalidTokenAccount.address,
          mint,
          redeemerAuthority: payer.publicKey,
        },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "ConstraintTokenOwner");
    });

    it(`Cannot Invoke \`complete_transfer_with_payload_native\` (Invalid Token Bridge VAA)`, async () => {
      const mint = mints[0].mint;

      // Create random token account.
      const payerToken = await getOrCreateAssociatedTokenAccount(
        connection,
        payer,
        mint,
        payer.publicKey
      );

      // Create a bogus attestation VAA.
      const published = dummyTokenBridge.publishAttestMeta(
        Buffer.from(ETHEREUM_DEADBEEF_TOKEN_ADDRESS).toString("hex"),
        8, // Decimals
        "EVOO", // Symbol.
        "Extra Virgin Olive Oil", // Name.
        420, // Nonce.
        1234567 // Timestamp.
      );
      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 14]
      );

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      // Create the complete transfer with payload instruction.
      const ix = tokenBridge.legacyCompleteTransferWithPayloadNativeIx(
        program,
        {
          payer: payer.publicKey,
          dstToken: payerToken.address,
          mint,
          redeemerAuthority: payer.publicKey,
        },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "InvalidTokenBridgeVaa");
    });
  });
});

function getSignedTransferVaa(
  mint: anchor.web3.PublicKey,
  amount: bigint,
  recipient: anchor.web3.PublicKey,
  payload: string,
  targetChain?: number,
  tokenChain?: number
): Buffer {
  const vaaBytes = dummyTokenBridge.publishTransferTokensWithPayload(
    tryNativeToHexString(mint.toString(), "solana"),
    tokenChain ?? CHAIN_ID_SOLANA,
    amount,
    targetChain ?? CHAIN_ID_SOLANA,
    recipient.toBuffer().toString("hex"), // TARGET CONTRACT (redeemer)
    Buffer.from(tryNativeToUint8Array(ETHEREUM_TOKEN_BRIDGE_ADDRESS, 2)),
    Buffer.from(payload.substring(2), "hex"),
    0 // Batch ID
  );
  return guardians.addSignatures(vaaBytes, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

async function parallelTxDetails(
  program: tokenBridge.TokenBridgeProgram,
  forkedProgram: tokenBridge.TokenBridgeProgram,
  accounts: tokenBridge.LegacyCompleteTransferWithPayloadNativeContext,
  signedVaa: Buffer,
  payer: anchor.web3.Keypair,
  redeemerAuthority?: anchor.web3.Keypair
) {
  const connection = program.provider.connection;

  // Post the VAA.
  await parallelPostVaa(connection, payer, signedVaa);

  // Create instruction.
  const ix = tokenBridge.legacyCompleteTransferWithPayloadNativeIx(
    program,
    accounts,
    parseVaa(signedVaa)
  );
  const forkedIx = tokenBridge.legacyCompleteTransferWithPayloadNativeIx(
    forkedProgram,
    accounts,
    parseVaa(signedVaa)
  );

  let signers = [payer];
  if (redeemerAuthority !== undefined) {
    signers.push(redeemerAuthority);
  }

  return await Promise.all([
    expectIxOkDetails(connection, [ix], signers),
    expectIxOkDetails(connection, [forkedIx], signers),
  ]);
}
